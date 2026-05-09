// Package config provides application configuration loaded from environment variables.
// All settings have sensible defaults for local development.
package config

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ScoringWeights holds weights for activity hot-score calculation (from env).
type ScoringWeights struct {
	ViewWeight      float64
	EnrollWeight    float64
	SpeedWeight     float64
	TimeDecayWeight float64
	HotTTLLHours    float64
}

// Config holds all application configuration.
type Config struct {
	AppEnv     string
	Port       string
	JWTSecret  string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	// MySQL 连接池（与 sql.DB / GORM 一致，见 ApplyMySQLPool）
	DBMaxIdleConns     int
	DBMaxOpenConns     int
	DBConnMaxLifetime  time.Duration
	RedisHost          string
	RedisPort          string
	KafkaBroker        string
	KafkaTopic         string
	KafkaConsumerGroup string

	Scoring                    ScoringWeights
	ScoreRecalcIntervalMinutes int
	StockReconcileMinutes      int
	ActivityOfflineMinutes     int
	BehaviorWriteAsync         bool
	CORSAllowedOrigins         []string
}

// Load returns a Config populated from environment variables,
// falling back to development-friendly defaults.
func Load() *Config {
	return &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          getEnv("JWT_SECRET", "uaad-super-secret-key-2026"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "3306"),
		DBUser:             getEnv("DB_USER", "root"),
		DBPassword:         getEnv("DB_PASSWORD", "root"),
		DBName:             getEnv("DB_NAME", "uaad"),
		DBMaxIdleConns:     parseIntEnv("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:     parseIntEnv("DB_MAX_OPEN_CONNS", 100),
		DBConnMaxLifetime:  parseDurationEnv("DB_CONN_MAX_LIFETIME", time.Hour),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		KafkaBroker:        getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:         getEnv("KAFKA_TOPIC_ENROLLMENT", "enrollment_requests"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "uaad-enrollment-consumer"),

		Scoring: ScoringWeights{
			ViewWeight:      parseFloatEnv("SCORE_WEIGHT_VIEW", 0.2),
			EnrollWeight:    parseFloatEnv("SCORE_WEIGHT_ENROLL", 0.35),
			SpeedWeight:     parseFloatEnv("SCORE_WEIGHT_SPEED", 0.25),
			TimeDecayWeight: parseFloatEnv("SCORE_WEIGHT_TIME_DECAY", 0.2),
			HotTTLLHours:    parseFloatEnv("SCORE_HOT_TTL_HOURS", 720),
		},
		ScoreRecalcIntervalMinutes: parseIntEnv("SCORE_RECALC_MINUTES", 30),
		StockReconcileMinutes:      parseIntEnv("STOCK_RECONCILE_MINUTES", 10),
		ActivityOfflineMinutes:     parseIntEnv("ACTIVITY_OFFLINE_MINUTES", 15),
		BehaviorWriteAsync:         parseBoolEnv("BEHAVIOR_ASYNC_WRITE", true),
		CORSAllowedOrigins:         parseCSVEnv("CORS_ALLOWED_ORIGINS"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseFloatEnv(key string, fallback float64) float64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func parseIntEnv(key string, fallback int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func parseBoolEnv(key string, fallback bool) bool {
	s := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if s == "" {
		return fallback
	}
	switch s {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// MySQLDSN returns a go-sql-driver DSN for GORM (utf8mb4, parseTime, UTC).
func (c *Config) MySQLDSN() string {
	mcfg := mysql.Config{
		User:                 c.DBUser,
		Passwd:               c.DBPassword,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", c.DBHost, c.DBPort),
		DBName:               c.DBName,
		Params:               map[string]string{"charset": "utf8mb4", "parseTime": "true", "loc": "UTC"},
		AllowNativePasswords: true,
	}
	return mcfg.FormatDSN()
}

// ApplyMySQLPool 将连接池参数应用到 *sql.DB（服务端、测试、seed 共用同一套 env）。
func (c *Config) ApplyMySQLPool(sqlDB *sql.DB) {
	sqlDB.SetMaxIdleConns(c.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(c.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(c.DBConnMaxLifetime)
}
