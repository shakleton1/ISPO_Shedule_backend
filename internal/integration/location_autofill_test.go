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

func TestRepositoryPlanning_LocationWeekAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	room := &schedule.Location{
		Name:         "Avail Room " + fmt.Sprint(time.Now().UnixNano()),
		Campus:       "main",
		LocationKind: "computer_lab",
	}
	require.NoError(t, repo.CreateLocation(room))
	t.Cleanup(func() { _ = db.Where("id = ?", room.ID).Delete(&schedule.Location{}).Error })

	disabled := &schedule.Location{
		Name:         "Disabled Room " + fmt.Sprint(time.Now().UnixNano()),
		Campus:       "main",
		LocationKind: "computer_lab",
	}
	require.NoError(t, repo.CreateLocation(disabled))
	t.Cleanup(func() { _ = db.Where("id = ?", disabled.ID).Delete(&schedule.Location{}).Error })

	weekStart := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	rows, err := repo.UpsertLocationWeekAvailability(weekStart, []schedule.LocationWeekAvailability{
		{LocationID: room.ID, IsAvailable: true, Comment: ptrString("ready")},
		{LocationID: disabled.ID, IsAvailable: false},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	t.Cleanup(func() {
		_ = db.Where("location_id IN ?", []int{room.ID, disabled.ID}).Delete(&schedule.LocationWeekAvailability{}).Error
	})

	campus := "main"
	kind := "computer_lab"
	available, err := repo.ListAvailableLocationsForWeek(weekStart, &campus, &kind)
	require.NoError(t, err)
	require.NotEmpty(t, available)
	assert.Equal(t, room.ID, available[0].ID)
	for _, loc := range available {
		assert.NotEqual(t, disabled.ID, loc.ID)
	}
}

func TestServiceAutofillLocations_CreatesLocationOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-07", Now: time.Now})

	group := &schedule.Group{Name: "AUTO-G-" + fmt.Sprint(time.Now().UnixNano()), Course: 1}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	subject := &schedule.Subject{Name: "AUTO-S-" + fmt.Sprint(time.Now().UnixNano())}
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", subject.ID).Delete(&schedule.Subject{}).Error })

	placeholder := &schedule.Location{
		Name:         "AUTO-V-" + fmt.Sprint(time.Now().UnixNano()),
		IsVirtual:    true,
		LocationKind: "virtual",
	}
	require.NoError(t, repo.CreateLocation(placeholder))
	t.Cleanup(func() { _ = db.Where("id = ?", placeholder.ID).Delete(&schedule.Location{}).Error })

	room := &schedule.Location{
		Name:         "AUTO-R-" + fmt.Sprint(time.Now().UnixNano()),
		Campus:       "main",
		LocationKind: "computer_lab",
	}
	require.NoError(t, repo.CreateLocation(room))
	t.Cleanup(func() { _ = db.Where("id = ?", room.ID).Delete(&schedule.Location{}).Error })

	tpl := &schedule.ScheduleTemplate{
		GroupID:    group.ID,
		DayOfWeek:  0,
		WeekParity: schedule.WeekParityBoth,
		PairNumber: 1,
		SubjectID:  subject.ID,
		LocationID: placeholder.ID,
		Status:     schedule.StatusPublished,
	}
	require.NoError(t, repo.CreateTemplate(tpl))
	t.Cleanup(func() { _ = db.Where("id = ?", tpl.ID).Delete(&schedule.ScheduleTemplate{}).Error })

	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	_, err := repo.UpsertLocationWeekAvailability(start, []schedule.LocationWeekAvailability{{LocationID: room.ID, IsAvailable: true}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Where("location_id = ?", room.ID).Delete(&schedule.LocationWeekAvailability{}).Error })
	t.Cleanup(func() { _ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleOverride{}).Error })

	campus := "main"
	kind := "computer_lab"
	resp, err := svc.AutofillLocations(schedule.LocationAutofillRequest{
		GroupID:        group.ID,
		StartDate:      start,
		EndDate:        start,
		Campus:         &campus,
		LocationKind:   &kind,
		ReplaceVirtual: true,
		Comment:        ptrString("autofill"),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Assigned)
	assert.Equal(t, 1, resp.Created)

	overrides, err := repo.ListOverrides(schedule.OverrideFilters{GroupID: &group.ID, TargetDate: &start})
	require.NoError(t, err)
	require.Len(t, overrides, 1)
	require.NotNil(t, overrides[0].NewLocationID)
	assert.Equal(t, room.ID, *overrides[0].NewLocationID)

	week, err := svc.GetRange(group.ID, start, start)
	require.NoError(t, err)
	require.Len(t, week.Days, 1)
	require.Len(t, week.Days[0].Lessons, 1)
	require.NotNil(t, week.Days[0].Lessons[0].LocationID)
	assert.Equal(t, room.ID, *week.Days[0].Lessons[0].LocationID)
}
