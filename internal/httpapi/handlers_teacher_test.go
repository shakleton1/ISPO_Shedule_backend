package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ispo-schedule/internal/schedule"
)

func TestBuildTeacherWorkload_DeduplicatesStreamSlots(t *testing.T) {
	t.Parallel()

	teacherID := 7
	teacherName := "Тузова Д.А."
	subjectID := 11
	locationID := 403
	view := schedule.ScheduleViewResponse{
		TeacherID:   &teacherID,
		TeacherName: &teacherName,
		DateStart:   "2026-09-07",
		DateEnd:     "2026-09-20",
		Days: []schedule.ScheduleViewDay{
			{
				Date: "2026-09-07",
				Lessons: []schedule.ScheduleViewLesson{
					{
						GroupID:      1,
						GroupName:    "22290907/1095",
						PairNumber:   2,
						SubjectID:    &subjectID,
						SubjectName:  "МДК.02.02 Инстр. средства разр. ПО",
						LocationID:   &locationID,
						LocationName: "403",
						TeacherName:  teacherName,
					},
					{
						GroupID:      2,
						GroupName:    "22290907/1096",
						PairNumber:   2,
						SubjectID:    &subjectID,
						SubjectName:  "МДК.02.02 Инстр. средства разр. ПО",
						LocationID:   &locationID,
						LocationName: "403",
						TeacherName:  teacherName,
					},
				},
			},
		},
	}

	out := buildTeacherWorkload(view)

	assert.Equal(t, teacherID, out.TeacherID)
	assert.Equal(t, teacherName, out.TeacherName)
	assert.Equal(t, 1, out.TotalPairs)
	assert.Equal(t, 2, out.TotalAcademicHours)
	assert.Len(t, out.BySubject, 1)
	assert.Equal(t, 1, out.BySubject[0].Pairs)
	assert.Len(t, out.ByGroup, 2)
	assert.Len(t, out.Lessons, 2)
}
