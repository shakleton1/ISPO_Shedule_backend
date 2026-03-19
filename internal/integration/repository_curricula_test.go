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

func shortCurriculaSpecialtyCode(prefix string) string {
	return fmt.Sprintf("%s-%05d", prefix, time.Now().Unix()%100000)
}

func TestRepositoryCurricula_SpecialtyCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	s := &schedule.Specialty{Code: shortCurriculaSpecialtyCode("SPC"), Name: "Specialty"}
	require.NoError(t, repo.CreateSpecialty(s))
	t.Cleanup(func() { _ = db.Where("id = ?", s.ID).Delete(&schedule.Specialty{}).Error })

	_, err := repo.UpdateSpecialty(s.ID, &schedule.Specialty{Code: s.Code + "-U", Name: "Specialty Updated"})
	require.NoError(t, err)

	list, err := repo.ListSpecialties()
	require.NoError(t, err)
	assert.NotEmpty(t, list)

	require.NoError(t, repo.DeleteSpecialty(s.ID))
}

func TestRepositoryCurricula_CurriculumCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	s := &schedule.Specialty{Code: shortCurriculaSpecialtyCode("CSP"), Name: "Cur Spec"}
	require.NoError(t, db.Create(s).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", s.ID).Delete(&schedule.Specialty{}).Error })

	c := &schedule.Curriculum{SpecialtyID: s.ID, AdmissionYear: 2024, Variant: "A", Title: "Curr", IsActive: true}
	require.NoError(t, repo.CreateCurriculum(c))
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", c.ID).Error })

	updated, err := repo.UpdateCurriculum(c.ID, &schedule.Curriculum{SpecialtyID: s.ID, AdmissionYear: 2024, Variant: "B", Title: "Curr Updated", IsActive: false})
	require.NoError(t, err)
	assert.Equal(t, "B", updated.Variant)

	list, err := repo.ListCurricula(schedule.CurriculumFilters{SpecialtyID: &s.ID})
	require.NoError(t, err)
	assert.NotEmpty(t, list)

	require.NoError(t, repo.DeleteCurriculum(c.ID))
}

func TestRepositoryCurricula_AcademicCalendarsAndWeeks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	s := &schedule.Specialty{Code: shortCurriculaSpecialtyCode("AWS"), Name: "ACW Spec"}
	require.NoError(t, db.Create(s).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", s.ID).Delete(&schedule.Specialty{}).Error })

	c := &schedule.Curriculum{SpecialtyID: s.ID, AdmissionYear: 2025, Variant: "A", Title: "ACW Curr", IsActive: true}
	require.NoError(t, db.Create(c).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", c.ID).Error })

	ac := &schedule.AcademicCalendar{CurriculumID: c.ID, AcademicYearStart: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), WeeksTotal: 52}
	require.NoError(t, repo.CreateAcademicCalendar(ac))
	t.Cleanup(func() { _ = db.Where("id = ?", ac.ID).Delete(&schedule.AcademicCalendar{}).Error })

	calendars, err := repo.ListAcademicCalendars(c.ID)
	require.NoError(t, err)
	require.NotEmpty(t, calendars)

	weeks, err := repo.UpsertAcademicCalendarWeeks(ac.ID, []schedule.AcademicCalendarWeek{{WeekNumber: 1, WeekStartDate: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), ActivityCode: "EDU", IsTeaching: true}})
	require.NoError(t, err)
	require.NotEmpty(t, weeks)

	listedWeeks, err := repo.ListAcademicCalendarWeeks(ac.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, listedWeeks)

	require.NoError(t, repo.DeleteAcademicCalendar(ac.ID))
}

func TestRepositoryCurricula_CurriculumItemsAndAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	spec := &schedule.Specialty{Code: shortCurriculaSpecialtyCode("CIA"), Name: "CIA Spec"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2026, Variant: "A", Title: "CIA Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID).Error })

	subject := &schedule.Subject{Name: fmt.Sprintf("CIA-subject-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() { _ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error })

	item := &schedule.CurriculumItem{CurriculumID: curr.ID, ItemType: "DISCIPLINE", Name: "CIA Item", SubjectID: &subject.ID}
	require.NoError(t, repo.CreateCurriculumItem(item))
	t.Cleanup(func() { _ = db.Where("id = ?", item.ID).Delete(&schedule.CurriculumItem{}).Error })

	updatedItem, err := repo.UpdateCurriculumItem(item.ID, &schedule.CurriculumItem{CurriculumID: curr.ID, ItemType: "DISCIPLINE", Name: "CIA Item Updated", SubjectID: &subject.ID})
	require.NoError(t, err)
	assert.Equal(t, "CIA Item Updated", updatedItem.Name)

	items, err := repo.ListCurriculumItems(curr.ID)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	allocs, err := repo.UpsertCurriculumItemAllocations(item.ID, []schedule.CurriculumItemAllocation{{Semester: 1, AssessmentType: ptrString("EXAM")}})
	require.NoError(t, err)
	require.NotEmpty(t, allocs)

	listedAllocs, err := repo.ListCurriculumItemAllocations(item.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, listedAllocs)

	require.NoError(t, repo.DeleteCurriculumItem(item.ID))
}

func ptrString(v string) *string { return &v }
