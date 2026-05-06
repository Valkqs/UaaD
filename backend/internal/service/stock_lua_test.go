package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
)

// testRedisClient returns a redis.Client pointing at localhost:6379.
// It skips the test when Redis is unreachable.
func testRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis unavailable, skipping: %v", err)
	}
	t.Cleanup(func() { rdb.FlushDB(context.Background()); rdb.Close() })
	return rdb
}

func TestStockEngine_NormalDeduct(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9001
	if err := engine.WarmUp(ctx, activityID, 10); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	pos, err := engine.TryEnroll(ctx, activityID, 1)
	if err != nil {
		t.Fatalf("TryEnroll: %v", err)
	}
	if pos <= 0 {
		t.Fatalf("expected positive queue pos, got %d", pos)
	}

	stock, _ := rdb.Get(ctx, stockKey(activityID)).Int()
	if stock != 9 {
		t.Errorf("stock should be 9, got %d", stock)
	}
	isMember, _ := rdb.SIsMember(ctx, enrolledSetKey(activityID), "1").Result()
	if !isMember {
		t.Error("user 1 should be in enrolled set")
	}
}

func TestStockEngine_Idempotency(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9002
	if err := engine.WarmUp(ctx, activityID, 10); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	_, err := engine.TryEnroll(ctx, activityID, 42)
	if err != nil {
		t.Fatalf("first enroll: %v", err)
	}

	_, err = engine.TryEnroll(ctx, activityID, 42)
	if !errors.Is(err, ErrAlreadyEnrolledRedis) {
		t.Fatalf("expected ErrAlreadyEnrolledRedis, got %v", err)
	}

	stock, _ := rdb.Get(ctx, stockKey(activityID)).Int()
	if stock != 9 {
		t.Errorf("stock should remain 9 after duplicate, got %d", stock)
	}
}

func TestStockEngine_SoldOut(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9003
	if err := engine.WarmUp(ctx, activityID, 0); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	_, err := engine.TryEnroll(ctx, activityID, 1)
	if !errors.Is(err, ErrStockDepleted) {
		t.Fatalf("expected ErrStockDepleted, got %v", err)
	}
}

func TestStockEngine_Rollback(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9004
	if err := engine.WarmUp(ctx, activityID, 10); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	_, err := engine.TryEnroll(ctx, activityID, 7)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	if err := engine.Rollback(ctx, activityID, 7); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	stock, _ := rdb.Get(ctx, stockKey(activityID)).Int()
	if stock != 10 {
		t.Errorf("stock should be 10 after rollback, got %d", stock)
	}
	isMember, _ := rdb.SIsMember(ctx, enrolledSetKey(activityID), "7").Result()
	if isMember {
		t.Error("user 7 should NOT be in enrolled set after rollback")
	}
}

func TestStockEngine_GetStock(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9006
	if err := engine.WarmUp(ctx, activityID, 12); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	remaining, err := engine.GetStock(ctx, activityID)
	if err != nil {
		t.Fatalf("GetStock: %v", err)
	}
	if remaining != 12 {
		t.Errorf("stock should be 12, got %d", remaining)
	}
}

func TestStockEngine_Concurrent(t *testing.T) {
	rdb := testRedisClient(t)
	engine := NewStockEngine(rdb)
	ctx := context.Background()

	const activityID uint64 = 9005
	const stock = 10
	const goroutines = 100

	if err := engine.WarmUp(ctx, activityID, stock); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	// Reset global queue counter for deterministic assertions.
	rdb.Del(ctx, queueCounterKey)

	var success atomic.Int32
	var wg sync.WaitGroup
	ready := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(uid uint64) {
			defer wg.Done()
			<-ready
			_, err := engine.TryEnroll(ctx, activityID, uid)
			if err == nil {
				success.Add(1)
			}
		}(uint64(i + 1))
	}

	close(ready)
	wg.Wait()

	s := success.Load()
	t.Logf("result: success=%d, total=%d, stock=%d", s, goroutines, stock)
	if int(s) != stock {
		t.Errorf("expected exactly %d successes, got %d", stock, s)
	}

	remaining, _ := rdb.Get(ctx, stockKey(activityID)).Int()
	if remaining != 0 {
		t.Errorf("remaining stock should be 0, got %d", remaining)
	}

	setSize, _ := rdb.SCard(ctx, enrolledSetKey(activityID)).Result()
	if setSize != int64(stock) {
		t.Errorf("enrolled set size should be %d, got %d", stock, setSize)
	}

	// Verify queue positions are sequential 1..stock.
	queueMax, _ := rdb.Get(ctx, queueCounterKey).Int()
	if queueMax != stock {
		t.Errorf("global queue counter should be %d, got %d", stock, queueMax)
	}

	// Verify all enrolled users are distinct and in 1..goroutines.
	members, _ := rdb.SMembers(ctx, enrolledSetKey(activityID)).Result()
	seen := make(map[uint64]bool)
	for _, m := range members {
		uid, _ := strconv.ParseUint(m, 10, 64)
		if uid < 1 || uid > goroutines {
			t.Errorf("unexpected user in enrolled set: %s", m)
		}
		if seen[uid] {
			t.Errorf("duplicate user in enrolled set: %d", uid)
		}
		seen[uid] = true
	}
	_ = fmt.Sprintf("placeholder") // silence import
}
