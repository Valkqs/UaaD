package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/middleware"
	"github.com/uaad/backend/internal/service"
	"gorm.io/gorm"
)

// EnrollmentWorker consumes enrollment messages from Kafka and persists
// them to MySQL. On transaction failure it compensates Redis via StockEngine.
type EnrollmentWorker struct {
	reader       *kafka.Reader
	db           *gorm.DB
	stockEngine  service.StockEngine
	notifSvc     service.NotificationService
	activityRepo interface {
		FindByID(id uint64) (*domain.Activity, error)
	}
}

// NewEnrollmentWorker creates a new worker.
func NewEnrollmentWorker(
	reader *kafka.Reader,
	db *gorm.DB,
	stockEngine service.StockEngine,
	notifSvc service.NotificationService,
	activityRepo interface {
		FindByID(id uint64) (*domain.Activity, error)
	},
) *EnrollmentWorker {
	return &EnrollmentWorker{
		reader:       reader,
		db:           db,
		stockEngine:  stockEngine,
		notifSvc:     notifSvc,
		activityRepo: activityRepo,
	}
}

// Run starts the consume loop. It blocks until ctx is cancelled.
func (w *EnrollmentWorker) Run(ctx context.Context) {
	log.Println("[EnrollWorker] started")
	for {
		msg, err := w.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[EnrollWorker] context cancelled, stopping")
				return
			}
			log.Printf("[EnrollWorker] read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		stats := w.reader.Stats()
		middleware.SetWorkerKafkaLag(stats.Topic, stats.Lag)
		w.handleMessage(ctx, msg)
	}
}

func (w *EnrollmentWorker) handleMessage(ctx context.Context, msg kafka.Message) {
	var em service.EnrollmentMessage
	if err := json.Unmarshal(msg.Value, &em); err != nil {
		log.Printf("[EnrollWorker] unmarshal error: %v, payload: %s", err, string(msg.Value))
		return
	}

	start := time.Now()
	now := time.Now()
	queuePos := int(em.QueuePos)
	updated := false

	err := w.db.Transaction(func(tx *gorm.DB) error {
		var current domain.Enrollment
		if err := tx.First(&current, em.EnrollmentID).Error; err != nil {
			return err
		}
		if current.Status != "QUEUING" {
			return nil
		}
		res := tx.Model(&domain.Enrollment{}).
			Where("id = ? AND status = ?", em.EnrollmentID, "QUEUING").
			Updates(map[string]interface{}{
				"status":         "SUCCESS",
				"queue_position": queuePos,
				"finalized_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		updated = true

		order := domain.Order{
			OrderNo:      service.GenerateOrderNo(),
			EnrollmentID: em.EnrollmentID,
			UserID:       em.UserID,
			ActivityID:   em.ActivityID,
			Amount:       em.Price,
			Status:       "PENDING",
			ExpiredAt:    now.Add(15 * time.Minute),
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		if err := tx.Model(&domain.Activity{}).
			Where("id = ? AND enroll_count < max_capacity", em.ActivityID).
			UpdateColumn("enroll_count", gorm.Expr("enroll_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		activityTitle := "unknown"
		if act, e := w.activityRepo.FindByID(em.ActivityID); e == nil {
			activityTitle = act.Title
		}
		log.Printf("[EnrollWorker] MySQL tx failed for user=%d activity=%d: %v — rolling back Redis", em.UserID, em.ActivityID, err)
		if rbErr := w.stockEngine.Rollback(ctx, em.ActivityID, em.UserID); rbErr != nil {
			log.Printf("[EnrollWorker] CRITICAL: Redis rollback also failed: %v", rbErr)
		}
		w.notifSvc.NotifyEnrollFail(em.UserID, 0, activityTitle)
		middleware.RecordWorkerMessage("failure", time.Since(start).Seconds())
		return
	}
	if !updated {
		log.Printf("[EnrollWorker] skip message: enrollment=%d status not QUEUING", em.EnrollmentID)
		middleware.RecordWorkerMessage("success", time.Since(start).Seconds())
		return
	}

	activityTitle := "unknown"
	if act, e := w.activityRepo.FindByID(em.ActivityID); e == nil {
		activityTitle = act.Title
	}
	w.notifSvc.NotifyEnrollSuccess(em.UserID, em.EnrollmentID, activityTitle)
	middleware.RecordWorkerMessage("success", time.Since(start).Seconds())
}
