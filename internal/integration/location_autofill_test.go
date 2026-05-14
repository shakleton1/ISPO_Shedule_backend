//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ispo-schedule/internal/schedule"
)

func TestRepositoryPlanning_LocationWeekAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	campusID, computerTypeID := ensureTestCampusAndLocationType(t, repo, db)

	room := &schedule.Location{Name: "Avail Room " + fmt.Sprint(time.Now().UnixNano()), CampusID: &campusID, Kind: "physical", IsActive: true}
	require.NoError(t, repo.CreateLocation(room))
	require.NoError(t, repo.CreateLocationTypeLink(&schedule.LocationTypeLink{LocationID: room.ID, TypeID: computerTypeID}))
	t.Cleanup(func() { _ = db.Where("id = ?", room.ID).Delete(&schedule.Location{}).Error })

	disabled := &schedule.Location{Name: "Disabled Room " + fmt.Sprint(time.Now().UnixNano()), CampusID: &campusID, Kind: "physical", IsActive: true}
	require.NoError(t, repo.CreateLocation(disabled))
	require.NoError(t, repo.CreateLocationTypeLink(&schedule.LocationTypeLink{LocationID: disabled.ID, TypeID: computerTypeID}))
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
	locationType := "computer_class"
	available, err := repo.ListAvailableLocationsForWeek(weekStart, &campus, &locationType)
	require.NoError(t, err)
	require.NotEmpty(t, available)
	assert.Equal(t, room.ID, available[0].ID)
	for _, loc := range available {
		assert.NotEqual(t, disabled.ID, loc.ID)
	}
}

func TestServiceAutofillLocations_CreatesRoomAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-07", Now: time.Now})
	campusID, computerTypeID := ensureTestCampusAndLocationType(t, repo, db)

	group := &schedule.Group{Name: "AUTO-G-" + fmt.Sprint(time.Now().UnixNano()), Course: 1}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	subject := &schedule.Subject{Name: "AUTO-S-" + fmt.Sprint(time.Now().UnixNano())}
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", subject.ID).Delete(&schedule.Subject{}).Error })

	placeholder := &schedule.Location{Name: "AUTO-V-" + fmt.Sprint(time.Now().UnixNano()), Kind: "virtual", IsActive: true}
	require.NoError(t, repo.CreateLocation(placeholder))
	t.Cleanup(func() { _ = db.Where("id = ?", placeholder.ID).Delete(&schedule.Location{}).Error })

	room := &schedule.Location{Name: "AUTO-R-" + fmt.Sprint(time.Now().UnixNano()), CampusID: &campusID, Kind: "physical", IsActive: true}
	require.NoError(t, repo.CreateLocation(room))
	require.NoError(t, repo.CreateLocationTypeLink(&schedule.LocationTypeLink{LocationID: room.ID, TypeID: computerTypeID}))
	t.Cleanup(func() { _ = db.Where("id = ?", room.ID).Delete(&schedule.Location{}).Error })

	placeholderID := placeholder.ID
	subjectID := subject.ID
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	lesson := &schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   start,
		PairNumber:   1,
		SubjectID:    &subjectID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}
	require.NoError(t, repo.CreateScheduleLesson(lesson))
	require.NoError(t, repo.CreateRoomAssignment(&schedule.RoomAssignment{ScheduleLessonID: lesson.ID, LocationID: placeholderID, Source: "manual", Status: schedule.StatusPublished}))
	t.Cleanup(func() { _ = db.Where("id = ?", lesson.ID).Delete(&schedule.ScheduleLesson{}).Error })

	_, err := repo.UpsertLocationWeekAvailability(start, []schedule.LocationWeekAvailability{{LocationID: room.ID, IsAvailable: true}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Where("location_id = ?", room.ID).Delete(&schedule.LocationWeekAvailability{}).Error })

	campus := "main"
	locationType := "computer_class"
	resp, err := svc.AutofillLocations(schedule.LocationAutofillRequest{
		GroupID:        group.ID,
		StartDate:      start,
		EndDate:        start,
		CampusName:     &campus,
		LocationType:   &locationType,
		ReplaceVirtual: true,
		Comment:        ptrString("autofill"),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Assigned)
	assert.Equal(t, 1, resp.Updated)

	assignments, err := repo.ListRoomAssignments(schedule.RoomAssignmentFilters{ScheduleLessonID: &lesson.ID})
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, room.ID, assignments[0].LocationID)

	week, err := svc.GetRange(group.ID, start, start)
	require.NoError(t, err)
	require.Len(t, week.Days, 1)
	require.Len(t, week.Days[0].Lessons, 1)
	require.NotNil(t, week.Days[0].Lessons[0].LocationID)
	assert.Equal(t, room.ID, *week.Days[0].Lessons[0].LocationID)
}

func ensureTestCampusAndLocationType(t *testing.T, repo *schedule.Repository, db *gorm.DB) (int, int) {
	t.Helper()
	_ = repo
	var campus schedule.Campus
	require.NoError(t, db.Raw(`
INSERT INTO campuses (name)
VALUES ('main')
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id, name, address, created_at, updated_at`).Scan(&campus).Error)

	var locationType schedule.LocationType
	require.NoError(t, db.Raw(`
INSERT INTO location_types (code, name)
VALUES ('computer_class', 'Computer class')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
RETURNING id, code, name, created_at, updated_at`).Scan(&locationType).Error)

	return campus.ID, locationType.ID
}
