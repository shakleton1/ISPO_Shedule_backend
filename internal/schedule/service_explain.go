package schedule

import (
	"fmt"
	"time"
)

type ExplainDecision struct {
	SemesterInferred *int16 `json:"semester_inferred"`
	NonTeaching      bool   `json:"non_teaching"`
}

type ExplainSlotResponse struct {
	GroupID     int        `json:"group_id"`
	Date        string     `json:"date"`
	DayOfWeek   int16      `json:"day_of_week"`
	WeekParity  WeekParity `json:"week_parity"`
	DataVersion string     `json:"data_version"`

	PairNumber int16  `json:"pair_number"`
	Subgroup   *int16 `json:"subgroup"`

	Lessons []Lesson `json:"lessons"`

	Decision ExplainDecision `json:"decision"`
}

func (s *Service) ExplainSlot(groupID int, date time.Time, pairNumber int16, subgroup *int16) (*ExplainSlotResponse, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	if pairNumber <= 0 {
		return nil, fmt.Errorf("pair_number required")
	}

	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}

	dayOfWeek := dayOfWeekForDate(date, nil)
	parity := s.weekParityForDate(date)

	nonTeaching := false
	studyStates, err := s.repo.ListStudyDayStatesForGroupBetween(groupID, date, date)
	if err != nil {
		return nil, err
	}
	dayKey := dateOnly(date).Format("2006-01-02")
	weekKey := mondayOfWeek(date).Format("2006-01-02")
	studyState := defaultStudyDayState()
	if state, ok := studyStates[weekKey]; ok {
		studyState = state
	}
	if state, ok := studyStates[dayKey]; ok && state.Source == "academic_day_override" {
		studyState = state
	}
	nonTeaching = !studyState.IsTeaching

	semester := inferSemesterForDate(date, group.Course)
	week, err := s.GetRange(groupID, dateOnly(date), dateOnly(date))
	if err != nil {
		return nil, err
	}
	merged := []Lesson{}
	if len(week.Days) > 0 {
		merged = week.Days[0].Lessons
	}

	filterLessons := func(in []Lesson) []Lesson {
		out := make([]Lesson, 0)
		for _, l := range in {
			if l.PairNumber != pairNumber {
				continue
			}
			// Whole-group lessons (Subgroup == nil) are relevant for any subgroup.
			if subgroup != nil && l.Subgroup != nil && *l.Subgroup != *subgroup {
				continue
			}
			out = append(out, l)
		}
		return out
	}

	return &ExplainSlotResponse{
		GroupID:     groupID,
		Date:        dateOnly(date).Format("2006-01-02"),
		DayOfWeek:   dayOfWeek,
		WeekParity:  parity,
		DataVersion: state.ScheduleVersion.UTC().Format(time.RFC3339),
		PairNumber:  pairNumber,
		Subgroup:    subgroup,
		Lessons:     filterLessons(merged),
		Decision:    ExplainDecision{SemesterInferred: semester, NonTeaching: nonTeaching},
	}, nil
}
