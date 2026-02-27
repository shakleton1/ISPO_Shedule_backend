package db

import (
	"fmt"
	"os"
	"strings"

	"ispo-schedule/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	level := strings.ToLower(strings.TrimSpace(os.Getenv("ISPO_DB_LOG_LEVEL")))
	mode := logger.Warn
	switch level {
	case "", "warn", "warning":
		mode = logger.Warn
	case "silent":
		mode = logger.Silent
	case "error":
		mode = logger.Error
	case "info":
		mode = logger.Info
	default:
		mode = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(mode),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	return db, nil
}
