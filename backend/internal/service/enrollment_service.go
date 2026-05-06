package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrAlreadyEnrolled     = errors.New("already enrolled for this activity")
	ErrStockInsufficient   = errors.New("stock insufficient")
	ErrEnrollmentClosed    = errors.New("enrollment window is closed")
	ErrEnrollNotFound      = errors.New("enrollment not found")
	ErrEnrollNotCancelable = errors.New("enrollment cannot be cancelled")
)

var orderSeq atomic.Int64

// GenerateOrderNo produces a collision-free order number:
// ORD + YYYYMMDD + 14-digit monotonic nanos + 4-digit random suffix.
// Total 29 chars, unique across process restarts and concurrent goroutines.
func GenerateOrderNo() string {
	seq := orderSeq.Add(1)
	now := time.Now()
	var rnd [2]byte
	rand.Read(rnd[:])
	return fmt.Sprintf("ORD%s%06d%04d",
		now.Format("20060102"),
		(now.UnixNano()/1000)%1_000_000+seq,
		int(rnd[0])<<8|int(rnd[1])%10000,
	)
}

// EnrollResult holds the data returned after a successful enrollment request.
// In async mode the enrollment is QUEUING; the Worker will finalize it.
type EnrollResult struct {
	EnrollmentID  uint64 `json:"enrollment_id,omitempty"`
	Status        string `json:"status"`
	OrderNo       string `json:"order_no,omitempty"`
	QueuePosition int64  `json:"queue_position"`
}

// EnrollmentService defines business logic for enrollments.
type EnrollmentService interface {
	Create(userID, activityID uint64) (*EnrollResult, error)
	GetStatus(enrollmentID, userID uint64) (*domain.Enrollment, *domain.Activity, *domain.Order, error)
	ListByUser(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error)
	Cancel(ctx context.Context, enrollmentID, userID uint64) error
}

type enrollmentService struct {
	db           *gorm.DB
	stockEngine  StockEngine
	kafkaWriter  *kafka.Writer
	enrollRepo   repository.EnrollmentRepository
	activityRepo repository.ActivityRepository
	orderRepo    repository.OrderRepository
}

// NewEnrollmentService creates a new EnrollmentService.
func NewEnrollmentService(
	db *gorm.DB,
	stockEngine StockEngine,
	kafkaWriter *kafka.Writer,
	enrollRepo repository.EnrollmentRepository,
	activityRepo repository.ActivityRepository,
	orderRepo repository.OrderRepository,
) EnrollmentService {
	return &enrollmentService{
		db:           db,
		stockEngine:  stockEngine,
		kafkaWriter:  kafkaWriter,
		enrollRepo:   enrollRepo,
		activityRepo: activityRepo,
		orderRepo:    orderRepo,
	}
}

func (s *enrollmentService) Create(userID, activityID uint64) (*EnrollResult, error) {
	// 1. Pre-flight: activity must exist, be published, and within enrollment window
	activity, err := s.activityRepo.FindByID(activityID)
	if err != nil {
		return nil, ErrActivityNotFound
	}
	if activity.Status != "PUBLISHED" && activity.Status != "SELLING_OUT" {
		return nil, ErrEnrollmentClosed
	}
	now := time.Now()
	if now.Before(activity.EnrollOpenAt) || now.After(activity.EnrollCloseAt) {
		return nil, ErrEnrollmentClosed
	}

	// 2. Redis Lua atomic: idempotency + stock deduction + queue number
	ctx := context.Background()
	queuePos, err := s.stockEngine.TryEnroll(ctx, activityID, userID)
	if err != nil {
		if errors.Is(err, ErrAlreadyEnrolledRedis) {
			return nil, ErrAlreadyEnrolled
		}
		if errors.Is(err, ErrStockDepleted) {
			return nil, ErrStockInsufficient
		}
		return nil, err
	}

	queuePosInt := int(queuePos)
	enrollment := &domain.Enrollment{
		UserID:        userID,
		ActivityID:    activityID,
		Status:        "QUEUING",
		QueuePosition: &queuePosInt,
		EnrolledAt:    now,
	}
	if err := s.enrollRepo.Create(enrollment); err != nil {
		s.rollbackRedis(ctx, activityID, userID)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrAlreadyEnrolled
		}
		return nil, err
	}

	// 3. Produce to Kafka — Worker will handle MySQL persistence
	msg := EnrollmentMessage{
		EnrollmentID: enrollment.ID,
		UserID:       userID,
		ActivityID:   activityID,
		QueuePos:     queuePos,
		Price:        activity.Price,
		Timestamp:    now.UnixMilli(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		s.rollbackRedis(ctx, activityID, userID)
		_ = s.enrollRepo.UpdateStatus(enrollment.ID, "FAILED")
		return nil, fmt.Errorf("marshal enrollment message: %w", err)
	}

	err = s.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("%d:%d", activityID, userID)),
		Value: payload,
	})
	if err != nil {
		s.rollbackRedis(ctx, activityID, userID)
		_ = s.enrollRepo.UpdateStatus(enrollment.ID, "FAILED")
		return nil, fmt.Errorf("kafka produce: %w", err)
	}

	return &EnrollResult{
		EnrollmentID:  enrollment.ID,
		Status:        "QUEUING",
		QueuePosition: queuePos,
	}, nil
}

func (s *enrollmentService) rollbackRedis(ctx context.Context, activityID, userID uint64) {
	if rbErr := s.stockEngine.Rollback(ctx, activityID, userID); rbErr != nil {
		log.Printf("[Enrollment] CRITICAL: Redis rollback failed for activity=%d user=%d: %v", activityID, userID, rbErr)
	}
}

func (s *enrollmentService) GetStatus(enrollmentID, userID uint64) (*domain.Enrollment, *domain.Activity, *domain.Order, error) {
	enrollment, err := s.enrollRepo.FindByID(enrollmentID)
	if err != nil {
		return nil, nil, nil, ErrEnrollNotFound
	}
	if enrollment.UserID != userID {
		return nil, nil, nil, ErrEnrollNotFound
	}

	activity, _ := s.activityRepo.FindByID(enrollment.ActivityID)

	var order *domain.Order
	orders, _, _ := s.orderRepo.FindByUserID(userID, 1, 100)
	for i := range orders {
		if orders[i].EnrollmentID == enrollmentID {
			order = &orders[i]
			break
		}
	}

	return enrollment, activity, order, nil
}

func (s *enrollmentService) ListByUser(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.enrollRepo.ListByUserID(userID, page, pageSize)
}

func (s *enrollmentService) Cancel(ctx context.Context, enrollmentID, userID uint64) error {
	enrollment, err := s.enrollRepo.FindByID(enrollmentID)
	if err != nil {
		return ErrEnrollNotFound
	}
	if enrollment.UserID != userID {
		return ErrEnrollNotFound
	}

	switch enrollment.Status {
	case "CANCELLED":
		return nil
	case "FAILED":
		return ErrEnrollNotCancelable
	case "QUEUING":
		updated, err := s.enrollRepo.UpdateStatusFromQueuing(enrollmentID, userID, "CANCELLED")
		if err != nil {
			return err
		}
		if !updated {
			return ErrEnrollNotCancelable
		}
		if err := s.stockEngine.Rollback(ctx, enrollment.ActivityID, userID); err != nil {
			log.Printf("[EnrollmentCancel] Redis rollback failed: enrollment=%d activity=%d user=%d: %v",
				enrollmentID, enrollment.ActivityID, userID, err)
		}
		return nil
	case "SUCCESS":
		order, err := s.orderRepo.FindByEnrollmentID(enrollmentID)
		if err != nil {
			return ErrEnrollNotCancelable
		}
		if order.UserID != userID {
			return ErrEnrollNotFound
		}
		if order.Status != "PENDING" {
			return ErrEnrollNotCancelable
		}
		updated, err := s.orderRepo.UpdateStatusFromPending(order.ID, "CLOSED")
		if err != nil {
			return err
		}
		if !updated {
			return ErrEnrollNotCancelable
		}
		_ = s.activityRepo.IncrementStock(order.ActivityID)
		if err := s.stockEngine.Rollback(ctx, order.ActivityID, userID); err != nil {
			log.Printf("[EnrollmentCancel] Redis rollback failed: order=%d activity=%d user=%d: %v",
				order.ID, order.ActivityID, userID, err)
		}
		if err := s.enrollRepo.UpdateStatus(enrollmentID, "CANCELLED"); err != nil {
			return err
		}
		return nil
	default:
		return ErrEnrollNotCancelable
	}
}
