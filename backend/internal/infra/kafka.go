package infra

import (
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/uaad/backend/internal/config"
)

// NewKafkaWriter creates a Kafka producer writing to the enrollment topic.
func NewKafkaWriter(cfg *config.Config) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBroker),
		Topic:        cfg.KafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
	}
}

// NewKafkaReader creates a Kafka consumer for the enrollment consumer group.
func NewKafkaReader(cfg *config.Config) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.KafkaBroker},
		Topic:          cfg.KafkaTopic,
		GroupID:        cfg.KafkaConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10e6, // 10 MB
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset, // read from earliest when no committed offset exists
	})
}
