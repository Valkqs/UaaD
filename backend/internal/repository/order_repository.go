package repository

import (
	"github.com/uaad/backend/internal/domain"
	"gorm.io/gorm"
)

// OrderRepository defines data access methods for orders.
type OrderRepository interface {
	Create(order *domain.Order) error
	FindByID(id uint64) (*domain.Order, error)
	FindByOrderNo(orderNo string) (*domain.Order, error)
	FindByEnrollmentID(enrollmentID uint64) (*domain.Order, error)
	FindByUserID(userID uint64, page, pageSize int) ([]domain.Order, int64, error)
	UpdateStatus(id uint64, status string) error
	UpdateStatusFromPending(id uint64, status string) (bool, error)
	ListExpired() ([]domain.Order, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order *domain.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepository) FindByID(id uint64) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindByOrderNo(orderNo string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindByEnrollmentID(enrollmentID uint64) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.Where("enrollment_id = ?", enrollmentID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindByUserID(userID uint64, page, pageSize int) ([]domain.Order, int64, error) {
	var orders []domain.Order
	var total int64

	query := r.db.Model(&domain.Order{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderRepository) UpdateStatus(id uint64, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == "PAID" {
		updates["paid_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}
	return r.db.Model(&domain.Order{}).Where("id = ?", id).Updates(updates).Error
}

func (r *orderRepository) UpdateStatusFromPending(id uint64, status string) (bool, error) {
	updates := map[string]interface{}{"status": status}
	if status == "PAID" {
		updates["paid_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}
	res := r.db.Model(&domain.Order{}).
		Where("id = ? AND status = ?", id, "PENDING").
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *orderRepository) ListExpired() ([]domain.Order, error) {
	var orders []domain.Order
	if err := r.db.Where("status = ? AND expired_at < CURRENT_TIMESTAMP", "PENDING").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
