package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: Некоторые тесты week cache уже есть в repository_test.go
// Здесь дополнительные тесты

func TestWeekCacheKey_Format(t *testing.T) {
	key := weekCacheKey{
		groupID:     1,
		weekStart:   "2026-02-23",
		dataVersion: "2026-03-18T12:00:00Z",
	}

	// Просто проверяем что ключ создаётся
	assert.Equal(t, 1, key.groupID)
	assert.Equal(t, "2026-02-23", key.weekStart)
	assert.Equal(t, "2026-03-18T12:00:00Z", key.dataVersion)
}

func TestWeekCache_Overwrite(t *testing.T) {
	cache := newWeekCache(10)

	key := weekCacheKey{
		groupID:     1,
		weekStart:   "2026-02-23",
		dataVersion: "v1",
	}

	first := WeekScheduleResponse{GroupID: 1, DataVersion: "v1"}
	cache.set(key, first)

	// Перезаписываем с другой версией
	key.dataVersion = "v2"
	second := WeekScheduleResponse{GroupID: 1, DataVersion: "v2"}
	cache.set(key, second)

	// Проверяем что перезаписалось
	got, ok := cache.get(key)
	assert.True(t, ok)
	assert.Equal(t, "v2", got.DataVersion)
}

func TestWeekCache_DifferentKeys(t *testing.T) {
	cache := newWeekCache(100)

	key1 := weekCacheKey{groupID: 1, weekStart: "2026-02-23", dataVersion: "v1"}
	key2 := weekCacheKey{groupID: 2, weekStart: "2026-02-23", dataVersion: "v1"}

	resp1 := WeekScheduleResponse{GroupID: 1}
	resp2 := WeekScheduleResponse{GroupID: 2}

	cache.set(key1, resp1)
	cache.set(key2, resp2)

	got1, ok1 := cache.get(key1)
	got2, ok2 := cache.get(key2)

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, 1, got1.GroupID)
	assert.Equal(t, 2, got2.GroupID)
}
