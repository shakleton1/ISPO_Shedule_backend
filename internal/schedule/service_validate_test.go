package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateScheduleBusinessRules_LoadsAndFloatingDayOff(t *testing.T) {
	group := Group{ID: 1, Course: 1}
	subjectID := 10
	days := []DaySchedule{
		{Date: "2026-09-07", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1, 2, 3, 4)},
		{Date: "2026-09-08", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1, 2, 3, 4)},
		{Date: "2026-09-09", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1, 2, 3, 4)},
		{Date: "2026-09-10", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1, 2, 3, 4)},
		{Date: "2026-09-11", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1, 2, 3, 4)},
		{Date: "2026-09-12", Lessons: lessonsForPairs(subjectID, "Math", "Teacher A", 1)},
	}

	warnings := validateScheduleBusinessRules(days, group, nil)
	codes := warningCodes(warnings)

	assert.Contains(t, codes, "student_week_hours_limit")
	assert.Contains(t, codes, "teacher_week_hours_limit")
	assert.Contains(t, codes, "group_floating_day_off")
	assert.Contains(t, codes, "teacher_floating_day_off")
}

func TestValidateScheduleBusinessRules_SubgroupPairsCountOnceForStudents(t *testing.T) {
	group := Group{ID: 1, Course: 1}
	subjectID := 10
	sg1 := int16(1)
	sg2 := int16(2)
	days := []DaySchedule{
		{
			Date: "2026-09-07",
			Lessons: []Lesson{
				{PairNumber: 1, SubjectID: &subjectID, SubjectName: "Math", Subgroup: &sg1},
				{PairNumber: 1, SubjectID: &subjectID, SubjectName: "Math", Subgroup: &sg2},
			},
		},
	}

	warnings := validateScheduleBusinessRules(days, group, nil)

	assert.NotContains(t, warningCodes(warnings), "student_week_hours_limit")
}

func TestValidateScheduleBusinessRules_MissingTeacherPlaceholder(t *testing.T) {
	group := Group{ID: 1, Course: 1}
	subjectID := 10
	days := []DaySchedule{
		{
			Date: "2026-09-07",
			Lessons: []Lesson{
				{PairNumber: 1, SubjectID: &subjectID, SubjectName: "Math"},
			},
		},
	}

	warnings := validateScheduleBusinessRules(days, group, nil)

	assert.Contains(t, warningCodes(warnings), "missing_teacher_placeholder")
}

func TestValidateScheduleBusinessRules_PEPracticeAndCampus(t *testing.T) {
	group := Group{ID: 1, Course: 1}
	peID := 20
	mathID := 21
	locA := 100
	locB := 101
	days := []DaySchedule{
		{
			Date: "2026-09-07",
			Lessons: []Lesson{
				{PairNumber: 1, SubjectID: &mathID, SubjectName: "Math", LocationID: &locA},
				{PairNumber: 2, SubjectID: &mathID, SubjectName: "Math", LocationID: &locB},
				{PairNumber: 3, SubjectID: &mathID, SubjectName: "Math", LocationID: &locA},
				{PairNumber: 5, SubjectID: &peID, SubjectName: "Физическая культура", LocationID: &locA},
			},
		},
	}
	meta := map[int]LocationMeta{
		locA: {ID: locA, Campus: "main", LocationKind: "classroom"},
		locB: {ID: locB, Campus: "sport", LocationKind: "gym"},
	}

	warnings := validateScheduleBusinessRules(days, group, meta)
	codes := warningCodes(warnings)

	assert.Contains(t, codes, "physical_education_fifth_pair")
	assert.Contains(t, codes, "three_pairs_requires_practice")
	assert.Contains(t, codes, "multiple_campuses_day")
}

func TestValidatePhysicalEducationFacilityRules(t *testing.T) {
	targetGroup := Group{ID: 1, Course: 1}
	otherGroups := []Group{{ID: 2, Course: 1}, {ID: 3, Course: 1}, {ID: 4, Course: 1}}
	peID := 20
	classroomID := 100
	gymID := 101
	groupsByID := map[int]Group{targetGroup.ID: targetGroup}
	for _, group := range otherGroups {
		groupsByID[group.ID] = group
	}

	allDaysByGroup := map[int][]DaySchedule{
		targetGroup.ID: {
			{
				Date: "2026-09-07",
				Lessons: []Lesson{
					{PairNumber: 1, SubjectID: &peID, SubjectName: "physical education", LocationID: &classroomID},
					{PairNumber: 2, SubjectID: &peID, SubjectName: "physical education", LocationID: &gymID},
				},
			},
		},
	}
	for _, group := range otherGroups {
		allDaysByGroup[group.ID] = []DaySchedule{
			{
				Date: "2026-09-07",
				Lessons: []Lesson{
					{PairNumber: 2, SubjectID: &peID, SubjectName: "physical education", LocationID: &gymID},
				},
			},
		}
	}
	meta := map[int]LocationMeta{
		classroomID: {ID: classroomID, LocationKind: "classroom"},
		gymID:       {ID: gymID, LocationKind: "gym"},
	}

	warnings := validatePhysicalEducationFacilityRules(targetGroup.ID, groupsByID, allDaysByGroup, meta)
	codes := warningCodes(warnings)

	assert.Contains(t, codes, "physical_education_location_kind")
	assert.Contains(t, codes, "physical_education_room_group_limit")
}

func lessonsForPairs(subjectID int, subjectName, teacherName string, pairs ...int16) []Lesson {
	lessons := make([]Lesson, 0, len(pairs))
	for _, pair := range pairs {
		lessons = append(lessons, Lesson{
			PairNumber:  pair,
			SubjectID:   &subjectID,
			SubjectName: subjectName,
			TeacherName: teacherName,
		})
	}
	return lessons
}

func warningCodes(warnings []ScheduleValidationWarning) []string {
	out := make([]string, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, w.Code)
	}
	return out
}
