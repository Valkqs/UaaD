package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
	"gorm.io/gorm"
)

type stubOrderRepo struct {
	order                    *domain.Order
	listExpired              []domain.Order
	updateFromPendingResult  bool
	updateFromPendingErr     error
	updateFromPendingStatus  string
	updateFromPendingOrderID uint64
}

func (s *stubOrderRepo) Create(order *domain.Order) error { return nil }

func (s *stubOrderRepo) FindByID(id uint64) (*domain.Order, error) {
	if s.order == nil || s.order.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return s.order, nil
}

func (s *stubOrderRepo) FindByOrderNo(orderNo string) (*domain.Order, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubOrderRepo) FindByEnrollmentID(enrollmentID uint64) (*domain.Order, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubOrderRepo) FindByUserID(userID uint64, page, pageSize int) ([]domain.Order, int64, error) {
	return nil, 0, nil
}

func (s *stubOrderRepo) UpdateStatus(id uint64, status string) error { return nil }

func (s *stubOrderRepo) UpdateStatusFromPending(id uint64, status string) (bool, error) {
	s.updateFromPendingOrderID = id
	s.updateFromPendingStatus = status
	if s.updateFromPendingErr != nil {
		return false, s.updateFromPendingErr
	}
	return s.updateFromPendingResult, nil
}

func (s *stubOrderRepo) ListExpired() ([]domain.Order, error) {
	return s.listExpired, nil
}

type stubActivityRepo struct {
	activity             *domain.Activity
	incrementStockCalled int
}

func (s *stubActivityRepo) Create(activity *domain.Activity) error { return nil }

func (s *stubActivityRepo) FindByID(id uint64) (*domain.Activity, error) {
	if s.activity == nil || s.activity.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return s.activity, nil
}

func (s *stubActivityRepo) Update(activity *domain.Activity) error { return nil }

func (s *stubActivityRepo) Delete(id uint64) error { return nil }

func (s *stubActivityRepo) List(filter repository.ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error) {
	return nil, 0, nil
}

func (s *stubActivityRepo) PublishedList(page, pageSize int) ([]domain.Activity, int64, error) {
	return nil, 0, nil
}

func (s *stubActivityRepo) FindByMerchantID(merchantID uint64) ([]domain.Activity, error) {
	return nil, nil
}

func (s *stubActivityRepo) DeductStock(activityID uint64) (int64, error) {
	return 0, nil
}

func (s *stubActivityRepo) IncrementStock(activityID uint64) error {
	s.incrementStockCalled++
	return nil
}

type stubStockEngine struct {
	rollbackCalled int
}

func (s *stubStockEngine) TryEnroll(ctx context.Context, activityID, userID uint64) (int64, error) {
	return 0, errors.New("not used")
}

func (s *stubStockEngine) GetStock(ctx context.Context, activityID uint64) (int, error) {
	return 0, errors.New("not used")
}

func (s *stubStockEngine) Rollback(ctx context.Context, activityID, userID uint64) error {
	s.rollbackCalled++
	return nil
}

func (s *stubStockEngine) WarmUp(ctx context.Context, activityID uint64, stock int) error {
	return nil
}

type stubNotificationService struct {
	expireNotified int
	lastOrderID    uint64
}

func (s *stubNotificationService) List(userID uint64, page, pageSize int) ([]domain.Notification, int64, error) {
	return nil, 0, nil
}

func (s *stubNotificationService) MarkRead(notificationID, userID uint64) error { return nil }

func (s *stubNotificationService) UnreadCount(userID uint64) (int64, error) { return 0, nil }

func (s *stubNotificationService) NotifyEnrollSuccess(userID, enrollmentID uint64, activityTitle string) {
}

func (s *stubNotificationService) NotifyEnrollFail(userID, enrollmentID uint64, activityTitle string) {
}

func (s *stubNotificationService) NotifyOrderExpire(userID, orderID uint64, activityTitle string) {
	s.expireNotified++
	s.lastOrderID = orderID
}

func (s *stubNotificationService) NotifyActivityReminder(userID, activityID uint64, activityTitle string) {
}

func TestOrderService_Pay_OptimisticUpdate(t *testing.T) {
	now := time.Now()
	repo := &stubOrderRepo{
		order:                   &domain.Order{ID: 1, UserID: 10, Status: "PENDING", ExpiredAt: now.Add(1 * time.Hour), OrderNo: "ORD1"},
		updateFromPendingResult: true,
	}
	actRepo := &stubActivityRepo{}
	stock := &stubStockEngine{}
	notify := &stubNotificationService{}

	svc := NewOrderService(repo, actRepo, stock, notify)
	res, err := svc.Pay(1, 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != "PAID" {
		t.Fatalf("want PAID, got %s", res.Status)
	}
	if repo.updateFromPendingStatus != "PAID" {
		t.Fatalf("want update status PAID, got %s", repo.updateFromPendingStatus)
	}
}

func TestOrderService_Pay_OptimisticConflict(t *testing.T) {
	now := time.Now()
	repo := &stubOrderRepo{
		order:                   &domain.Order{ID: 1, UserID: 10, Status: "PENDING", ExpiredAt: now.Add(1 * time.Hour), OrderNo: "ORD1"},
		updateFromPendingResult: false,
	}
	actRepo := &stubActivityRepo{}
	stock := &stubStockEngine{}
	notify := &stubNotificationService{}

	svc := NewOrderService(repo, actRepo, stock, notify)
	_, err := svc.Pay(1, 10)
	if !errors.Is(err, ErrOrderNotPending) {
		t.Fatalf("want ErrOrderNotPending, got %v", err)
	}
}

func TestOrderService_ScanExpired_NotifyAndRollback(t *testing.T) {
	repo := &stubOrderRepo{
		listExpired:             []domain.Order{{ID: 1, UserID: 10, ActivityID: 99}},
		updateFromPendingResult: true,
	}
	actRepo := &stubActivityRepo{activity: &domain.Activity{ID: 99, Title: "活动A"}}
	stock := &stubStockEngine{}
	notify := &stubNotificationService{}

	svc := NewOrderService(repo, actRepo, stock, notify)
	closed, err := svc.ScanExpired()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if closed != 1 {
		t.Fatalf("want closed=1, got %d", closed)
	}
	if stock.rollbackCalled != 1 {
		t.Fatalf("want rollback=1, got %d", stock.rollbackCalled)
	}
	if notify.expireNotified != 1 || notify.lastOrderID != 1 {
		t.Fatalf("expected order expire notified for order 1")
	}
	if actRepo.incrementStockCalled != 1 {
		t.Fatalf("expected increment stock called once")
	}
}

func TestOrderService_ScanExpired_SkipWhenAlreadyClosed(t *testing.T) {
	repo := &stubOrderRepo{
		listExpired:             []domain.Order{{ID: 2, UserID: 11, ActivityID: 88}},
		updateFromPendingResult: false,
	}
	actRepo := &stubActivityRepo{}
	stock := &stubStockEngine{}
	notify := &stubNotificationService{}

	svc := NewOrderService(repo, actRepo, stock, notify)
	closed, err := svc.ScanExpired()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if closed != 0 {
		t.Fatalf("want closed=0, got %d", closed)
	}
	if stock.rollbackCalled != 0 {
		t.Fatalf("rollback should not be called")
	}
	if notify.expireNotified != 0 {
		t.Fatalf("notification should not be sent")
	}
}
