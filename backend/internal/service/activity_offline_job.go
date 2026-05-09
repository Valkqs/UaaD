package service

import (
	"context"
	"time"

	"github.com/uaad/backend/internal/domain"
	"gorm.io/gorm"
)

// ActivityOfflineResult summarizes one auto-offline scan.
type ActivityOfflineResult struct {
	OfflineCount int `json:"offline_count"`
}

// ActivityOfflineJob periodically scans for activities whose `activity_at`
// has passed and transitions them to OFFLINE. It implements SPRINT3 §三 task 8
// (活动逾期自动转 OFFLINE) — a separate concern from heat-score recalculation.
//
// Status transition rule:
//
//	WHERE activity_at < NOW() AND status NOT IN ('OFFLINE', 'CANCELLED')
//	  → SET status = 'OFFLINE'
//
// CANCELLED activities are kept as-is (already terminal). DRAFT activities
// that have an `activity_at` in the past are also transitioned because they
// cannot be meaningfully resurrected.
type ActivityOfflineJob struct {
	db *gorm.DB
	// now is injected for deterministic tests; production should use nil
	// to default to time.Now.
	now func() time.Time
}

// NewActivityOfflineJob creates the job.
func NewActivityOfflineJob(db *gorm.DB) *ActivityOfflineJob {
	return &ActivityOfflineJob{db: db}
}

// withNow returns a job copy that uses the provided clock (for tests).
func (j *ActivityOfflineJob) withNow(clock func() time.Time) *ActivityOfflineJob {
	cp := *j
	cp.now = clock
	return &cp
}

// Run executes one scan-and-transition pass. Safe to call concurrently with
// itself only if the database supports it (MySQL does — the WHERE clause
// is idempotent; double-run yields zero second-pass updates).
func (j *ActivityOfflineJob) Run(ctx context.Context) (*ActivityOfflineResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := time.Now()
	if j.now != nil {
		now = j.now()
	}

	res := j.db.WithContext(ctx).
		Model(&domain.Activity{}).
		Where("activity_at < ? AND status NOT IN ?",
			now, []string{"OFFLINE", "CANCELLED"}).
		Update("status", "OFFLINE")

	if res.Error != nil {
		return nil, res.Error
	}

	return &ActivityOfflineResult{OfflineCount: int(res.RowsAffected)}, nil
}
