package schedule

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ScheduleViewFilter struct {
	Scope       string
	GroupID     *int
	TeacherID   *int
	TeacherName *string
	LocationID  *int
}

type ScheduleViewResponse struct {
	Scope        string            `json:"scope"`
	DateStart    string            `json:"date_start"`
	DateEnd      string            `json:"date_end"`
	DataVersion  string            `json:"data_version"`
	GroupID      *int              `json:"group_id,omitempty"`
	GroupName    *string           `json:"group_name,omitempty"`
	TeacherID    *int              `json:"teacher_id,omitempty"`
	TeacherName  *string           `json:"teacher_name,omitempty"`
	LocationID   *int              `json:"location_id,omitempty"`
	LocationName *string           `json:"location_name,omitempty"`
	Days         []ScheduleViewDay `json:"days"`
}

type ScheduleViewDay struct {
	Date       string               `json:"date"`
	DayOfWeek  int16                `json:"day_of_week"`
	WeekParity WeekParity           `json:"week_parity"`
	Lessons    []ScheduleViewLesson `json:"lessons"`
}

type ScheduleViewLesson struct {
	GroupID      int     `json:"group_id"`
	GroupName    string  `json:"group_name"`
	PairNumber   int16   `json:"pair_number"`
	SubjectID    *int    `json:"subject_id"`
	SubjectName  string  `json:"subject_name"`
	LocationID   *int    `json:"location_id"`
	LocationName string  `json:"location_name"`
	LessonFormat string  `json:"lesson_format"`
	TeacherName  string  `json:"teacher_name"`
	Subgroup     *int16  `json:"subgroup"`
	IsChanged    bool    `json:"is_changed"`
	IsAdded      bool    `json:"is_added"`
	Comment      *string `json:"comment"`
}

func (s *Service) GetScheduleView(filter ScheduleViewFilter, startDate, endDate time.Time) (*ScheduleViewResponse, error) {
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("date_end before date_start")
	}

	scope := normalizeScheduleViewScope(filter.Scope)
	if scope == "" {
		return nil, fmt.Errorf("scope must be group, teacher, or location")
	}

	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}
	resp := &ScheduleViewResponse{
		Scope:       scope,
		DateStart:   startDate.Format("2006-01-02"),
		DateEnd:     endDate.Format("2006-01-02"),
		DataVersion: state.ScheduleVersion.UTC().Format(time.RFC3339),
	}

	switch scope {
	case "group":
		return s.getGroupScheduleView(resp, filter, startDate, endDate)
	case "teacher":
		return s.getTeacherScheduleView(resp, filter, startDate, endDate)
	case "location":
		return s.getLocationScheduleView(resp, filter, startDate, endDate)
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
}

func (s *Service) getGroupScheduleView(resp *ScheduleViewResponse, filter ScheduleViewFilter, startDate, endDate time.Time) (*ScheduleViewResponse, error) {
	if filter.GroupID == nil || *filter.GroupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	group, err := s.repo.GetGroup(*filter.GroupID)
	if err != nil {
		return nil, err
	}
	groupID := group.ID
	groupName := group.Name
	resp.GroupID = &groupID
	resp.GroupName = &groupName

	week, err := s.GetRange(group.ID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	resp.DataVersion = week.DataVersion
	resp.Days = viewDaysFromWeek(*week, *group, func(Lesson) bool { return true })
	return resp, nil
}

func (s *Service) getTeacherScheduleView(resp *ScheduleViewResponse, filter ScheduleViewFilter, startDate, endDate time.Time) (*ScheduleViewResponse, error) {
	teacherName, teacherID, err := s.resolveScheduleViewTeacher(filter)
	if err != nil {
		return nil, err
	}
	resp.TeacherName = &teacherName
	resp.TeacherID = teacherID

	return s.getAggregatedScheduleView(resp, startDate, endDate, func(lesson Lesson) bool {
		return strings.EqualFold(strings.TrimSpace(lesson.TeacherName), teacherName)
	})
}

func (s *Service) getLocationScheduleView(resp *ScheduleViewResponse, filter ScheduleViewFilter, startDate, endDate time.Time) (*ScheduleViewResponse, error) {
	if filter.LocationID == nil || *filter.LocationID <= 0 {
		return nil, fmt.Errorf("location_id required")
	}
	location, err := s.repo.GetLocation(*filter.LocationID)
	if err != nil {
		return nil, err
	}
	locationID := location.ID
	locationName := location.Name
	resp.LocationID = &locationID
	resp.LocationName = &locationName

	return s.getAggregatedScheduleView(resp, startDate, endDate, func(lesson Lesson) bool {
		return lesson.LocationID != nil && *lesson.LocationID == locationID
	})
}

func (s *Service) resolveScheduleViewTeacher(filter ScheduleViewFilter) (string, *int, error) {
	if filter.TeacherID != nil && *filter.TeacherID > 0 {
		teacher, err := s.repo.GetTeacher(*filter.TeacherID)
		if err != nil {
			return "", nil, err
		}
		teacherID := teacher.ID
		return strings.TrimSpace(teacher.Name), &teacherID, nil
	}
	if filter.TeacherName == nil || strings.TrimSpace(*filter.TeacherName) == "" {
		return "", nil, fmt.Errorf("teacher_id or teacher_name required")
	}
	return strings.TrimSpace(*filter.TeacherName), nil, nil
}

func (s *Service) getAggregatedScheduleView(resp *ScheduleViewResponse, startDate, endDate time.Time, matches func(Lesson) bool) (*ScheduleViewResponse, error) {
	days, byDate, err := s.emptyScheduleViewDays(startDate, endDate)
	if err != nil {
		return nil, err
	}

	groups, err := s.repo.ListGroups()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		week, err := s.GetRange(group.ID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		for _, day := range viewDaysFromWeek(*week, group, matches) {
			targetIdx, ok := byDate[day.Date]
			if !ok {
				continue
			}
			days[targetIdx].Lessons = append(days[targetIdx].Lessons, day.Lessons...)
		}
	}

	for i := range days {
		sortScheduleViewLessons(days[i].Lessons)
	}
	resp.Days = days
	return resp, nil
}

func (s *Service) emptyScheduleViewDays(startDate, endDate time.Time) ([]ScheduleViewDay, map[string]int, error) {
	days := make([]ScheduleViewDay, 0)
	byDate := map[string]int{}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		days = append(days, ScheduleViewDay{
			Date:       dateKey,
			DayOfWeek:  dayOfWeekForDate(d, nil),
			WeekParity: s.weekParityForDate(d),
			Lessons:    []ScheduleViewLesson{},
		})
		byDate[dateKey] = len(days) - 1
	}
	return days, byDate, nil
}

func viewDaysFromWeek(week WeekScheduleResponse, group Group, matches func(Lesson) bool) []ScheduleViewDay {
	out := make([]ScheduleViewDay, 0, len(week.Days))
	for _, day := range week.Days {
		viewDay := ScheduleViewDay{
			Date:       day.Date,
			DayOfWeek:  day.DayOfWeek,
			WeekParity: day.WeekParity,
			Lessons:    []ScheduleViewLesson{},
		}
		for _, lesson := range day.Lessons {
			if !matches(lesson) {
				continue
			}
			viewDay.Lessons = append(viewDay.Lessons, ScheduleViewLesson{
				GroupID:      group.ID,
				GroupName:    group.Name,
				PairNumber:   lesson.PairNumber,
				SubjectID:    lesson.SubjectID,
				SubjectName:  lesson.SubjectName,
				LocationID:   lesson.LocationID,
				LocationName: lesson.LocationName,
				LessonFormat: lesson.LessonFormat,
				TeacherName:  lesson.TeacherName,
				Subgroup:     lesson.Subgroup,
				IsChanged:    lesson.IsChanged,
				IsAdded:      lesson.IsAdded,
				Comment:      lesson.Comment,
			})
		}
		sortScheduleViewLessons(viewDay.Lessons)
		out = append(out, viewDay)
	}
	return out
}

func sortScheduleViewLessons(lessons []ScheduleViewLesson) {
	sort.SliceStable(lessons, func(i, j int) bool {
		if lessons[i].PairNumber != lessons[j].PairNumber {
			return lessons[i].PairNumber < lessons[j].PairNumber
		}
		if lessons[i].GroupName != lessons[j].GroupName {
			return lessons[i].GroupName < lessons[j].GroupName
		}
		return subgroupKey(lessons[i].Subgroup) < subgroupKey(lessons[j].Subgroup)
	})
}

func normalizeScheduleViewScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "group", "groups", "student", "students":
		return "group"
	case "teacher", "teachers":
		return "teacher"
	case "location", "locations", "room", "rooms", "auditorium", "auditoriums":
		return "location"
	default:
		return ""
	}
}
