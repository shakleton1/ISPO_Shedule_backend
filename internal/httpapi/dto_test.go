package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ispo-schedule/internal/schedule"
)

func TestToScheduleDayEventDTO(t *testing.T) {
	details := "Important details"

	input := schedule.ScheduleDayEvent{
		ID:         1,
		TargetDate: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		GroupID:    1,
		EventType:  "CANCEL",
		Title:      "Class Cancelled",
		Details:    &details,
		LocationID: intPtr(5),
	}

	got := toScheduleDayEventDTO(input)

	assert.Equal(t, int64(1), got.ID)
	assert.Equal(t, input.TargetDate, got.TargetDate)
	assert.Equal(t, 1, got.GroupID)
	assert.Equal(t, "CANCEL", got.EventType)
	assert.Equal(t, "Class Cancelled", got.Title)
	assert.Equal(t, details, *got.Details)
	assert.Equal(t, 5, *got.LocationID)
}

func TestToScheduleDayEventDTO_NilFields(t *testing.T) {
	input := schedule.ScheduleDayEvent{
		ID:         2,
		TargetDate: time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC),
		GroupID:    1,
		EventType:  "ADD",
		Title:      "Extra Class",
		Details:    nil,
		LocationID: nil,
	}

	got := toScheduleDayEventDTO(input)

	assert.Equal(t, int64(2), got.ID)
	assert.Equal(t, input.TargetDate, got.TargetDate)
	assert.Nil(t, got.Details)
	assert.Nil(t, got.LocationID)
}

// Note: toScheduleOverrideDTO требует integration тестов
// с полной загрузкой данных из БД
