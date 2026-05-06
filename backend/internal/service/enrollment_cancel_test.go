package service

import (
	"context"
	"errors"
	"testing"

	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
	"gorm.io/gorm"
)

type stubEnrollmentRepo struct {
	enrollment                 *domain.Enrollment
	updateFromQueuingResult    bool
	updateFromQueuingErr       error
	lastStatusUpdate           string
	updateStatusErr            error
	lastUpdateStatusEnrollment uint64
}

func (s *stubEnrollmentRepo) Create(enrollment *domain.Enrollment) error { return nil }

func (s *stubEnrollmentRepo) FindByID(id uint64) (*domain.Enrollment, error) {
	if s.enrollment == nil || s.enrollment.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return s.enrollment, nil
}

func (s *stubEnrollmentRepo) FindByUserAndActivity(userID, activityID uint64) (*domain.Enrollment, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubEnrollmentRepo) UpdateStatus(id uint64, status string) error {
	s.lastUpdateStatusEnrollment = id
	s.lastStatusUpdate = status
	return s.updateStatusErr
}

func (s *stubEnrollmentRepo) UpdateStatusFromQueuing(id, userID uint64, status string) (bool, error) {
	if s.updateFromQueuingErr != nil {
		return false, s.updateFromQueuingErr
	}
	s.lastUpdateStatusEnrollment = id
	s.lastStatusUpdate = status
	return s.updateFromQueuingResult, nil
}

func (s *stubEnrollmentRepo) ListByUserID(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error) {
	return nil, 0, nil
}

func (s *stubEnrollmentRepo) ListByActivityID(activityID uint64, status string) ([]domain.Enrollment, error) {
	return nil, nil
}

type stubOrderRepoForCancel struct {
	order                   *domain.Order
	updateFromPendingResult bool
	updateFromPendingErr    error
}

func (s *stubOrderRepoForCancel) Create(order *domain.Order) error { return nil }

func (s *stubOrderRepoForCancel) FindByID(id uint64) (*domain.Order, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubOrderRepoForCancel) FindByOrderNo(orderNo string) (*domain.Order, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubOrderRepoForCancel) FindByEnrollmentID(enrollmentID uint64) (*domain.Order, error) {
	if s.order == nil || s.order.EnrollmentID != enrollmentID {
		return nil, gorm.ErrRecordNotFound
	}
	return s.order, nil
}

func (s *stubOrderRepoForCancel) FindByUserID(userID uint64, page, pageSize int) ([]domain.Order, int64, error) {
	return nil, 0, nil
}

func (s *stubOrderRepoForCancel) UpdateStatus(id uint64, status string) error { return nil }

func (s *stubOrderRepoForCancel) UpdateStatusFromPending(id uint64, status string) (bool, error) {
	if s.updateFromPendingErr != nil {
		return false, s.updateFromPendingErr
	}
	return s.updateFromPendingResult, nil
}

func (s *stubOrderRepoForCancel) ListExpired() ([]domain.Order, error) { return nil, nil }

type stubActivityRepoForCancel struct {
	incrementStockCalled int
}

func (s *stubActivityRepoForCancel) Create(activity *domain.Activity) error { return nil }

func (s *stubActivityRepoForCancel) FindByID(id uint64) (*domain.Activity, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubActivityRepoForCancel) Update(activity *domain.Activity) error { return nil }

func (s *stubActivityRepoForCancel) Delete(id uint64) error { return nil }

func (s *stubActivityRepoForCancel) List(filter repository.ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error) {
	return nil, 0, nil
}

func (s *stubActivityRepoForCancel) PublishedList(page, pageSize int) ([]domain.Activity, int64, error) {
	return nil, 0, nil
}

func (s *stubActivityRepoForCancel) FindByMerchantID(merchantID uint64) ([]domain.Activity, error) {
	return nil, nil
}

func (s *stubActivityRepoForCancel) DeductStock(activityID uint64) (int64, error) {
	return 0, nil
}

func (s *stubActivityRepoForCancel) IncrementStock(activityID uint64) error {
	s.incrementStockCalled++
	return nil
}

type stubStockEngineForCancel struct {
	rollbackCalled int
}

func (s *stubStockEngineForCancel) TryEnroll(ctx context.Context, activityID, userID uint64) (int64, error) {
	return 0, errors.New("not used")
}

func (s *stubStockEngineForCancel) GetStock(ctx context.Context, activityID uint64) (int, error) {
	return 0, errors.New("not used")
}

func (s *stubStockEngineForCancel) Rollback(ctx context.Context, activityID, userID uint64) error {
	s.rollbackCalled++
	return nil
}

func (s *stubStockEngineForCancel) WarmUp(ctx context.Context, activityID uint64, stock int) error {
	return nil
}

func TestEnrollmentCancel_Queuing(t *testing.T) {
	enrollRepo := &stubEnrollmentRepo{
		enrollment: &domain.Enrollment{ID: 1, UserID: 10, ActivityID: 99, Status: "QUEUING"},
		updateFromQueuingResult: true,
	}
	orderRepo := &stubOrderRepoForCancel{}
	activityRepo := &stubActivityRepoForCancel{}
	stockEngine := &stubStockEngineForCancel{}

	svc := NewEnrollmentService(nil, stockEngine, nil, enrollRepo, activityRepo, orderRepo)
	if err := svc.Cancel(context.Background(), 1, 10); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if enrollRepo.lastStatusUpdate != "CANCELLED" {
		t.Fatalf("want CANCELLED, got %s", enrollRepo.lastStatusUpdate)
	}
	if stockEngine.rollbackCalled != 1 {
		t.Fatalf("rollback should be called once")
	}
}

func TestEnrollmentCancel_SuccessPendingOrder(t *testing.T) {
	enrollRepo := &stubEnrollmentRepo{
		enrollment: &domain.Enrollment{ID: 2, UserID: 11, ActivityID: 88, Status: "SUCCESS"},
	}
	orderRepo := &stubOrderRepoForCancel{
		order:                   &domain.Order{ID: 5, EnrollmentID: 2, UserID: 11, ActivityID: 88, Status: "PENDING"},
		updateFromPendingResult: true,
	}
	activityRepo := &stubActivityRepoForCancel{}
	stockEngine := &stubStockEngineForCancel{}

	svc := NewEnrollmentService(nil, stockEngine, nil, enrollRepo, activityRepo, orderRepo)
	if err := svc.Cancel(context.Background(), 2, 11); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if enrollRepo.lastStatusUpdate != "CANCELLED" {
		t.Fatalf("want CANCELLED, got %s", enrollRepo.lastStatusUpdate)
	}
	if activityRepo.incrementStockCalled != 1 {
		t.Fatalf("expected increment stock called once")
	}
	if stockEngine.rollbackCalled != 1 {
		t.Fatalf("rollback should be called once")
	}
}

func TestEnrollmentCancel_SuccessNotPending(t *testing.T) {
	enrollRepo := &stubEnrollmentRepo{
		enrollment: &domain.Enrollment{ID: 3, UserID: 12, ActivityID: 77, Status: "SUCCESS"},
	}
	orderRepo := &stubOrderRepoForCancel{
		order: &domain.Order{ID: 6, EnrollmentID: 3, UserID: 12, ActivityID: 77, Status: "PAID"},
	}
	activityRepo := &stubActivityRepoForCancel{}
	stockEngine := &stubStockEngineForCancel{}

	svc := NewEnrollmentService(nil, stockEngine, nil, enrollRepo, activityRepo, orderRepo)
	if err := svc.Cancel(context.Background(), 3, 12); !errors.Is(err, ErrEnrollNotCancelable) {
		t.Fatalf("want ErrEnrollNotCancelable, got %v", err)
	}
}

func TestEnrollmentCancel_IdempotentCancelled(t *testing.T) {
	enrollRepo := &stubEnrollmentRepo{
		enrollment: &domain.Enrollment{ID: 4, UserID: 13, ActivityID: 66, Status: "CANCELLED"},
	}
	orderRepo := &stubOrderRepoForCancel{}
	activityRepo := &stubActivityRepoForCancel{}
	stockEngine := &stubStockEngineForCancel{}

	svc := NewEnrollmentService(nil, stockEngine, nil, enrollRepo, activityRepo, orderRepo)
	if err := svc.Cancel(context.Background(), 4, 13); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}
