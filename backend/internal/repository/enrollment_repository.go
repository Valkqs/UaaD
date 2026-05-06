package repository

import (
	"github.com/uaad/backend/internal/domain"
	"gorm.io/gorm"
)

// EnrollmentRepository defines data access methods for enrollments.
type EnrollmentRepository interface {
	Create(enrollment *domain.Enrollment) error
	FindByID(id uint64) (*domain.Enrollment, error)
	FindByUserAndActivity(userID, activityID uint64) (*domain.Enrollment, error)
	UpdateStatus(id uint64, status string) error
	UpdateStatusFromQueuing(id, userID uint64, status string) (bool, error)
	ListByUserID(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error)
	ListByActivityID(activityID uint64, status string) ([]domain.Enrollment, error)
}

type enrollmentRepository struct {
	db *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) EnrollmentRepository {
	return &enrollmentRepository{db: db}
}

func (r *enrollmentRepository) Create(enrollment *domain.Enrollment) error {
	return r.db.Create(enrollment).Error
}

func (r *enrollmentRepository) FindByID(id uint64) (*domain.Enrollment, error) {
	var enrollment domain.Enrollment
	if err := r.db.First(&enrollment, id).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}

func (r *enrollmentRepository) FindByUserAndActivity(userID, activityID uint64) (*domain.Enrollment, error) {
	var enrollment domain.Enrollment
	if err := r.db.Where("user_id = ? AND activity_id = ?", userID, activityID).
		First(&enrollment).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}

func (r *enrollmentRepository) UpdateStatus(id uint64, status string) error {
	return r.db.Model(&domain.Enrollment{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"finalized_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *enrollmentRepository) UpdateStatusFromQueuing(id, userID uint64, status string) (bool, error) {
	res := r.db.Model(&domain.Enrollment{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, "QUEUING").
		Updates(map[string]interface{}{
			"status":       status,
			"finalized_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *enrollmentRepository) ListByUserID(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error) {
	var enrollments []domain.Enrollment
	var total int64

	query := r.db.Model(&domain.Enrollment{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&enrollments).Error; err != nil {
		return nil, 0, err
	}
	return enrollments, total, nil
}

func (r *enrollmentRepository) ListByActivityID(activityID uint64, status string) ([]domain.Enrollment, error) {
	var enrollments []domain.Enrollment
	query := r.db.Where("activity_id = ?", activityID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}
