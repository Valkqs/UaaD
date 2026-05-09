package repository

import (
	"github.com/uaad/backend/internal/domain"
	"gorm.io/gorm"
)

// ActivityFilter represents query filters for listing activities.
type ActivityFilter struct {
	Category string
	Status   string
	Keyword  string
	Sort     string // "hot", "recent", "soon"
}

// ActivityRepository defines data access methods for activities.
type ActivityRepository interface {
	Create(activity *domain.Activity) error
	FindByID(id uint64) (*domain.Activity, error)
	Update(activity *domain.Activity) error
	Delete(id uint64) error
	List(filter ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error)
	PublishedList(page, pageSize int) ([]domain.Activity, int64, error)
	FindByMerchantID(merchantID uint64) ([]domain.Activity, error)
	DeductStock(activityID uint64) (int64, error)
	IncrementStock(activityID uint64) error
	IncrementViewCount(activityID uint64) error
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

func (r *activityRepository) Create(activity *domain.Activity) error {
	return r.db.Create(activity).Error
}

func (r *activityRepository) FindByID(id uint64) (*domain.Activity, error) {
	var activity domain.Activity
	if err := r.db.First(&activity, id).Error; err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *activityRepository) Update(activity *domain.Activity) error {
	return r.db.Save(activity).Error
}

func (r *activityRepository) Delete(id uint64) error {
	return r.db.Delete(&domain.Activity{}, id).Error
}

func (r *activityRepository) List(filter ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error) {
	var activities []domain.Activity
	var total int64

	query := r.db.Model(&domain.Activity{})
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	switch filter.Sort {
	case "hot":
		query = query.Order("enroll_count DESC")
	case "soon":
		query = query.Order("enroll_open_at ASC")
	default: // "recent" or empty
		query = query.Order("created_at DESC")
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&activities).Error; err != nil {
		return nil, 0, err
	}
	return activities, total, nil
}

func (r *activityRepository) PublishedList(page, pageSize int) ([]domain.Activity, int64, error) {
	var activities []domain.Activity
	var total int64

	query := r.db.Model(&domain.Activity{}).Where("status = ?", "PUBLISHED")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&activities).Error; err != nil {
		return nil, 0, err
	}
	return activities, total, nil
}

func (r *activityRepository) FindByMerchantID(merchantID uint64) ([]domain.Activity, error) {
	var activities []domain.Activity
	if err := r.db.Where("created_by = ?", merchantID).Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *activityRepository) DeductStock(activityID uint64) (int64, error) {
	result := r.db.Model(&domain.Activity{}).
		Where("id = ? AND enroll_count < max_capacity AND status = ?", activityID, "PUBLISHED").
		Updates(map[string]interface{}{
			"enroll_count": gorm.Expr("enroll_count + 1"),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *activityRepository) IncrementStock(activityID uint64) error {
	return r.db.Model(&domain.Activity{}).
		Where("id = ? AND enroll_count > 0", activityID).
		Update("enroll_count", gorm.Expr("enroll_count - 1")).Error
}

// IncrementViewCount atomically increases activities.view_count by 1.
// Best-effort: callers (e.g. behavior service for VIEW events) should not
// fail their main flow when this errors. SPRINT3 §三 task 8 wires this into
// the recommendation hot-score pipeline.
func (r *activityRepository) IncrementViewCount(activityID uint64) error {
	return r.db.Model(&domain.Activity{}).
		Where("id = ?", activityID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}
