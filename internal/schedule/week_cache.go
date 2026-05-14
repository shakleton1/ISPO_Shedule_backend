package schedule

import (
	"sync"
)

type weekCacheKey struct {
	groupID     int
	weekStart   string
	dataVersion string
}

type weekCache struct {
	mu         sync.Mutex
	maxEntries int
	items      map[weekCacheKey]WeekScheduleResponse
	order      []weekCacheKey
}

func newWeekCache(maxEntries int) *weekCache {
	if maxEntries <= 0 {
		maxEntries = 1
	}
	return &weekCache{
		maxEntries: maxEntries,
		items:      make(map[weekCacheKey]WeekScheduleResponse, maxEntries),
		order:      make([]weekCacheKey, 0, maxEntries),
	}
}

func (c *weekCache) get(k weekCacheKey) (WeekScheduleResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[k]
	return v, ok
}

func (c *weekCache) set(k weekCacheKey, v WeekScheduleResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[k]; exists {
		c.items[k] = v
		return
	}
	if len(c.items) >= c.maxEntries {
		old := c.order[0]
		c.order = c.order[1:]
		delete(c.items, old)
	}
	c.items[k] = v
	c.order = append(c.order, k)
}

func cloneWeekScheduleResponse(in WeekScheduleResponse) WeekScheduleResponse {
	out := in
	out.Days = make([]DaySchedule, len(in.Days))
	for i := range in.Days {
		out.Days[i] = cloneDaySchedule(in.Days[i])
	}
	return out
}

func cloneDaySchedule(in DaySchedule) DaySchedule {
	out := in
	if in.StudyDayState != nil {
		v := *in.StudyDayState
		out.StudyDayState = &v
	}
	if in.GlobalDayConstraint != nil {
		v := *in.GlobalDayConstraint
		out.GlobalDayConstraint = &v
	}
	out.Events = append([]DayEvent(nil), in.Events...)
	out.Lessons = append([]Lesson(nil), in.Lessons...)
	return out
}
