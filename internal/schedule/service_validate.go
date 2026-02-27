package schedule

import (
	"time"
)

type ScheduleValidationWarning struct {
	Date        string `json:"date"`
	PairNumber  int16  `json:"pair_number"`
	Subgroup    *int16 `json:"subgroup"`
	SubjectID   int    `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Semester    int16  `json:"semester"`
}

type ScheduleValidationResponse struct {
	GroupID      int                         `json:"group_id"`
	StartDate    string                      `json:"start_date"`
	EndDate      string                      `json:"end_date"`
	Warnings     []ScheduleValidationWarning `json:"warnings"`
	WarnCount    int                         `json:"warn_count"`
	Validated    bool                        `json:"validated"`
	NoCurriculum bool                        `json:"no_curriculum"`
}

func (s *Service) ValidateScheduleRange(groupID int, startDate, endDate time.Time) (*ScheduleValidationResponse, error) {
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)

	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	resp := &ScheduleValidationResponse{
		GroupID:      groupID,
		StartDate:    startDate.Format("2006-01-02"),
		EndDate:      endDate.Format("2006-01-02"),
		Warnings:     []ScheduleValidationWarning{},
		Validated:    false,
		NoCurriculum: false,
	}

	if group.CurriculumID == nil {
		resp.NoCurriculum = true
		return resp, nil
	}

	days, err := s.buildDays(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Gather semesters we can infer for the requested date range.
	semSet := map[int16]bool{}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		sem := inferSemesterForDate(d, group.Course)
		if sem != nil {
			semSet[*sem] = true
		}
	}
	semList := make([]int16, 0, len(semSet))
	for sem := range semSet {
		semList = append(semList, sem)
	}

	allowedBySem, err := s.repo.ListAllocatedSubjectsBySemester(*group.CurriculumID, semList)
	if err != nil {
		return nil, err
	}

	for _, day := range days {
		d, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			continue
		}
		sem := inferSemesterForDate(d, group.Course)
		if sem == nil {
			continue
		}
		allowed := allowedBySem[*sem]
		for _, l := range day.Lessons {
			if l.SubjectID == nil {
				continue
			}
			if allowed != nil && allowed[*l.SubjectID] {
				continue
			}
			// If allowed is nil/empty: treat as not allocated.
			resp.Warnings = append(resp.Warnings, ScheduleValidationWarning{
				Date:        day.Date,
				PairNumber:  l.PairNumber,
				Subgroup:    l.Subgroup,
				SubjectID:   *l.SubjectID,
				SubjectName: l.SubjectName,
				Semester:    *sem,
			})
		}
	}

	resp.WarnCount = len(resp.Warnings)
	resp.Validated = true
	return resp, nil
}
