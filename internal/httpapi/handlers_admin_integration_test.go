//go:build integration

package httpapi

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

func integrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISPO_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/ispo_test?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func uniqueSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func shortCode(prefix string) string {
	return fmt.Sprintf("%s-%06d", prefix, time.Now().UnixNano()%1000000)
}

func makeAdminUser(t *testing.T, db *gorm.DB) *auth.User {
	t.Helper()
	u := &auth.User{
		Login:        "admin_httpapi_" + uniqueSuffix(),
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	require.NoError(t, db.Create(u).Error)
	t.Cleanup(func() {
		_ = db.Where("id = ?", u.ID).Delete(&auth.User{}).Error
	})
	return u
}

func makeGroup(t *testing.T, db *gorm.DB) *schedule.Group {
	t.Helper()
	g := &schedule.Group{Name: "G-" + uniqueSuffix(), Course: 2}
	require.NoError(t, db.Create(g).Error)
	t.Cleanup(func() {
		_ = db.Where("id = ?", g.ID).Delete(&schedule.Group{}).Error
	})
	return g
}

func makeSubject(t *testing.T, db *gorm.DB) *schedule.Subject {
	t.Helper()
	s := &schedule.Subject{Name: "S-" + uniqueSuffix()}
	require.NoError(t, db.Create(s).Error)
	t.Cleanup(func() {
		_ = db.Where("id = ?", s.ID).Delete(&schedule.Subject{}).Error
	})
	return s
}

func makeLocation(t *testing.T, db *gorm.DB) *schedule.Location {
	t.Helper()
	l := &schedule.Location{Name: "L-" + uniqueSuffix(), Kind: "physical", IsActive: true}
	require.NoError(t, db.Create(l).Error)
	t.Cleanup(func() {
		_ = db.Where("id = ?", l.ID).Delete(&schedule.Location{}).Error
	})
	return l
}

func makeTemplate(t *testing.T, db *gorm.DB, repo *schedule.Repository, groupID, subjectID, locationID int, pair int16, status schedule.EntityStatus) *schedule.ScheduleTemplate {
	t.Helper()
	locID := locationID
	tpl := &schedule.ScheduleTemplate{
		GroupID:    groupID,
		DayOfWeek:  0,
		WeekParity: schedule.WeekParityBoth,
		PairNumber: pair,
		SubjectID:  subjectID,
		LocationID: &locID,
		Status:     status,
	}
	require.NoError(t, repo.CreateTemplate(tpl))
	t.Cleanup(func() {
		_ = db.Where("id = ?", tpl.ID).Delete(&schedule.ScheduleTemplate{}).Error
	})
	return tpl
}

func testCtx(method, path string, body *bytes.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body == nil {
		c.Request = httptest.NewRequest(method, path, nil)
	} else {
		c.Request = httptest.NewRequest(method, path, body)
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func TestHandleAdminListDayEvents_PaginationAndFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g1 := makeGroup(t, db)
	g2 := makeGroup(t, db)

	d1 := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	e1 := &schedule.ScheduleDayEvent{GroupID: g1.ID, TargetDate: d1, EventType: "INFO", Title: "A"}
	e2 := &schedule.ScheduleDayEvent{GroupID: g1.ID, TargetDate: d2, EventType: "INFO", Title: "B"}
	e3 := &schedule.ScheduleDayEvent{GroupID: g2.ID, TargetDate: d1, EventType: "INFO", Title: "C"}
	require.NoError(t, repo.CreateDayEvent(e1))
	require.NoError(t, repo.CreateDayEvent(e2))
	require.NoError(t, repo.CreateDayEvent(e3))
	t.Cleanup(func() {
		_ = db.Where("id IN ?", []int64{e1.ID, e2.ID, e3.ID}).Delete(&schedule.ScheduleDayEvent{}).Error
	})

	h := handleAdminListDayEvents(repo)
	c, w := testCtx(http.MethodGet, "/api/v1/admin/day-events?group_id="+strconv.Itoa(g1.ID)+"&target_date=2026-03-16&limit=1&offset=0", nil)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"title\":\"A\"")
	assert.NotContains(t, w.Body.String(), "\"title\":\"B\"")
	assert.NotContains(t, w.Body.String(), "\"title\":\"C\"")
}

func TestHandleAdminCreateDayEvent_SuccessAndValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)

	h := handleAdminCreateDayEvent(repo)
	good := fmt.Sprintf(`{"group_id":%d,"target_date":"2026-03-16T00:00:00Z","event_type":"info","title":"Open day"}`, g.ID)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/day-events", bytes.NewReader([]byte(good)))
	h(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)
	assert.Contains(t, w1.Body.String(), "Open day")
	assert.Contains(t, w1.Body.String(), "INFO")

	bad := `{"group_id":0,"target_date":"2026-03-16T00:00:00Z","event_type":"","title":""}`
	c2, w2 := testCtx(http.MethodPost, "/api/v1/admin/day-events", bytes.NewReader([]byte(bad)))
	h(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "validation_error")
}

func TestHandleAdminUpdateDayEvent_SuccessAndNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	e := &schedule.ScheduleDayEvent{GroupID: g.ID, TargetDate: time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), EventType: "INFO", Title: "Old"}
	require.NoError(t, repo.CreateDayEvent(e))
	t.Cleanup(func() { _ = db.Where("id = ?", e.ID).Delete(&schedule.ScheduleDayEvent{}).Error })

	h := handleAdminUpdateDayEvent(repo)
	okBody := fmt.Sprintf(`{"group_id":%d,"target_date":"2026-03-16T00:00:00Z","event_type":"other","title":"New"}`, g.ID)
	c1, w1 := testCtx(http.MethodPut, "/api/v1/admin/day-events/"+strconv.FormatInt(e.ID, 10), bytes.NewReader([]byte(okBody)))
	c1.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(e.ID, 10)}}
	h(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "OTHER")
	assert.Contains(t, w1.Body.String(), "New")

	missingID := strconv.FormatInt(e.ID+999999, 10)
	c2, w2 := testCtx(http.MethodPut, "/api/v1/admin/day-events/"+missingID, bytes.NewReader([]byte(okBody)))
	c2.Params = []gin.Param{{Key: "id", Value: missingID}}
	h(c2)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestHandleAdminDeleteDayEvent_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	e := &schedule.ScheduleDayEvent{GroupID: g.ID, TargetDate: time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), EventType: "INFO", Title: "Delete"}
	require.NoError(t, repo.CreateDayEvent(e))

	h := handleAdminDeleteDayEvent(repo)
	c, w := testCtx(http.MethodDelete, "/api/v1/admin/day-events/"+strconv.FormatInt(e.ID, 10), nil)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(e.ID, 10)}}
	h(c)
	c.Writer.WriteHeaderNow()

	assert.Equal(t, http.StatusNoContent, w.Code)
	var cnt int64
	require.NoError(t, db.Model(&schedule.ScheduleDayEvent{}).Where("id = ?", e.ID).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestHandleAdminListTemplates_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)
	_ = makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusPublished)
	_ = makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 2, schedule.StatusPublished)

	h := handleAdminListTemplates(repo)
	c, w := testCtx(http.MethodGet, "/api/v1/admin/templates?group_id="+strconv.Itoa(g.ID)+"&limit=1&offset=0", nil)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"pair_number\":1")
}

func TestHandleAdminCreateTemplate_SuccessAndValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)

	h := handleAdminCreateTemplate(repo, nil)
	okBody := fmt.Sprintf(`{"group_id":%d,"day_of_week":0,"week_parity":"both","pair_number":1,"subject_id":%d,"location_id":%d,"status":"published"}`,
		g.ID, s.ID, l.ID)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/templates", bytes.NewReader([]byte(okBody)))
	h(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)
	assert.Contains(t, w1.Body.String(), "\"status\":\"published\"")

	badBody := `{"group_id":0,"pair_number":0,"subject_id":0,"location_id":0}`
	c2, w2 := testCtx(http.MethodPost, "/api/v1/admin/templates", bytes.NewReader([]byte(badBody)))
	h(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "validation_error")
}

func TestHandleAdminUpdateTemplate_BumpsVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)
	tpl := makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusPublished)

	before, err := repo.GetSystemState()
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)

	h := handleAdminUpdateTemplate(repo, nil)
	body := fmt.Sprintf(`{"group_id":%d,"day_of_week":0,"week_parity":"both","pair_number":2,"subject_id":%d,"location_id":%d,"status":"published"}`,
		g.ID, s.ID, l.ID)
	c, w := testCtx(http.MethodPut, "/api/v1/admin/templates/"+strconv.FormatInt(tpl.ID, 10), bytes.NewReader([]byte(body)))
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(tpl.ID, 10)}}
	h(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"pair_number\":2")

	after, err := repo.GetSystemState()
	require.NoError(t, err)
	assert.True(t, after.ScheduleVersion.After(before.ScheduleVersion) || !after.ScheduleVersion.Equal(before.ScheduleVersion))
}

func TestHandleAdminDeleteTemplate_SoftDeleteBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)
	tpl := makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusPublished)

	h := handleAdminDeleteTemplate(repo, nil)
	c, w := testCtx(http.MethodDelete, "/api/v1/admin/templates/"+strconv.FormatInt(tpl.ID, 10), nil)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(tpl.ID, 10)}}
	h(c)
	c.Writer.WriteHeaderNow()
	assert.Equal(t, http.StatusNoContent, w.Code)

	var cnt int64
	require.NoError(t, db.Model(&schedule.ScheduleTemplate{}).Where("id = ?", tpl.ID).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestHandleAdminListOverrides_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	d := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	o1 := &schedule.ScheduleOverride{GroupID: g.ID, TargetDate: d, PairNumber: 1, ActionType: schedule.OverrideCancel}
	o2 := &schedule.ScheduleOverride{GroupID: g.ID, TargetDate: d, PairNumber: 2, ActionType: schedule.OverrideCancel}
	require.NoError(t, repo.CreateOverride(o1))
	require.NoError(t, repo.CreateOverride(o2))
	t.Cleanup(func() { _ = db.Where("id IN ?", []int64{o1.ID, o2.ID}).Delete(&schedule.ScheduleOverride{}).Error })

	h := handleAdminListOverrides(repo)
	c, w := testCtx(http.MethodGet, "/api/v1/admin/overrides?group_id="+strconv.Itoa(g.ID)+"&date=2026-03-16&limit=1", nil)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"pair_number\":1")
}

func TestHandleAdminCreateOverride_CancelReplaceAddAndValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)
	h := handleAdminCreateOverride(repo, nil)

	cancelBody := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-16","pair":1,"action":"CANCEL"}`, g.ID)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(cancelBody)))
	h(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	replaceBody := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-16","pair":2,"action":"REPLACE","new_subject_id":%d}`, g.ID, s.ID)
	c2, w2 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(replaceBody)))
	h(c2)
	assert.Equal(t, http.StatusCreated, w2.Code)

	addBody := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-16","pair":3,"action":"ADD","new_subject_id":%d,"new_location_id":%d}`,
		g.ID, s.ID, l.ID)
	c3, w3 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(addBody)))
	h(c3)
	assert.Equal(t, http.StatusCreated, w3.Code)

	badBody := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-16","pair":9,"action":"ADD"}`, g.ID)
	c4, w4 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(badBody)))
	h(c4)
	assert.Equal(t, http.StatusBadRequest, w4.Code)
	assert.Contains(t, w4.Body.String(), "validation_error")
}

func TestHandleAdminCreateOverride_TeacherDayConstraintRequiresConfirmation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	teacher := &schedule.Teacher{Name: "Constraint Teacher " + uniqueSuffix()}
	require.NoError(t, repo.CreateTeacher(teacher))
	t.Cleanup(func() { _ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error })
	constraint := &schedule.TeacherDayConstraint{
		TeacherID:            teacher.ID,
		TargetDate:           time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
		Reason:               "method day",
		ConstraintLevel:      "warning",
		RequiresConfirmation: true,
	}
	require.NoError(t, repo.CreateTeacherDayConstraint(constraint))
	t.Cleanup(func() { _ = db.Where("id = ?", constraint.ID).Delete(&schedule.TeacherDayConstraint{}).Error })

	h := handleAdminCreateOverride(repo, nil)
	body := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-17","pair":1,"action":"ADD","new_subject_id":%d,"new_teacher_name":%q}`, g.ID, s.ID, teacher.Name)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(body)))
	h(c1)
	require.Equal(t, http.StatusConflict, w1.Code)
	assert.Contains(t, w1.Body.String(), "teacher_day_constraint_confirmation_required")

	confirmed := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-17","pair":1,"action":"ADD","new_subject_id":%d,"new_teacher_name":%q,"confirm_constraints":true}`, g.ID, s.ID, teacher.Name)
	c2, w2 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(confirmed)))
	h(c2)
	assert.Equal(t, http.StatusCreated, w2.Code)

	hardBlock := &schedule.TeacherDayConstraint{
		TeacherID:       teacher.ID,
		TargetDate:      time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
		Reason:          "medical leave",
		ConstraintLevel: "hard_block",
	}
	require.NoError(t, repo.CreateTeacherDayConstraint(hardBlock))
	t.Cleanup(func() { _ = db.Where("id = ?", hardBlock.ID).Delete(&schedule.TeacherDayConstraint{}).Error })

	blocked := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-18","pair":1,"action":"ADD","new_subject_id":%d,"new_teacher_name":%q,"confirm_constraints":true}`, g.ID, s.ID, teacher.Name)
	c3, w3 := testCtx(http.MethodPost, "/api/v1/admin/override", bytes.NewReader([]byte(blocked)))
	h(c3)
	assert.Equal(t, http.StatusConflict, w3.Code)
	assert.Contains(t, w3.Body.String(), "teacher_day_constraint_hard_block")
}

func TestHandleAdminBulkOverrides_MassCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)

	h := handleAdminBulkOverrides(repo)
	body := fmt.Sprintf(`{"group_id":%d,"start_date":"2026-03-16","end_date":"2026-03-16","pair_numbers":[1,2],"action_type":"CANCEL","on_conflict":"error"}`,
		g.ID)
	c, w := testCtx(http.MethodPost, "/api/v1/admin/overrides/bulk", bytes.NewReader([]byte(body)))
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"created\":2")
}

func TestHandleAdminMovePair_AtomicMove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-02-01", Now: time.Now})
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)
	_ = makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusPublished)

	h := handleAdminMovePair(svc, repo)
	body := fmt.Sprintf(`{"group_id":%d,"target_date":"2026-03-16","from_pair_number":1,"to_pair_number":2}`, g.ID)
	c, w := testCtx(http.MethodPost, "/api/v1/admin/override/move", bytes.NewReader([]byte(body)))
	h(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"status\":\"ok\"")

	list, err := repo.ListOverrides(schedule.OverrideFilters{GroupID: &g.ID, TargetDate: ptrTime(time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC))})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestHandleAdminUpsertOverlay_DayOverlay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)

	h := handleAdminUpsertOverlay(repo, nil)
	body := fmt.Sprintf(`{"group_id":%d,"date":"2026-03-16","text":"No classes","style_preset":"warning"}`, g.ID)
	c, w := testCtx(http.MethodPost, "/api/v1/admin/overlay", bytes.NewReader([]byte(body)))
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No classes")
}

func TestHandleAdminUpsertCalendarException_WorksAsDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	targetDate := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		_ = db.Where("target_date = ?", targetDate).Delete(&schedule.CalendarException{}).Error
	})

	h := handleAdminUpsertCalendarException(repo, nil)
	body := `{"date":"2026-03-16","works_as_day":4,"comment":"shift"}`
	c, w := testCtx(http.MethodPost, "/api/v1/admin/calendar-exceptions", bytes.NewReader([]byte(body)))
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"works_as_day\":4")
}

func multipartBodyWithFile(t *testing.T, field, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, err := mw.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = fw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func multipartBodyWithXLSX(t *testing.T, field, filename string, f *excelize.File) (*bytes.Buffer, string) {
	t.Helper()
	b, err := f.WriteToBuffer()
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, err := mw.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = fw.Write(b.Bytes())
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func TestHandleAdminImportTemplatesCSV_Success_MissingColumn_InvalidDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	h := handleAdminImportTemplatesCSV(repo, nil)

	okCSV := "day_of_week,week_parity,pair_number,subject,location,teacher_name\n1,both,1,Math,101,Dr A\n"
	body1, ctype1 := multipartBodyWithFile(t, "file", "ok.csv", okCSV)
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/templates/csv?group_id="+strconv.Itoa(g.ID), body1)
	c1.Request.Header.Set("Content-Type", ctype1)
	h(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "\"inserted\":1")

	missingCSV := "day_of_week,week_parity,pair_number,subject,teacher_name\n1,both,1,Math,Dr A\n"
	body2, ctype2 := multipartBodyWithFile(t, "file", "missing.csv", missingCSV)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/templates/csv?group_id="+strconv.Itoa(g.ID), body2)
	c2.Request.Header.Set("Content-Type", ctype2)
	h(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "validation_error")

	badCSV := "day_of_week,week_parity,pair_number,subject,location,teacher_name\n9,both,1,Math,101,Dr A\n"
	body3, ctype3 := multipartBodyWithFile(t, "file", "bad.csv", badCSV)
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/templates/csv?group_id="+strconv.Itoa(g.ID), body3)
	c3.Request.Header.Set("Content-Type", ctype3)
	h(c3)
	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, w3.Body.String(), "validation_error")
}

func TestHandleAdminImportTemplatesXLSX_Success_InvalidPair(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	h := handleAdminImportTemplatesXLSX(repo, nil)

	fx := excelize.NewFile()
	sheet := fx.GetSheetName(0)
	_ = fx.SetCellValue(sheet, "A1", "day_of_week")
	_ = fx.SetCellValue(sheet, "B1", "week_parity")
	_ = fx.SetCellValue(sheet, "C1", "pair_number")
	_ = fx.SetCellValue(sheet, "D1", "subject")
	_ = fx.SetCellValue(sheet, "E1", "location")
	_ = fx.SetCellValue(sheet, "F1", "teacher_name")
	_ = fx.SetCellValue(sheet, "A2", 1)
	_ = fx.SetCellValue(sheet, "B2", "both")
	_ = fx.SetCellValue(sheet, "C2", 1)
	_ = fx.SetCellValue(sheet, "D2", "Math")
	_ = fx.SetCellValue(sheet, "E2", "101")
	_ = fx.SetCellValue(sheet, "F2", "Teacher")
	body1, ctype1 := multipartBodyWithXLSX(t, "file", "ok.xlsx", fx)
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/templates/xlsx?group_id="+strconv.Itoa(g.ID), body1)
	c1.Request.Header.Set("Content-Type", ctype1)
	h(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "\"inserted\":1")
	_ = fx.Close()

	fxBad := excelize.NewFile()
	s2 := fxBad.GetSheetName(0)
	_ = fxBad.SetCellValue(s2, "A1", "day_of_week")
	_ = fxBad.SetCellValue(s2, "B1", "week_parity")
	_ = fxBad.SetCellValue(s2, "C1", "pair_number")
	_ = fxBad.SetCellValue(s2, "D1", "subject")
	_ = fxBad.SetCellValue(s2, "E1", "location")
	_ = fxBad.SetCellValue(s2, "F1", "teacher_name")
	_ = fxBad.SetCellValue(s2, "A2", 1)
	_ = fxBad.SetCellValue(s2, "B2", "both")
	_ = fxBad.SetCellValue(s2, "C2", 99)
	_ = fxBad.SetCellValue(s2, "D2", "Math")
	_ = fxBad.SetCellValue(s2, "E2", "101")
	_ = fxBad.SetCellValue(s2, "F2", "Teacher")
	body2, ctype2 := multipartBodyWithXLSX(t, "file", "bad.xlsx", fxBad)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/templates/xlsx?group_id="+strconv.Itoa(g.ID), body2)
	c2.Request.Header.Set("Content-Type", ctype2)
	h(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "validation_error")
	_ = fxBad.Close()
}

func TestHandleAdminPublishAndDiscardDraftTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	g := makeGroup(t, db)
	s := makeSubject(t, db)
	l := makeLocation(t, db)

	tplDraft := makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusDraft)
	require.NotZero(t, tplDraft.ID)

	pubH := handleAdminPublishDraftTemplates(repo, nil)
	c1, w1 := testCtx(http.MethodPost, "/api/v1/admin/templates/publish?group_id="+strconv.Itoa(g.ID), nil)
	pubH(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "\"moved\":1")

	tplDraft2 := makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 2, schedule.StatusDraft)
	require.NotZero(t, tplDraft2.ID)
	discH := handleAdminDiscardDraftTemplates(repo)
	c2, w2 := testCtx(http.MethodPost, "/api/v1/admin/templates/discard-drafts?group_id="+strconv.Itoa(g.ID), nil)
	discH(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "\"deleted\":1")
}

func TestHandleAdminValidateSchedule_Warnings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := integrationDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-02-01", Now: time.Now})

	spec := &schedule.Specialty{Code: shortCode("SP"), Name: "Specialty"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })

	cur := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2025, Variant: "v1", Title: "Cur", IsActive: true}
	require.NoError(t, db.Create(cur).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", cur.ID).Delete(&schedule.Curriculum{}).Error })

	g := &schedule.Group{Name: "GV-" + uniqueSuffix(), Course: 2, CurriculumID: &cur.ID}
	require.NoError(t, db.Create(g).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", g.ID).Delete(&schedule.Group{}).Error })

	s := makeSubject(t, db)
	l := makeLocation(t, db)
	_ = makeTemplate(t, db, repo, g.ID, s.ID, l.ID, 1, schedule.StatusPublished)

	h := handleAdminValidateSchedule(svc)
	c, w := testCtx(http.MethodGet, "/api/v1/admin/schedule/validate?group_id="+strconv.Itoa(g.ID)+"&start_date=2026-03-16&end_date=2026-03-16", nil)
	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"validated\":true")
	assert.Contains(t, w.Body.String(), "\"warn_count\":")
	assert.NotContains(t, strings.ToLower(w.Body.String()), "\"warn_count\":0")
}

func ptrTime(t time.Time) *time.Time { return &t }
