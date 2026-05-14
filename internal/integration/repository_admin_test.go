//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/schedule"
)

func shortSpecialtyCode(prefix string) string {
	return fmt.Sprintf("%s-%05d", prefix, time.Now().Unix()%100000)
}

func createAdminBaseData(t *testing.T) (*schedule.Repository, *schedule.Group, *schedule.Subject, *schedule.Location) {
	t.Helper()
	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := &schedule.Group{Name: fmt.Sprintf("repo-admin-group-%d", time.Now().UnixNano()), Course: 2}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	subject := &schedule.Subject{Name: fmt.Sprintf("repo-admin-subject-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() { _ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error })

	location := &schedule.Location{Name: fmt.Sprintf("repo-admin-location-%d", time.Now().UnixNano()), Kind: "physical", IsActive: true}
	require.NoError(t, db.Create(location).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", location.ID).Delete(&schedule.Location{}).Error })

	return repo, group, subject, location
}

func TestRepositoryAdmin_ListScheduleLessonViewsBetween(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	group := &schedule.Group{Name: fmt.Sprintf("tpl-group-%d", time.Now().UnixNano()), Course: 1}
	subject := &schedule.Subject{Name: fmt.Sprintf("tpl-subject-%d", time.Now().UnixNano())}
	location := &schedule.Location{Name: fmt.Sprintf("tpl-location-%d", time.Now().UnixNano()), Kind: "physical", IsActive: true}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(subject).Error)
	require.NoError(t, db.Create(location).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error
		_ = db.Where("id = ?", location.ID).Delete(&schedule.Location{}).Error
	})

	locationID := location.ID
	subjectID := subject.ID
	lessonDate := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	lesson := &schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   lessonDate,
		PairNumber:   2,
		SubjectID:    &subjectID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}
	require.NoError(t, repo.CreateScheduleLesson(lesson))
	require.NoError(t, repo.CreateRoomAssignment(&schedule.RoomAssignment{ScheduleLessonID: lesson.ID, LocationID: locationID, Source: "manual", Status: schedule.StatusPublished}))

	rows, err := repo.ListScheduleLessonViewsBetween([]int{group.ID}, lessonDate, lessonDate, false)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, int16(2), rows[0].PairNumber)
	require.NotNil(t, rows[0].SubjectID)
	assert.Equal(t, subject.ID, *rows[0].SubjectID)
	require.NotNil(t, rows[0].LocationID)
	assert.Equal(t, location.ID, *rows[0].LocationID)
}

func TestRepositoryAdmin_OverridesAndOverlaysBetween(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo, group, subject, location := createAdminBaseData(t)
	db := repo.DB()
	targetDate := time.Now().UTC().AddDate(0, 0, 1)
	comment := "repo-admin-override"
	overlayText := "special day"
	subjectID := subject.ID
	lesson := &schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   targetDate,
		PairNumber:   3,
		SubjectID:    &subjectID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}
	require.NoError(t, repo.CreateScheduleLesson(lesson))
	t.Cleanup(func() { _ = db.Where("id = ?", lesson.ID).Delete(&schedule.ScheduleLesson{}).Error })

	o := &schedule.ScheduleOverride{
		ScheduleLessonID:      &lesson.ID,
		GroupID:               group.ID,
		LessonDate:            targetDate,
		PairNumber:            3,
		ActionType:            schedule.OverrideReplace,
		ReplacementSubjectID:  &subject.ID,
		ReplacementLocationID: &location.ID,
		Reason:                &comment,
		Status:                "applied",
	}
	require.NoError(t, db.Create(o).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", o.ID).Delete(&schedule.ScheduleOverride{}).Error })

	ov, err := repo.UpsertOverlay(group.ID, targetDate, overlayText, "warn")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Where("id = ?", ov.ID).Delete(&schedule.ScheduleDayOverlay{}).Error })

	start := targetDate.AddDate(0, 0, -1)
	end := targetDate.AddDate(0, 0, 1)

	overrides, err := repo.ListScheduleOverrides(schedule.ScheduleOverrideFilters{GroupID: &group.ID, StartDate: &start, EndDate: &end})
	require.NoError(t, err)
	require.NotEmpty(t, overrides)
	assert.Equal(t, int16(3), overrides[0].PairNumber)

	overlays, err := repo.ListOverlaysBetween(group.ID, start, end)
	require.NoError(t, err)
	require.NotEmpty(t, overlays)
	assert.Equal(t, overlayText, overlays[0].Text)
}

func TestRepositoryAdmin_DayEvents_CRUDBetweenAndPaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo, group, _, location := createAdminBaseData(t)
	targetDate := time.Now().UTC().AddDate(0, 0, 2)
	details := "exam details"
	e := &schedule.ScheduleDayEvent{
		TargetDate: targetDate,
		GroupID:    group.ID,
		EventType:  "EXAM",
		Title:      "Math exam",
		Details:    &details,
		LocationID: &location.ID,
	}
	require.NoError(t, repo.CreateDayEvent(e))
	t.Cleanup(func() { _ = repo.DB().Where("id = ?", e.ID).Delete(&schedule.ScheduleDayEvent{}).Error })

	between, err := repo.ListDayEventsBetween(group.ID, targetDate.AddDate(0, 0, -1), targetDate.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.NotEmpty(t, between)
	assert.Equal(t, "Math exam", between[0].Title)

	newTitle := "Math exam moved"
	updated, err := repo.UpdateDayEvent(e.ID, &schedule.ScheduleDayEvent{
		TargetDate: targetDate,
		GroupID:    group.ID,
		EventType:  "EXAM",
		Title:      newTitle,
		Details:    &details,
		LocationID: &location.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, newTitle, updated.Title)

	limit, offset := 10, 0
	paged, err := repo.ListDayEventsPaged(schedule.DayEventFilters{GroupID: &group.ID}, &limit, &offset)
	require.NoError(t, err)
	require.NotEmpty(t, paged)
	assert.Equal(t, e.ID, paged[0].ID)

	require.NoError(t, repo.DeleteDayEvent(e.ID))
	afterDelete, err := repo.ListDayEvents(schedule.DayEventFilters{GroupID: &group.ID})
	require.NoError(t, err)
	assert.Empty(t, afterDelete)
}

func TestRepositoryAdmin_ListTeachingWeeksForGroupBetween(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	spec := &schedule.Specialty{Code: shortSpecialtyCode("SP"), Name: "Spec"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2025, Variant: "A", Title: "Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID).Error })

	group := &schedule.Group{Name: fmt.Sprintf("teach-weeks-group-%d", time.Now().UnixNano()), Course: 1, CurriculumID: &curr.ID}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	cal := &schedule.AcademicCalendar{CurriculumID: curr.ID, AcademicYearStart: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), WeeksTotal: 52}
	require.NoError(t, db.Create(cal).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", cal.ID).Delete(&schedule.AcademicCalendar{}).Error })

	weekStart := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
	week := &schedule.AcademicCalendarWeek{CalendarID: cal.ID, WeekNumber: 2, WeekStartDate: weekStart, ActivityCode: "EDU", IsTeaching: true}
	require.NoError(t, db.Create(week).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", week.ID).Delete(&schedule.AcademicCalendarWeek{}).Error })

	result, err := repo.ListTeachingWeeksForGroupBetween(group.ID, weekStart, weekStart)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	assert.Equal(t, true, result[weekStart.Format("2006-01-02")])
}

func TestRepositoryAdmin_ListCourseAssignmentTeachersForGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := &schedule.Group{Name: fmt.Sprintf("ca-group-%d", time.Now().UnixNano()), Course: 1}
	subject := &schedule.Subject{Name: fmt.Sprintf("ca-subject-%d", time.Now().UnixNano())}
	teacher := &schedule.Teacher{Name: fmt.Sprintf("ca-teacher-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(subject).Error)
	require.NoError(t, db.Create(teacher).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.CourseAssignment{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error
		_ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error
	})

	assignment := &schedule.CourseAssignment{GroupID: group.ID, Semester: 1, SubjectID: subject.ID, Status: schedule.StatusPublished, TeacherID: &teacher.ID}
	require.NoError(t, repo.CreateCourseAssignment(assignment))

	rows, err := repo.ListCourseAssignmentTeachersForGroup(group.ID)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	require.NotNil(t, rows[0].TeacherName)
	assert.Equal(t, teacher.Name, *rows[0].TeacherName)
}

func TestRepositoryAdmin_GroupSubjectLocationTeacherCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := &schedule.Group{Name: fmt.Sprintf("crud-group-%d", time.Now().UnixNano()), Course: 3}
	require.NoError(t, repo.CreateGroup(group))
	gotGroup, err := repo.GetGroup(group.ID)
	require.NoError(t, err)
	assert.Equal(t, group.Name, gotGroup.Name)

	groupPatch := &schedule.Group{Name: group.Name + "-upd", Course: 4}
	updatedGroup, err := repo.UpdateGroup(group.ID, groupPatch)
	require.NoError(t, err)
	assert.Equal(t, 4, updatedGroup.Course)
	require.NoError(t, repo.DeleteGroup(group.ID))

	subject := &schedule.Subject{Name: fmt.Sprintf("crud-subject-%d", time.Now().UnixNano())}
	require.NoError(t, repo.CreateSubject(subject))
	_, err = repo.UpdateSubject(subject.ID, &schedule.Subject{Name: subject.Name + "-upd", ShortName: "UPD"})
	require.NoError(t, err)
	require.NoError(t, repo.DeleteSubject(subject.ID))

	var deletedSubject schedule.Subject
	require.NoError(t, db.Unscoped().First(&deletedSubject, subject.ID).Error)
	assert.NotZero(t, deletedSubject.UpdatedAt)

	location := &schedule.Location{Name: fmt.Sprintf("crud-location-%d", time.Now().UnixNano())}
	require.NoError(t, repo.CreateLocation(location))
	updatedLocation, err := repo.UpdateLocation(location.ID, &schedule.Location{Name: location.Name + "-upd", Kind: "virtual", IsActive: true})
	require.NoError(t, err)
	assert.Equal(t, "virtual", updatedLocation.Kind)
	require.NoError(t, repo.DeleteLocation(location.ID))

	teacher := &schedule.Teacher{Name: fmt.Sprintf("crud-teacher-%d", time.Now().UnixNano())}
	require.NoError(t, repo.CreateTeacher(teacher))
	updatedTeacher, err := repo.UpdateTeacher(teacher.ID, &schedule.Teacher{Name: teacher.Name + "-upd"})
	require.NoError(t, err)
	assert.Equal(t, teacher.Name+"-upd", updatedTeacher.Name)
	require.NoError(t, repo.DeleteTeacher(teacher.ID))
}

func TestRepositoryAdmin_ListAllocatedSubjectsBySemester(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	spec := &schedule.Specialty{Code: shortSpecialtyCode("LAS"), Name: "LAS Spec"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "B", Title: "LAS Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID).Error })

	subject := &schedule.Subject{Name: fmt.Sprintf("las-subject-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() { _ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error })

	item := &schedule.CurriculumItem{CurriculumID: curr.ID, ItemType: "DISCIPLINE", Name: "Item", SubjectID: &subject.ID}
	require.NoError(t, db.Create(item).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", item.ID).Delete(&schedule.CurriculumItem{}).Error })

	alloc := &schedule.CurriculumItemAllocation{ItemID: item.ID, Semester: 1}
	require.NoError(t, db.Create(alloc).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", alloc.ID).Delete(&schedule.CurriculumItemAllocation{}).Error })

	out, err := repo.ListAllocatedSubjectsBySemester(curr.ID, []int16{1, 2})
	require.NoError(t, err)
	require.Contains(t, out, int16(1))
	assert.True(t, out[1][subject.ID])
}
