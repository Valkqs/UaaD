package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
)

var (
	ErrActivityNotFound     = errors.New("activity not found")
	ErrNotActivityOwner     = errors.New("not the owner of this activity")
	ErrInvalidActivityState = errors.New("invalid activity state transition")
	ErrActivityPublished    = errors.New("cannot modify locked fields after publish")
	ErrInvalidTimeRange     = errors.New("invalid time range: enroll_open < enroll_close < activity_at required")
)

// CreateActivityReq represents the request body for creating an activity.
type CreateActivityReq struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description" binding:"required"`
	CoverURL      *string  `json:"cover_url"`
	Location      string   `json:"location" binding:"required"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	Category      string   `json:"category" binding:"required"`
	Tags          []string `json:"tags"`
	MaxCapacity   int      `json:"max_capacity" binding:"required,min=1"`
	Price         float64  `json:"price" binding:"min=0"`
	EnrollOpenAt  string   `json:"enroll_open_at" binding:"required"`
	EnrollCloseAt string   `json:"enroll_close_at" binding:"required"`
	ActivityAt    string   `json:"activity_at" binding:"required"`
}

// UpdateActivityReq represents the request body for updating an activity.
type UpdateActivityReq struct {
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	CoverURL      *string  `json:"cover_url"`
	Location      *string  `json:"location"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	Category      *string  `json:"category"`
	Tags          []string `json:"tags"`
	MaxCapacity   *int     `json:"max_capacity"`
	Price         *float64 `json:"price"`
	EnrollOpenAt  *string  `json:"enroll_open_at"`
	EnrollCloseAt *string  `json:"enroll_close_at"`
	ActivityAt    *string  `json:"activity_at"`
}

// ActivityService defines business logic for activities.
type ActivityService interface {
	Create(merchantID uint64, req CreateActivityReq) (uint64, error)
	Update(activityID, merchantID uint64, req UpdateActivityReq) error
	Publish(activityID, merchantID uint64) (*domain.Activity, error)
	List(filter repository.ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error)
	Detail(id uint64) (*domain.Activity, error)
	Stock(id uint64) (remaining int, maxCapacity int, err error)
	MerchantList(merchantID uint64) ([]domain.Activity, error)
}

type activityService struct {
	repo        repository.ActivityRepository
	stockEngine StockEngine
}

// NewActivityService creates a new ActivityService.
func NewActivityService(repo repository.ActivityRepository, stockEngine StockEngine) ActivityService {
	return &activityService{repo: repo, stockEngine: stockEngine}
}

func (s *activityService) Create(merchantID uint64, req CreateActivityReq) (uint64, error) {
	enrollOpen, err := time.Parse(time.RFC3339, req.EnrollOpenAt)
	if err != nil {
		return 0, ErrInvalidTimeRange
	}
	enrollClose, err := time.Parse(time.RFC3339, req.EnrollCloseAt)
	if err != nil {
		return 0, ErrInvalidTimeRange
	}
	activityAt, err := time.Parse(time.RFC3339, req.ActivityAt)
	if err != nil {
		return 0, ErrInvalidTimeRange
	}

	if !enrollOpen.Before(enrollClose) || !enrollClose.Before(activityAt) {
		return 0, ErrInvalidTimeRange
	}

	var tagsJSON *string
	if len(req.Tags) > 0 {
		s, _ := jsonMarshalTags(req.Tags)
		tagsJSON = &s
	}

	activity := &domain.Activity{
		Title:         req.Title,
		Description:   req.Description,
		CoverURL:      req.CoverURL,
		Location:      req.Location,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Category:      req.Category,
		Tags:          tagsJSON,
		MaxCapacity:   req.MaxCapacity,
		Price:         req.Price,
		EnrollOpenAt:  enrollOpen,
		EnrollCloseAt: enrollClose,
		ActivityAt:    activityAt,
		Status:        "DRAFT",
		CreatedBy:     merchantID,
	}

	if err := s.repo.Create(activity); err != nil {
		return 0, err
	}
	return activity.ID, nil
}

func (s *activityService) Update(activityID, merchantID uint64, req UpdateActivityReq) error {
	activity, err := s.repo.FindByID(activityID)
	if err != nil {
		return ErrActivityNotFound
	}
	if activity.CreatedBy != merchantID {
		return ErrNotActivityOwner
	}

	// After PUBLISHED, cannot modify max_capacity and enroll_open_at
	published := activity.Status != "DRAFT" && activity.Status != "PREHEAT"
	if published {
		if req.MaxCapacity != nil || req.EnrollOpenAt != nil {
			return ErrActivityPublished
		}
	}

	if req.Title != nil {
		activity.Title = *req.Title
	}
	if req.Description != nil {
		activity.Description = *req.Description
	}
	if req.CoverURL != nil {
		activity.CoverURL = req.CoverURL
	}
	if req.Location != nil {
		activity.Location = *req.Location
	}
	if req.Latitude != nil {
		activity.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		activity.Longitude = req.Longitude
	}
	if req.Category != nil {
		activity.Category = *req.Category
	}
	if len(req.Tags) > 0 {
		t, _ := jsonMarshalTags(req.Tags)
		activity.Tags = &t
	}
	if req.MaxCapacity != nil {
		activity.MaxCapacity = *req.MaxCapacity
	}
	if req.Price != nil {
		activity.Price = *req.Price
	}
	if req.EnrollOpenAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EnrollOpenAt)
		if err != nil {
			return ErrInvalidTimeRange
		}
		activity.EnrollOpenAt = t
	}
	if req.EnrollCloseAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EnrollCloseAt)
		if err != nil {
			return ErrInvalidTimeRange
		}
		activity.EnrollCloseAt = t
	}
	if req.ActivityAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ActivityAt)
		if err != nil {
			return ErrInvalidTimeRange
		}
		activity.ActivityAt = t
	}

	// Validate time order
	if !activity.EnrollOpenAt.Before(activity.EnrollCloseAt) || !activity.EnrollCloseAt.Before(activity.ActivityAt) {
		return ErrInvalidTimeRange
	}

	return s.repo.Update(activity)
}

func (s *activityService) Publish(activityID, merchantID uint64) (*domain.Activity, error) {
	activity, err := s.repo.FindByID(activityID)
	if err != nil {
		return nil, ErrActivityNotFound
	}
	if activity.CreatedBy != merchantID {
		return nil, ErrNotActivityOwner
	}
	if activity.Status != "DRAFT" && activity.Status != "PREHEAT" {
		return nil, ErrInvalidActivityState
	}

	activity.Status = "PUBLISHED"
	if err := s.repo.Update(activity); err != nil {
		return nil, err
	}

	// Cache warm-up: pre-load stock into Redis (non-blocking on failure)
	if err := s.stockEngine.WarmUp(context.Background(), activityID, activity.MaxCapacity); err != nil {
		log.Printf("[ActivityPublish] WARNING: Redis warm-up failed for activity=%d: %v", activityID, err)
	}

	return activity, nil
}

func (s *activityService) List(filter repository.ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(filter, page, pageSize)
}

func (s *activityService) Detail(id uint64) (*domain.Activity, error) {
	activity, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrActivityNotFound
	}
	return activity, nil
}

func (s *activityService) Stock(id uint64) (int, int, error) {
	activity, err := s.repo.FindByID(id)
	if err != nil {
		return 0, 0, ErrActivityNotFound
	}

	remaining, err := s.stockEngine.GetStock(context.Background(), id)
	if err != nil {
		remaining = activity.MaxCapacity - int(activity.EnrollCount)
		if remaining < 0 {
			remaining = 0
		}
	}
	return remaining, activity.MaxCapacity, nil
}

func (s *activityService) MerchantList(merchantID uint64) ([]domain.Activity, error) {
	return s.repo.FindByMerchantID(merchantID)
}

func jsonMarshalTags(tags []string) (string, error) {
	// Simple JSON array marshaling without importing encoding/json at package level
	result := "["
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += `"` + tag + `"`
	}
	result += "]"
	return result, nil
}
