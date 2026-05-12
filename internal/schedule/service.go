package schedule

import (
	"fmt"
	"time"
)

type ServiceDeps struct {
	Repo              *Repository
	SemesterStartDate string
	Now               func() time.Time
}

type Service struct {
	repo              *Repository
	semesterStartDate time.Time
	now               func() time.Time
	weekCache         *weekCache
}

func NewService(deps ServiceDeps) *Service {
	start, _ := time.Parse("2006-01-02", deps.SemesterStartDate)
	return &Service{
		repo:              deps.Repo,
		semesterStartDate: start,
		now:               deps.Now,
		weekCache:         newWeekCache(256),
	}
}

type Lesson struct {
	PairNumber   int16   `json:"pair_number"`
	SubjectID    *int    `json:"subject_id"`
	SubjectName  string  `json:"subject_name"`
	LocationID   *int    `json:"location_id"`
	LocationName string  `json:"location_name"`
	TeacherName  string  `json:"teacher_name"`
	Subgroup     *int16  `json:"subgroup"` // nil/0=вся группа
	FlowKey      *string `json:"flow_key,omitempty"`
	IsChanged    bool    `json:"is_changed"`
	IsAdded      bool    `json:"is_added"`
	Comment      *string `json:"comment"`
}

type DaySchedule struct {
	Date        string     `json:"date"`        // YYYY-MM-DD
	DayOfWeek   int16      `json:"day_of_week"` // 0=Пн
	WeekParity  WeekParity `json:"week_parity"`
	OverlayText *string    `json:"overlay_text"`
	Events      []DayEvent `json:"events"`
	Lessons     []Lesson   `json:"lessons"`
}

type DayEvent struct {
	ID           int64   `json:"id"`
	EventType    string  `json:"event_type"`
	Title        string  `json:"title"`
	Details      *string `json:"details"`
	LocationID   *int    `json:"location_id"`
	LocationName *string `json:"location_name"`
}

type WeekScheduleResponse struct {
	GroupID     int           `json:"group_id"`
	DateStart   string        `json:"date_start"`
	DateEnd     string        `json:"date_end"`
	DataVersion string        `json:"data_version"`
	Days        []DaySchedule `json:"days"`
}

func (s *Service) GetCurrentWeek(groupID int, refDate time.Time) (*WeekScheduleResponse, error) {
	start := mondayOfWeek(refDate)
	end := start.AddDate(0, 0, 5) // Пн..Сб
	return s.GetRange(groupID, start, end)
}

func (s *Service) GetRange(groupID int, startDate, endDate time.Time) (*WeekScheduleResponse, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("date_end before date_start")
	}

	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}
	dataVersion := state.ScheduleVersion.UTC().Format(time.RFC3339)

	// Week cache: only for canonical Mon..Sat weeks.
	weekStart := mondayOfWeek(startDate)
	if startDate.Equal(weekStart) && endDate.Equal(weekStart.AddDate(0, 0, 5)) {
		k := weekCacheKey{groupID: groupID, weekStart: weekStart.Format("2006-01-02"), dataVersion: dataVersion}
		if cached, ok := s.weekCache.get(k); ok {
			clone := cloneWeekScheduleResponse(cached)
			return &clone, nil
		}
	}

	days, err := s.buildDays(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	resp := WeekScheduleResponse{
		GroupID:     groupID,
		DateStart:   startDate.Format("2006-01-02"),
		DateEnd:     endDate.Format("2006-01-02"),
		DataVersion: dataVersion,
		Days:        days,
	}

	// Save to cache for canonical Mon..Sat weeks.
	weekStart = mondayOfWeek(startDate)
	if startDate.Equal(weekStart) && endDate.Equal(weekStart.AddDate(0, 0, 5)) {
		k := weekCacheKey{groupID: groupID, weekStart: weekStart.Format("2006-01-02"), dataVersion: dataVersion}
		s.weekCache.set(k, cloneWeekScheduleResponse(resp))
	}

	return &resp, nil
}
