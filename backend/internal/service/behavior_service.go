package service

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
)

var (
	ErrInvalidBehaviorType = errors.New("invalid behavior_type")
	ErrBehaviorBatchEmpty  = errors.New("batch items empty")
	ErrBehaviorBatchTooBig = errors.New("batch exceeds max size")
)

const behaviorBatchMax = 100

var allowedBehaviorTypes = map[string]struct{}{
	"VIEW":    {},
	"COLLECT": {},
	"SHARE":   {},
	"CLICK":   {},
	"SEARCH":  {},
}

// BehaviorSubmit is one behavior event from the client.
type BehaviorSubmit struct {
	ActivityID   uint64                 `json:"activity_id"`
	BehaviorType string                 `json:"behavior_type"`
	Detail       map[string]interface{} `json:"detail"`
}

// BehaviorService records user behaviors for recommendation analytics.
type BehaviorService interface {
	Submit(userID uint64, async bool, item BehaviorSubmit) error
	SubmitBatch(userID uint64, async bool, items []BehaviorSubmit) error
}

type behaviorService struct {
	repo         repository.BehaviorRepository
	activityRepo repository.ActivityRepository // optional: drives view_count for hot-score (SPRINT3 §三 task 8)
}

// NewBehaviorService creates a BehaviorService.
//
// Pass a non-nil activityRepo to wire VIEW behaviors into activities.view_count
// — required for the recommendation hot-score pipeline. nil disables that
// integration (kept for tests that only care about behavior persistence).
func NewBehaviorService(repo repository.BehaviorRepository, activityRepo repository.ActivityRepository) BehaviorService {
	return &behaviorService{repo: repo, activityRepo: activityRepo}
}

// bumpViewCount runs activityRepo.IncrementViewCount for VIEW events.
// Best-effort: errors are logged but do not fail the behavior write.
func (s *behaviorService) bumpViewCount(b *domain.UserBehavior) {
	if s.activityRepo == nil || b.BehaviorType != "VIEW" {
		return
	}
	if err := s.activityRepo.IncrementViewCount(b.ActivityID); err != nil {
		log.Printf("[behavior] increment view_count failed activity=%d: %v", b.ActivityID, err)
	}
}

func validateAndToDomain(userID uint64, item BehaviorSubmit) (*domain.UserBehavior, error) {
	if item.ActivityID == 0 {
		return nil, errors.New("activity_id required")
	}
	if _, ok := allowedBehaviorTypes[item.BehaviorType]; !ok {
		return nil, ErrInvalidBehaviorType
	}
	var detailStr *string
	if len(item.Detail) > 0 {
		b, err := json.Marshal(item.Detail)
		if err != nil {
			return nil, err
		}
		s := string(b)
		detailStr = &s
	}
	return &domain.UserBehavior{
		UserID:       userID,
		ActivityID:   item.ActivityID,
		BehaviorType: item.BehaviorType,
		Detail:       detailStr,
	}, nil
}

func (s *behaviorService) Submit(userID uint64, async bool, item BehaviorSubmit) error {
	b, err := validateAndToDomain(userID, item)
	if err != nil {
		return err
	}
	if async {
		clone := *b
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[behavior] async submit panic: %v", r)
				}
			}()
			if err := s.repo.Create(&clone); err != nil {
				log.Printf("[behavior] async create: %v", err)
				return
			}
			s.bumpViewCount(&clone)
		}()
		return nil
	}
	if err := s.repo.Create(b); err != nil {
		return err
	}
	s.bumpViewCount(b)
	return nil
}

func (s *behaviorService) SubmitBatch(userID uint64, async bool, items []BehaviorSubmit) error {
	if len(items) == 0 {
		return ErrBehaviorBatchEmpty
	}
	if len(items) > behaviorBatchMax {
		return ErrBehaviorBatchTooBig
	}
	rows := make([]*domain.UserBehavior, 0, len(items))
	for i := range items {
		b, err := validateAndToDomain(userID, items[i])
		if err != nil {
			return err
		}
		rows = append(rows, b)
	}
	if async {
		cp := make([]*domain.UserBehavior, len(rows))
		for i, r := range rows {
			c := *r
			cp[i] = &c
		}
		go func(list []*domain.UserBehavior) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[behavior] async batch panic: %v", r)
				}
			}()
			if err := s.repo.BatchCreate(list); err != nil {
				log.Printf("[behavior] async batch create: %v", err)
				return
			}
			for _, b := range list {
				s.bumpViewCount(b)
			}
		}(cp)
		return nil
	}
	if err := s.repo.BatchCreate(rows); err != nil {
		return err
	}
	for _, b := range rows {
		s.bumpViewCount(b)
	}
	return nil
}
