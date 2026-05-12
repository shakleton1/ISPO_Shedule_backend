//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ispo-schedule/internal/schedule"
)

// getTestDB возвращает подключение к тестовой БД
func getTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("ISPO_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/ispo_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	return db
}

// setupTestGroup создаёт тестовую группу и возвращает её
func setupTestGroup(t *testing.T, db *gorm.DB) *schedule.Group {
	t.Helper()

	group := &schedule.Group{
		Name:   "Test Group",
		Course: 1,
	}

	// Clean up if exists
	db.Where("name = ?", group.Name).Delete(&schedule.Group{})

	err := db.Create(group).Error
	require.NoError(t, err)

	return group
}
