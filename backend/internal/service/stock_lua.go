package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// Sentinel errors returned by StockEngine.TryEnroll.
var (
	ErrAlreadyEnrolledRedis = errors.New("user already in enrolled set")
	ErrStockDepleted        = errors.New("redis stock depleted")
)

// StockEngine abstracts the Redis-backed atomic inventory operations required
// by the enrollment flow.
type StockEngine interface {
	// TryEnroll atomically checks idempotency, deducts one stock unit, adds
	// the user to the enrolled set, and returns a global queue position.
	//   -2 → ErrAlreadyEnrolledRedis
	//   -1 → ErrStockDepleted
	//   >0 → queue position (success)
	TryEnroll(ctx context.Context, activityID, userID uint64) (queuePos int64, err error)

	// GetStock returns current remaining stock from Redis.
	GetStock(ctx context.Context, activityID uint64) (int, error)

	// Rollback compensates a previous TryEnroll: increments stock by 1 and
	// removes the user from the enrolled set. Used by order-expiry and
	// MySQL-failure compensation paths.
	Rollback(ctx context.Context, activityID, userID uint64) error

	// WarmUp pre-loads the stock value into Redis and resets the enrolled set
	// for a given activity. Typically called when an activity is published.
	WarmUp(ctx context.Context, activityID uint64, stock int) error
}

// --- key helpers (SPRINT2 convention) ---

func stockKey(activityID uint64) string {
	return fmt.Sprintf("activity:%d:stock", activityID)
}

func enrolledSetKey(activityID uint64) string {
	return fmt.Sprintf("activity:%d:enrolled_set", activityID)
}

const queueCounterKey = "activity:queue:global:counter"

// --- Lua scripts ---

// enrollScript performs the three-step atomic enrollment:
//
//	KEYS[1] = activity:{id}:stock
//	KEYS[2] = activity:{id}:enrolled_set
//	KEYS[3] = activity:queue:global:counter
//	ARGV[1] = userID (string)
//
// Returns: -2 (duplicate), -1 (no stock), or a positive queue position.
var enrollScript = redis.NewScript(`
local stockKey    = KEYS[1]
local enrolledKey = KEYS[2]
local queueKey    = KEYS[3]
local userID      = ARGV[1]

if redis.call('SISMEMBER', enrolledKey, userID) == 1 then
    return -2
end

local stock = tonumber(redis.call('GET', stockKey))
if not stock or stock <= 0 then
    return -1
end

redis.call('DECRBY', stockKey, 1)
redis.call('SADD', enrolledKey, userID)
local pos = redis.call('INCR', queueKey)
return pos
`)

// --- implementation ---

type redisStockEngine struct {
	rdb *redis.Client
}

// NewStockEngine creates a StockEngine backed by the given Redis client.
func NewStockEngine(rdb *redis.Client) StockEngine {
	return &redisStockEngine{rdb: rdb}
}

func (e *redisStockEngine) TryEnroll(ctx context.Context, activityID, userID uint64) (int64, error) {
	keys := []string{
		stockKey(activityID),
		enrolledSetKey(activityID),
		queueCounterKey,
	}
	result, err := enrollScript.Run(ctx, e.rdb, keys, strconv.FormatUint(userID, 10)).Int64()
	if err != nil {
		return 0, fmt.Errorf("stock lua eval: %w", err)
	}

	switch result {
	case -2:
		return 0, ErrAlreadyEnrolledRedis
	case -1:
		return 0, ErrStockDepleted
	default:
		return result, nil
	}
}

func (e *redisStockEngine) GetStock(ctx context.Context, activityID uint64) (int, error) {
	return e.rdb.Get(ctx, stockKey(activityID)).Int()
}

func (e *redisStockEngine) Rollback(ctx context.Context, activityID, userID uint64) error {
	pipe := e.rdb.Pipeline()
	pipe.Incr(ctx, stockKey(activityID))
	pipe.SRem(ctx, enrolledSetKey(activityID), strconv.FormatUint(userID, 10))
	_, err := pipe.Exec(ctx)
	return err
}

func (e *redisStockEngine) WarmUp(ctx context.Context, activityID uint64, stock int) error {
	pipe := e.rdb.Pipeline()
	pipe.Set(ctx, stockKey(activityID), stock, 0)
	pipe.Del(ctx, enrolledSetKey(activityID))
	_, err := pipe.Exec(ctx)
	return err
}
