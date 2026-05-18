package httpapi

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

type teacherWorkloadResponse struct {
	TeacherID          int                         `json:"teacher_id"`
	TeacherName        string                      `json:"teacher_name"`
	DateStart          string                      `json:"date_start"`
	DateEnd            string                      `json:"date_end"`
	TotalPairs         int                         `json:"total_pairs"`
	TotalAcademicHours int                         `json:"total_academic_hours"`
	BySubject          []teacherWorkloadSubjectRow `json:"by_subject"`
	ByGroup            []teacherWorkloadGroupRow   `json:"by_group"`
	Lessons            []teacherWorkloadLessonRow  `json:"lessons"`
}

type teacherWorkloadSubjectRow struct {
	SubjectID     *int   `json:"subject_id"`
	SubjectName   string `json:"subject_name"`
	Pairs         int    `json:"pairs"`
	AcademicHours int    `json:"academic_hours"`
}

type teacherWorkloadGroupRow struct {
	GroupID       int    `json:"group_id"`
	GroupName     string `json:"group_name"`
	Pairs         int    `json:"pairs"`
	AcademicHours int    `json:"academic_hours"`
}

type teacherWorkloadLessonRow struct {
	Date         string `json:"date"`
	PairNumber   int16  `json:"pair_number"`
	GroupID      int    `json:"group_id"`
	GroupName    string `json:"group_name"`
	SubjectID    *int   `json:"subject_id"`
	SubjectName  string `json:"subject_name"`
	LocationID   *int   `json:"location_id"`
	LocationName string `json:"location_name"`
	Subgroup     *int16 `json:"subgroup"`
}

func handleTeacherSchedule(svc *schedule.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		teacherID, ok := teacherIDFromContext(c)
		if !ok {
			return
		}
		start, end, ok := parseScheduleViewDateRange(c)
		if !ok {
			return
		}
		resp, err := svc.GetScheduleView(schedule.ScheduleViewFilter{Scope: "teacher", TeacherID: &teacherID}, start, end)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func handleTeacherWorkload(svc *schedule.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		teacherID, ok := teacherIDFromContext(c)
		if !ok {
			return
		}
		start, end, ok := parseScheduleViewDateRange(c)
		if !ok {
			return
		}
		view, err := svc.GetScheduleView(schedule.ScheduleViewFilter{Scope: "teacher", TeacherID: &teacherID}, start, end)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		c.JSON(http.StatusOK, buildTeacherWorkload(*view))
	}
}

func teacherIDFromContext(c *gin.Context) (int, bool) {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		writeUnauthorized(c, "unauthorized")
		return 0, false
	}
	u, ok := v.(*auth.User)
	if !ok || u == nil {
		writeUnauthorized(c, "unauthorized")
		return 0, false
	}
	if u.TeacherID == nil || *u.TeacherID <= 0 {
		writeError(c, http.StatusForbidden, "teacher_not_linked", "teacher_id", "teacher account is not linked to teacher")
		return 0, false
	}
	return *u.TeacherID, true
}

func buildTeacherWorkload(view schedule.ScheduleViewResponse) teacherWorkloadResponse {
	teacherID := 0
	if view.TeacherID != nil {
		teacherID = *view.TeacherID
	}
	teacherName := ""
	if view.TeacherName != nil {
		teacherName = *view.TeacherName
	}

	type counter struct {
		pairs int
		name  string
		id    *int
	}
	bySubject := map[string]*counter{}
	byGroup := map[int]*counter{}
	seenSlots := map[string]bool{}
	lessons := make([]teacherWorkloadLessonRow, 0)

	for _, day := range view.Days {
		for _, lesson := range day.Lessons {
			slotKey := teacherWorkloadSlotKey(day.Date, lesson)
			if byGroup[lesson.GroupID] == nil {
				byGroup[lesson.GroupID] = &counter{name: lesson.GroupName}
			}
			byGroup[lesson.GroupID].pairs++

			if !seenSlots[slotKey] {
				seenSlots[slotKey] = true
				subjectKey := "nil:" + lesson.SubjectName
				if lesson.SubjectID != nil {
					subjectKey = fmt.Sprintf("%d", *lesson.SubjectID)
				}
				if bySubject[subjectKey] == nil {
					bySubject[subjectKey] = &counter{id: lesson.SubjectID, name: lesson.SubjectName}
				}
				bySubject[subjectKey].pairs++
			}

			lessons = append(lessons, teacherWorkloadLessonRow{
				Date:         day.Date,
				PairNumber:   lesson.PairNumber,
				GroupID:      lesson.GroupID,
				GroupName:    lesson.GroupName,
				SubjectID:    lesson.SubjectID,
				SubjectName:  lesson.SubjectName,
				LocationID:   lesson.LocationID,
				LocationName: lesson.LocationName,
				Subgroup:     lesson.Subgroup,
			})
		}
	}

	subjectRows := make([]teacherWorkloadSubjectRow, 0, len(bySubject))
	for _, row := range bySubject {
		subjectRows = append(subjectRows, teacherWorkloadSubjectRow{
			SubjectID:     row.id,
			SubjectName:   row.name,
			Pairs:         row.pairs,
			AcademicHours: row.pairs * 2,
		})
	}
	sort.Slice(subjectRows, func(i, j int) bool {
		return subjectRows[i].SubjectName < subjectRows[j].SubjectName
	})

	groupRows := make([]teacherWorkloadGroupRow, 0, len(byGroup))
	for groupID, row := range byGroup {
		groupRows = append(groupRows, teacherWorkloadGroupRow{
			GroupID:       groupID,
			GroupName:     row.name,
			Pairs:         row.pairs,
			AcademicHours: row.pairs * 2,
		})
	}
	sort.Slice(groupRows, func(i, j int) bool {
		return groupRows[i].GroupName < groupRows[j].GroupName
	})

	sort.Slice(lessons, func(i, j int) bool {
		if lessons[i].Date != lessons[j].Date {
			return lessons[i].Date < lessons[j].Date
		}
		if lessons[i].PairNumber != lessons[j].PairNumber {
			return lessons[i].PairNumber < lessons[j].PairNumber
		}
		return lessons[i].GroupName < lessons[j].GroupName
	})

	totalPairs := len(seenSlots)
	return teacherWorkloadResponse{
		TeacherID:          teacherID,
		TeacherName:        teacherName,
		DateStart:          view.DateStart,
		DateEnd:            view.DateEnd,
		TotalPairs:         totalPairs,
		TotalAcademicHours: totalPairs * 2,
		BySubject:          subjectRows,
		ByGroup:            groupRows,
		Lessons:            lessons,
	}
}

func teacherWorkloadSlotKey(date string, lesson schedule.ScheduleViewLesson) string {
	subgroup := int16(0)
	if lesson.Subgroup != nil {
		subgroup = *lesson.Subgroup
	}
	subjectID := 0
	if lesson.SubjectID != nil {
		subjectID = *lesson.SubjectID
	}
	locationID := 0
	if lesson.LocationID != nil {
		locationID = *lesson.LocationID
	}
	return fmt.Sprintf("%s|%d|%d|%d|%d", date, lesson.PairNumber, subgroup, subjectID, locationID)
}
