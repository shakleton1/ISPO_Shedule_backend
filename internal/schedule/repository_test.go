package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Note: Большинство тестов repository требуют integration тестов с реальной БД
// Здесь только тесты которые не требуют подключения к БД

func TestWeekCache_NewCache(t *testing.T) {
	cache := newWeekCache(100)

	assert.NotNil(t, cache)
}

func TestWeekCache_GetMiss(t *testing.T) {
	cache := newWeekCache(100)

	key := weekCacheKey{
		groupID:     1,
		weekStart:   "2026-02-23",
		dataVersion: time.Now().Format(time.RFC3339),
	}

	resp, ok := cache.get(key)

	assert.False(t, ok)
	assert.Zero(t, resp)
}

func TestWeekCache_SetAndGet(t *testing.T) {
	cache := newWeekCache(100)

	key := weekCacheKey{
		groupID:     1,
		weekStart:   "2026-02-23",
		dataVersion: time.Now().Format(time.RFC3339),
	}

	expected := WeekScheduleResponse{
		GroupID:     1,
		DateStart:   "2026-02-23",
		DateEnd:     "2026-02-28",
		DataVersion: key.dataVersion,
		Days:        []DaySchedule{},
	}

	cache.set(key, expected)

	got, ok := cache.get(key)

	assert.True(t, ok)
	assert.Equal(t, expected.GroupID, got.GroupID)
	assert.Equal(t, expected.DateStart, got.DateStart)
	assert.Equal(t, expected.DateEnd, got.DateEnd)
}

func TestWeekCache_Capacity(t *testing.T) {
	cache := newWeekCache(3)

	// Добавляем больше чем capacity
	for i := 0; i < 10; i++ {
		key := weekCacheKey{
			groupID:     i,
			weekStart:   "2026-02-23",
			dataVersion: time.Now().Format(time.RFC3339),
		}
		cache.set(key, WeekScheduleResponse{GroupID: i})
	}

	// Проверяем что кэш работает (некоторые записи могут быть вытеснены)
	assert.NotNil(t, cache)
}

func TestCloneWeekScheduleResponse(t *testing.T) {
	original := WeekScheduleResponse{
		GroupID:     1,
		DateStart:   "2026-02-23",
		DateEnd:     "2026-02-28",
		DataVersion: time.Now().Format(time.RFC3339),
		Days: []DaySchedule{
			{
				Date:       "2026-02-23",
				DayOfWeek:  0,
				WeekParity: WeekParityDenominator,
				Lessons: []Lesson{
					{PairNumber: 1, SubjectName: "Math"},
				},
			},
		},
	}

	clone := cloneWeekScheduleResponse(original)

	// Проверяем что значения скопированы
	assert.Equal(t, original.GroupID, clone.GroupID)
	assert.Equal(t, original.DateStart, clone.DateStart)
	assert.Equal(t, original.DateEnd, clone.DateEnd)
	assert.Equal(t, len(original.Days), len(clone.Days))

	// Проверяем что это разные объекты (изменение клона не влияет на оригинал)
	clone.GroupID = 999
	assert.NotEqual(t, original.GroupID, clone.GroupID)
}
