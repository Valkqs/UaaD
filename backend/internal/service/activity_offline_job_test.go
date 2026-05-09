package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/uaad/backend/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOfflineJobDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:offline_job_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Activity{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func makeActivity(t *testing.T, db *gorm.DB, status string, activityAt time.Time) uint64 {
	t.Helper()
	a := &domain.Activity{
		Title:         fmt.Sprintf("activity-%s-%d", status, activityAt.UnixNano()),
		Description:   "test",
		Location:      "x",
		Category:      "CONCERT",
		MaxCapacity:   100,
		EnrollOpenAt:  activityAt.Add(-3 * 24 * time.Hour),
		EnrollCloseAt: activityAt.Add(-1 * 24 * time.Hour),
		ActivityAt:    activityAt,
		Status:        status,
		CreatedBy:     1,
	}
	if err := db.Create(a).Error; err != nil {
		t.Fatalf("seed activity: %v", err)
	}
	return a.ID
}

// TestActivityOfflineJob_TransitionsExpiredPublishedActivities
// covers the happy path: a PUBLISHED activity whose activity_at lies in the
// past must flip to OFFLINE.
func TestActivityOfflineJob_TransitionsExpiredPublishedActivities(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	// 1 expired PUBLISHED → should flip
	expiredID := makeActivity(t, db, "PUBLISHED", now.Add(-1*time.Hour))
	// 1 future PUBLISHED → must stay
	futureID := makeActivity(t, db, "PUBLISHED", now.Add(48*time.Hour))

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	res, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.OfflineCount != 1 {
		t.Errorf("OfflineCount: want 1, got %d", res.OfflineCount)
	}

	var expired, future domain.Activity
	db.First(&expired, expiredID)
	db.First(&future, futureID)

	if expired.Status != "OFFLINE" {
		t.Errorf("expired activity should be OFFLINE, got %s", expired.Status)
	}
	if future.Status != "PUBLISHED" {
		t.Errorf("future activity must remain PUBLISHED, got %s", future.Status)
	}
}

// TestActivityOfflineJob_SkipsAlreadyTerminal verifies that activities
// already in OFFLINE or CANCELLED are NOT re-touched (idempotent + leaves
// CANCELLED as a separate terminal state).
func TestActivityOfflineJob_SkipsAlreadyTerminal(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	offlineID := makeActivity(t, db, "OFFLINE", now.Add(-2*time.Hour))
	cancelledID := makeActivity(t, db, "CANCELLED", now.Add(-2*time.Hour))

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	res, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.OfflineCount != 0 {
		t.Errorf("terminal states should not be touched, got %d transitions", res.OfflineCount)
	}

	var offline, cancelled domain.Activity
	db.First(&offline, offlineID)
	db.First(&cancelled, cancelledID)

	if offline.Status != "OFFLINE" {
		t.Errorf("OFFLINE must remain OFFLINE, got %s", offline.Status)
	}
	if cancelled.Status != "CANCELLED" {
		t.Errorf("CANCELLED must remain CANCELLED, got %s", cancelled.Status)
	}
}

// TestActivityOfflineJob_TransitionsAllNonTerminalStates verifies DRAFT,
// PREHEAT, PUBLISHED, SELLING_OUT, SOLD_OUT all flip to OFFLINE when expired.
func TestActivityOfflineJob_TransitionsAllNonTerminalStates(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)

	states := []string{"DRAFT", "PREHEAT", "PUBLISHED", "SELLING_OUT", "SOLD_OUT"}
	ids := make(map[string]uint64, len(states))
	for _, s := range states {
		ids[s] = makeActivity(t, db, s, past)
	}

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	res, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.OfflineCount != len(states) {
		t.Errorf("OfflineCount: want %d, got %d", len(states), res.OfflineCount)
	}

	for _, s := range states {
		var a domain.Activity
		db.First(&a, ids[s])
		if a.Status != "OFFLINE" {
			t.Errorf("%s should flip to OFFLINE, got %s", s, a.Status)
		}
	}
}

// TestActivityOfflineJob_Idempotent verifies that running twice yields zero
// updates on the second pass — the WHERE clause already excludes OFFLINE.
func TestActivityOfflineJob_Idempotent(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	makeActivity(t, db, "PUBLISHED", now.Add(-1*time.Hour))

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	first, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if first.OfflineCount != 1 {
		t.Errorf("first run: want 1 transition, got %d", first.OfflineCount)
	}

	second, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.OfflineCount != 0 {
		t.Errorf("second run should be idempotent (0 transitions), got %d", second.OfflineCount)
	}
}

// TestActivityOfflineJob_RespectsContext verifies the job aborts when the
// context is already cancelled.
func TestActivityOfflineJob_RespectsContext(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	makeActivity(t, db, "PUBLISHED", now.Add(-1*time.Hour))

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := job.Run(ctx)
	if err == nil {
		t.Errorf("expected error when context is cancelled, got nil")
	}
}

// TestActivityOfflineJob_NoExpiredActivities verifies the job is a no-op
// when nothing has expired yet.
func TestActivityOfflineJob_NoExpiredActivities(t *testing.T) {
	db := setupOfflineJobDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	makeActivity(t, db, "PUBLISHED", now.Add(72*time.Hour))
	makeActivity(t, db, "PREHEAT", now.Add(48*time.Hour))

	job := NewActivityOfflineJob(db).withNow(func() time.Time { return now })
	res, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if res.OfflineCount != 0 {
		t.Errorf("no expired activities, want 0 transitions, got %d", res.OfflineCount)
	}
}
