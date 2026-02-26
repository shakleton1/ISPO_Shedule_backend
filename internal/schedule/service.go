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
}

func NewService(deps ServiceDeps) *Service {
	start, _ := time.Parse("2006-01-02", deps.SemesterStartDate)
	return &Service{
		repo:              deps.Repo,
		semesterStartDate: start,
		now:               deps.Now,
	}
}

type Lesson struct {
	PairNumber  int16   `json:"pair_number"`
	SubjectID   *int    `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	LocationID  *int    `json:"location_id"`
	LocationName string `json:"location_name"`
	TeacherName string  `json:"teacher_name"`
	Subgroup    *int16  `json:"subgroup"` // nil/0=вся группа
	IsChanged   bool    `json:"is_changed"`
	IsAdded     bool    `json:"is_added"`
	Comment     *string `json:"comment"`
}

type DaySchedule struct {
	Date        string   `json:"date"` // YYYY-MM-DD
	DayOfWeek   int16    `json:"day_of_week"` // 0=Пн
	WeekParity  WeekParity `json:"week_parity"`
	OverlayText *string  `json:"overlay_text"`
	Lessons     []Lesson `json:"lessons"`
}

type WeekScheduleResponse struct {
	GroupID     int          `json:"group_id"`
	DateStart   string       `json:"date_start"`
	DateEnd     string       `json:"date_end"`
	DataVersion string       `json:"data_version"`
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
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("date_end before date_start")
	}

	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}

	days, err := s.buildDays(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &WeekScheduleResponse{
		GroupID:     groupID,
		DateStart:   startDate.Format("2006-01-02"),
		DateEnd:     endDate.Format("2006-01-02"),
		DataVersion: state.ScheduleVersion.UTC().Format(time.RFC3339),
		Days:        days,
	}, nil
}
