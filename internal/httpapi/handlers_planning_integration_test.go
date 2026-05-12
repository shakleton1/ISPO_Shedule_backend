//go:build integration

package httpapi

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/schedule"
)

func TestHandleAdminStudyActivityCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)

	createH := handleAdminCreateStudyActivity(repo)
	code := "ACT-" + uniqueSuffix()
	body := fmt.Sprintf(`{"code":%q,"name":"Practice","activity_kind":"PRACTICE","allows_lessons":false}`, code)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/study-activities", bytes.NewReader([]byte(body)))
	createH(c1)
	require.Equal(t, http.StatusCreated, w1.Code)
	assert.Contains(t, w1.Body.String(), code)

	var created schedule.StudyActivity
	require.NoError(t, db.Where("code = ?", code).First(&created).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", created.ID).Delete(&schedule.StudyActivity{}).Error })

	listH := handleAdminListStudyActivities(repo)
	c2, w2 := testCtx(http.MethodGet, "/api/v1/admin/study-activities?limit=20", nil)
	listH(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), code)

	updateH := handleAdminUpdateStudyActivity(repo)
	updateBody := fmt.Sprintf(`{"code":%q,"name":"Practice Updated","activity_kind":"PRACTICE","allows_lessons":true}`, code)
	c3, w3 := testCtx(http.MethodPut, "/api/v1/admin/study-activities/"+strconv.Itoa(created.ID), bytes.NewReader([]byte(updateBody)))
	c3.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(created.ID)}}
	updateH(c3)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, w3.Body.String(), "Practice Updated")
}

func TestHandleAdminUpsertStudyCalendarWeeks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	group := makeGroup(t, db)
	activity := &schedule.StudyActivity{Code: "CAL-" + uniqueSuffix(), Name: "Practice", ActivityKind: "PRACTICE", AllowsLessons: false}
	require.NoError(t, repo.CreateStudyActivity(activity))
	t.Cleanup(func() { _ = db.Where("id = ?", activity.ID).Delete(&schedule.StudyActivity{}).Error })

	h := handleAdminUpsertStudyCalendarWeeks(repo, nil)
	body := fmt.Sprintf(`[{"week_number":1,"week_start_date":"2026-09-07","activity_id":%d,"allows_lessons":false}]`, activity.ID)
	c, w := testCtx(http.MethodPut, "/api/v1/admin/groups/"+strconv.Itoa(group.ID)+"/study-calendar-weeks", bytes.NewReader([]byte(body)))
	c.Params = []gin.Param{{Key: "id", Value: strconv.Itoa(group.ID)}}
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"allows_lessons":false`)
}

func TestHandleAdminTeacherDayConstraintCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	teacher := &schedule.Teacher{Name: "Teacher Constraint " + uniqueSuffix()}
	require.NoError(t, repo.CreateTeacher(teacher))
	t.Cleanup(func() { _ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error })

	createH := handleAdminCreateTeacherDayConstraint(repo)
	body := fmt.Sprintf(`{"teacher_id":%d,"date":"2026-09-08","reason":"illness","allows_lessons":false}`, teacher.ID)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/teacher-day-constraints", bytes.NewReader([]byte(body)))
	createH(c1)
	require.Equal(t, http.StatusCreated, w1.Code)

	var row schedule.TeacherDayConstraint
	require.NoError(t, db.Where("teacher_id = ?", teacher.ID).First(&row).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", row.ID).Delete(&schedule.TeacherDayConstraint{}).Error })

	listH := handleAdminListTeacherDayConstraints(repo)
	c2, w2 := testCtx(http.MethodGet, "/api/v1/admin/teacher-day-constraints?teacher_id="+strconv.Itoa(teacher.ID), nil)
	listH(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "illness")
}

func TestHandleAdminScheduleReplacementCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	group := makeGroup(t, db)

	createH := handleAdminCreateScheduleReplacement(repo)
	body := fmt.Sprintf(`{"group_id":%d,"date":"2026-09-09","pair_number":2,"reason":"teacher replacement"}`, group.ID)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/replacements", bytes.NewReader([]byte(body)))
	createH(c1)
	require.Equal(t, http.StatusCreated, w1.Code)

	var row schedule.ScheduleReplacement
	require.NoError(t, db.Where("group_id = ?", group.ID).First(&row).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", row.ID).Delete(&schedule.ScheduleReplacement{}).Error })

	listH := handleAdminListScheduleReplacements(repo)
	c2, w2 := testCtx(http.MethodGet, "/api/v1/admin/replacements?group_id="+strconv.Itoa(group.ID), nil)
	listH(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "teacher replacement")
}

func TestHandleAdminLocationWeekAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	location := makeLocation(t, db)

	upsertH := handleAdminUpsertLocationWeekAvailability(repo)
	body := fmt.Sprintf(`[{"location_id":%d,"is_available":true,"comment":"week list"}]`, location.ID)
	c1, w1 := testCtx(http.MethodPut, "/api/v1/admin/location-availability/weeks?week_start_date=2026-09-09", bytes.NewReader([]byte(body)))
	upsertH(c1)
	require.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "week list")
	t.Cleanup(func() {
		_ = db.Where("location_id = ?", location.ID).Delete(&schedule.LocationWeekAvailability{}).Error
	})

	listH := handleAdminListLocationWeekAvailability(repo)
	c2, w2 := testCtx(http.MethodGet, "/api/v1/admin/location-availability/weeks?week_start_date=2026-09-07", nil)
	listH(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"location_id":`+strconv.Itoa(location.ID))
}

func TestHandleAdminScheduleViewTeacherAndLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-02-01", Now: time.Now})
	group := makeGroup(t, db)
	subject := makeSubject(t, db)
	location := makeLocation(t, db)
	teacher := &schedule.Teacher{Name: "Teacher Filter " + uniqueSuffix()}
	require.NoError(t, repo.CreateTeacher(teacher))
	t.Cleanup(func() { _ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error })

	tpl := &schedule.ScheduleTemplate{
		GroupID:     group.ID,
		DayOfWeek:   0,
		WeekParity:  schedule.WeekParityBoth,
		PairNumber:  1,
		SubjectID:   subject.ID,
		LocationID:  location.ID,
		TeacherName: teacher.Name,
		Status:      schedule.StatusPublished,
	}
	require.NoError(t, repo.CreateTemplate(tpl))
	t.Cleanup(func() { _ = db.Where("id = ?", tpl.ID).Delete(&schedule.ScheduleTemplate{}).Error })

	h := handleAdminScheduleView(svc)
	c1, w1 := testCtx(http.MethodGet, "/api/v1/admin/schedule/view?scope=teacher&teacher_id="+strconv.Itoa(teacher.ID)+"&date_start=2026-04-06&date_end=2026-04-06", nil)
	h(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), `"scope":"teacher"`)
	assert.Contains(t, w1.Body.String(), teacher.Name)
	assert.Contains(t, w1.Body.String(), group.Name)

	c2, w2 := testCtx(http.MethodGet, "/api/v1/admin/schedule/view?scope=location&location_id="+strconv.Itoa(location.ID)+"&date_start=2026-04-06&date_end=2026-04-06", nil)
	h(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"scope":"location"`)
	assert.Contains(t, w2.Body.String(), location.Name)
	assert.Contains(t, w2.Body.String(), group.Name)
}

func TestHandleAdminImportCurriculumItemsCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	spec := &schedule.Specialty{Code: shortCode("IMP"), Name: "Import Spec"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })
	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2026, Variant: "A", Title: "Import Curriculum", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID).Error })

	h := handleAdminImportCurriculumItemsCSV(repo)
	csvData := "discipline,course,semester,hours\nMath,1,1,72\n"
	body, ctype := multipartBodyWithFile(t, "file", "curriculum.csv", csvData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/curriculum-items/csv?curriculum_id="+strconv.FormatInt(curr.ID, 10), body)
	c.Request.Header.Set("Content-Type", ctype)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"imported":1`)
}

func TestHandleAdminImportStudyCalendarCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	group := makeGroup(t, db)

	h := handleAdminImportStudyCalendarCSV(repo, nil)
	csvData := fmt.Sprintf("group,week_number,activity_code,activity_name,allows_lessons\n%s,1,PRACTICE,Practice,no\n", group.Name)
	body, ctype := multipartBodyWithFile(t, "file", "calendar.csv", csvData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/study-calendar/csv", body)
	c.Request.Header.Set("Content-Type", ctype)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"imported":1`)
}
