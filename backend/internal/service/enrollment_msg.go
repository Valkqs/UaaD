package service

// EnrollmentMessage is the Kafka message body for an enrollment request
// that has passed the Redis Lua gate and awaits MySQL persistence.
type EnrollmentMessage struct {
	EnrollmentID uint64  `json:"enrollment_id"`
	UserID       uint64  `json:"user_id"`
	ActivityID   uint64  `json:"activity_id"`
	QueuePos     int64   `json:"queue_pos"`
	Price        float64 `json:"price"`
	Timestamp    int64   `json:"timestamp"`
}
