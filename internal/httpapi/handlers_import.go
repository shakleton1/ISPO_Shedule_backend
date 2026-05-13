package httpapi

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

type importTemplateRow struct {
	DayOfWeek   int16
	WeekParity  schedule.WeekParity
	PairNumber  int16
	SubjectName string
	Location    string
	TeacherName string
	Subgroup    *int16
	FlowKey     *string
}

type importCurriculumItemRow struct {
	Discipline  string
	Course      int16
	Semester    int16
	HoursTotal  int
	ItemType    string
	SubjectName string
}

type importStudyCalendarRow struct {
	GroupID       *int
	GroupName     string
	WeekNumber    int16
	WeekStartDate *time.Time
	ActivityCode  string
	ActivityName  string
	ActivityKind  string
	AllowsLessons bool
	Comment       *string
}

func handleAdminImportTemplatesCSV(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := parseGroupIDFromRequest(c)
		if !ok {
			return
		}

		status := schedule.StatusPublished
		if v := strings.TrimSpace(c.Query("status")); v != "" {
			status = schedule.EntityStatus(v)
		}
		if status != schedule.StatusDraft && status != schedule.StatusPublished {
			writeValidationError(c, "status", "invalid status")
			return
		}

		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseTemplatesCSV(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}

		inserted, ver, err := importTemplatesReplace(c, repo, groupID, status, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil && status == schedule.StatusPublished {
			pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
		}
		writeAudit(c, repo, "import", "schedule_templates", fmt.Sprintf("group:%d", groupID), gin.H{"inserted": inserted})
		c.JSON(http.StatusOK, gin.H{"inserted": inserted, "status": status, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportTemplatesXLSX(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := parseGroupIDFromRequest(c)
		if !ok {
			return
		}

		status := schedule.StatusPublished
		if v := strings.TrimSpace(c.Query("status")); v != "" {
			status = schedule.EntityStatus(v)
		}
		if status != schedule.StatusDraft && status != schedule.StatusPublished {
			writeValidationError(c, "status", "invalid status")
			return
		}

		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseTemplatesXLSX(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}

		inserted, ver, err := importTemplatesReplace(c, repo, groupID, status, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil && status == schedule.StatusPublished {
			pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
		}
		writeAudit(c, repo, "import", "schedule_templates", fmt.Sprintf("group:%d", groupID), gin.H{"inserted": inserted})
		c.JSON(http.StatusOK, gin.H{"inserted": inserted, "status": status, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportCurriculumItemsCSV(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		curriculumID, ok := parseCurriculumIDFromRequest(c)
		if !ok {
			return
		}
		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseCurriculumItemsCSV(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}
		imported, ver, err := importCurriculumItems(c, repo, curriculumID, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "import", "curriculum_items", fmt.Sprintf("curriculum:%d", curriculumID), gin.H{"imported": imported})
		c.JSON(http.StatusOK, gin.H{"imported": imported, "curriculum_id": curriculumID, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportCurriculumItemsXLSX(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		curriculumID, ok := parseCurriculumIDFromRequest(c)
		if !ok {
			return
		}
		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseCurriculumItemsXLSX(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}
		imported, ver, err := importCurriculumItems(c, repo, curriculumID, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "import", "curriculum_items", fmt.Sprintf("curriculum:%d", curriculumID), gin.H{"imported": imported})
		c.JSON(http.StatusOK, gin.H{"imported": imported, "curriculum_id": curriculumID, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportStudyCalendarCSV(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseStudyCalendarCSV(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}
		imported, groupIDs, ver, err := importStudyCalendar(c, repo, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil {
			for _, groupID := range groupIDs {
				pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
			}
		}
		writeAudit(c, repo, "import", "study_calendar_weeks", "bulk", gin.H{"imported": imported, "groups": groupIDs})
		c.JSON(http.StatusOK, gin.H{"imported": imported, "group_ids": groupIDs, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportStudyCalendarXLSX(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		f, err := getUploadedFile(c, "file")
		if err != nil {
			if errors.Is(err, errRequestBodyTooLarge) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", err.Error())
			return
		}
		defer f.Close()

		rows, err := parseStudyCalendarXLSX(f)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}
		imported, groupIDs, ver, err := importStudyCalendar(c, repo, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil {
			for _, groupID := range groupIDs {
				pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
			}
		}
		writeAudit(c, repo, "import", "study_calendar_weeks", "bulk", gin.H{"imported": imported, "groups": groupIDs})
		c.JSON(http.StatusOK, gin.H{"imported": imported, "group_ids": groupIDs, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func parseGroupIDFromRequest(c *gin.Context) (int, bool) {
	if q := strings.TrimSpace(c.Query("group_id")); q != "" {
		gid, err := strconv.Atoi(q)
		if err != nil || gid <= 0 {
			writeValidationError(c, "group_id", "invalid group_id")
			return 0, false
		}
		return gid, true
	}
	gidStr := strings.TrimSpace(c.PostForm("group_id"))
	if gidStr == "" {
		writeValidationError(c, "group_id", "group_id required")
		return 0, false
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil || gid <= 0 {
		writeValidationError(c, "group_id", "invalid group_id")
		return 0, false
	}
	return gid, true
}

func parseCurriculumIDFromRequest(c *gin.Context) (int64, bool) {
	if q := strings.TrimSpace(c.Query("curriculum_id")); q != "" {
		id, err := strconv.ParseInt(q, 10, 64)
		if err != nil || id <= 0 {
			writeValidationError(c, "curriculum_id", "invalid curriculum_id")
			return 0, false
		}
		return id, true
	}
	raw := strings.TrimSpace(c.PostForm("curriculum_id"))
	if raw == "" {
		writeValidationError(c, "curriculum_id", "curriculum_id required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeValidationError(c, "curriculum_id", "invalid curriculum_id")
		return 0, false
	}
	return id, true
}

func getUploadedFile(c *gin.Context, field string) (io.ReadCloser, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		if isRequestBodyTooLarge(err) {
			return nil, errRequestBodyTooLarge
		}
		return nil, fmt.Errorf("missing %s file", field)
	}
	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("cannot open upload")
	}
	return file, nil
}

func importTemplatesReplace(c *gin.Context, repo *schedule.Repository, groupID int, status schedule.EntityStatus, rows []importTemplateRow) (int, time.Time, error) {
	if len(rows) == 0 {
		return 0, time.Time{}, fmt.Errorf("empty file")
	}

	var inserted int
	err := repo.DB().Transaction(func(tx *gorm.DB) error {
		// ensure group exists
		var g schedule.Group
		if err := tx.First(&g, "id = ?", groupID).Error; err != nil {
			return err
		}

		if err := tx.Where("group_id = ? AND status = ?", groupID, status).Delete(&schedule.ScheduleTemplate{}).Error; err != nil {
			return err
		}

		templates := make([]schedule.ScheduleTemplate, 0, len(rows))
		for i, r := range rows {
			subID, err := getOrCreateSubjectID(tx, r.SubjectName)
			if err != nil {
				return fmt.Errorf("row %d: subject: %w", i+1, err)
			}
			locID, err := getOrCreateLocationID(tx, r.Location)
			if err != nil {
				return fmt.Errorf("row %d: location: %w", i+1, err)
			}
			teacherID, err := getOrCreateTeacherID(tx, r.TeacherName)
			if err != nil {
				return fmt.Errorf("row %d: teacher: %w", i+1, err)
			}
			templates = append(templates, schedule.ScheduleTemplate{
				GroupID:    groupID,
				DayOfWeek:  r.DayOfWeek,
				WeekParity: r.WeekParity,
				PairNumber: r.PairNumber,
				SubjectID:  subID,
				LocationID: &locID,
				Status:     status,
				TeacherID:  teacherID,
				Subgroup:   r.Subgroup,
				FlowKey:    r.FlowKey,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			})
		}
		if err := tx.CreateInBatches(&templates, 200).Error; err != nil {
			return err
		}
		inserted = len(templates)
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	if status != schedule.StatusPublished {
		// Draft import must not affect clients; return current version.
		state, err := repo.GetSystemState()
		if err != nil {
			return inserted, time.Time{}, err
		}
		return inserted, state.ScheduleVersion, nil
	}
	ver, err := bumpScheduleVersionAndGet(repo)
	if err != nil {
		return inserted, time.Time{}, err
	}
	return inserted, ver, nil
}

func importCurriculumItems(c *gin.Context, repo *schedule.Repository, curriculumID int64, rows []importCurriculumItemRow) (int, time.Time, error) {
	_ = c
	if len(rows) == 0 {
		return 0, time.Time{}, fmt.Errorf("empty file")
	}
	imported := 0
	err := repo.DB().Transaction(func(tx *gorm.DB) error {
		var curr schedule.Curriculum
		if err := tx.First(&curr, "id = ?", curriculumID).Error; err != nil {
			return err
		}
		for i, r := range rows {
			if r.Course <= 0 || r.Course > 6 {
				return fmt.Errorf("row %d: course must be 1..6", i+1)
			}
			minSem := (r.Course-1)*2 + 1
			maxSem := minSem + 1
			if r.Semester < minSem || r.Semester > maxSem {
				return fmt.Errorf("row %d: semester %d does not belong to course %d", i+1, r.Semester, r.Course)
			}
			subjectName := r.SubjectName
			if subjectName == "" {
				subjectName = r.Discipline
			}
			subID, err := getOrCreateSubjectID(tx, subjectName)
			if err != nil {
				return fmt.Errorf("row %d: subject: %w", i+1, err)
			}

			itemType := strings.ToUpper(strings.TrimSpace(r.ItemType))
			if itemType == "" {
				itemType = "DISCIPLINE"
			}
			var item schedule.CurriculumItem
			err = tx.Where("curriculum_id = ? AND name = ?", curriculumID, r.Discipline).First(&item).Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				item = schedule.CurriculumItem{
					CurriculumID: curriculumID,
					ItemType:     itemType,
					Name:         r.Discipline,
					SubjectID:    &subID,
				}
				if err := tx.Create(&item).Error; err != nil {
					return err
				}
			} else {
				item.ItemType = itemType
				item.SubjectID = &subID
				if err := tx.Save(&item).Error; err != nil {
					return err
				}
			}

			if err := tx.Exec(`
INSERT INTO curriculum_item_allocations
  (item_id, semester, hours_total)
VALUES
  (?, ?, ?)
ON CONFLICT (item_id, semester)
DO UPDATE SET hours_total = EXCLUDED.hours_total`,
				item.ID, r.Semester, r.HoursTotal,
			).Error; err != nil {
				return err
			}
			imported++
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	ver, err := bumpScheduleVersionAndGet(repo)
	if err != nil {
		return imported, time.Time{}, err
	}
	return imported, ver, nil
}

func importStudyCalendar(c *gin.Context, repo *schedule.Repository, rows []importStudyCalendarRow) (int, []int, time.Time, error) {
	_ = c
	if len(rows) == 0 {
		return 0, nil, time.Time{}, fmt.Errorf("empty file")
	}
	imported := 0
	groupSeen := map[int]bool{}
	err := repo.DB().Transaction(func(tx *gorm.DB) error {
		for i, r := range rows {
			groupID, err := resolveGroupIDForImport(tx, r.GroupID, r.GroupName)
			if err != nil {
				return fmt.Errorf("row %d: group: %w", i+1, err)
			}
			activityID, err := getOrCreateStudyActivityID(tx, r.ActivityCode, r.ActivityName, r.ActivityKind, r.AllowsLessons)
			if err != nil {
				return fmt.Errorf("row %d: activity: %w", i+1, err)
			}
			if r.WeekStartDate != nil {
				d := mondayOfWeekHTTP(*r.WeekStartDate)
				r.WeekStartDate = &d
			}
			if err := tx.Exec(`
INSERT INTO study_calendar_weeks
  (group_id, week_number, week_start_date, activity_id, allows_lessons, comment)
VALUES
  (?, ?, ?, ?, ?, ?)
ON CONFLICT (group_id, week_number)
DO UPDATE SET
  week_start_date = EXCLUDED.week_start_date,
  activity_id = EXCLUDED.activity_id,
  allows_lessons = EXCLUDED.allows_lessons,
  comment = EXCLUDED.comment`,
				groupID, r.WeekNumber, r.WeekStartDate, activityID, r.AllowsLessons, r.Comment,
			).Error; err != nil {
				return err
			}
			groupSeen[groupID] = true
			imported++
		}
		return nil
	})
	if err != nil {
		return 0, nil, time.Time{}, err
	}
	ver, err := bumpScheduleVersionAndGet(repo)
	if err != nil {
		return imported, nil, time.Time{}, err
	}
	groupIDs := make([]int, 0, len(groupSeen))
	for id := range groupSeen {
		groupIDs = append(groupIDs, id)
	}
	return imported, groupIDs, ver, nil
}

func getOrCreateSubjectID(tx *gorm.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("name required")
	}
	var s schedule.Subject
	err := tx.Where("name = ?", name).First(&s).Error
	if err == nil {
		return s.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	s = schedule.Subject{Name: name}
	if err := tx.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func getOrCreateStudyActivityID(tx *gorm.DB, code, name, kind string, allowsLessons bool) (int, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if code == "" {
		code = name
	}
	if name == "" {
		name = code
	}
	if code == "" {
		return 0, fmt.Errorf("activity code or name required")
	}
	if kind == "" {
		kind = "OTHER"
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := tx.Raw(`
INSERT INTO study_activities (code, name, activity_kind, allows_lessons)
VALUES (?, ?, ?, ?)
ON CONFLICT (code)
DO UPDATE SET
  name = EXCLUDED.name,
  activity_kind = EXCLUDED.activity_kind,
  allows_lessons = EXCLUDED.allows_lessons
RETURNING id`, code, name, kind, allowsLessons).Scan(&out).Error
	if err != nil {
		return 0, err
	}
	return out.ID, nil
}

func resolveGroupIDForImport(tx *gorm.DB, groupID *int, groupName string) (int, error) {
	if groupID != nil && *groupID > 0 {
		var g schedule.Group
		if err := tx.First(&g, "id = ?", *groupID).Error; err != nil {
			return 0, err
		}
		return g.ID, nil
	}
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return 0, fmt.Errorf("group_id or group required")
	}
	var g schedule.Group
	if err := tx.Where("name = ?", groupName).First(&g).Error; err != nil {
		return 0, err
	}
	return g.ID, nil
}

func getOrCreateLocationID(tx *gorm.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("name required")
	}
	var l schedule.Location
	err := tx.Where("name = ?", name).First(&l).Error
	if err == nil {
		return l.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	l = schedule.Location{Name: name, Kind: "physical", IsActive: true}
	if err := tx.Create(&l).Error; err != nil {
		return 0, err
	}
	return l.ID, nil
}

func getOrCreateTeacherID(tx *gorm.DB, name string) (*int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	if err := tx.Raw(
		"INSERT INTO teachers (name) VALUES (?) ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name RETURNING id",
		name,
	).Scan(&out).Error; err != nil {
		return nil, err
	}
	return &out.ID, nil
}

func parseTemplatesCSV(r io.Reader) ([]importTemplateRow, error) {
	br := bufio.NewReader(r)
	peek, _ := br.Peek(4096)
	del := detectCSVDelimiter(peek)

	cr := csv.NewReader(br)
	cr.Comma = del
	cr.FieldsPerRecord = -1

	head, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := mapHeaders(head)
	required := []string{"day_of_week", "week_parity", "pair_number", "subject", "location", "teacher_name"}
	for _, k := range required {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}

	var out []importTemplateRow
	rowNum := 1
	for {
		rowNum++
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		if len(rec) == 0 {
			continue
		}
		parsed, err := parseTemplateRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func detectCSVDelimiter(peek []byte) rune {
	// default comma, but semicolon is common in RU locales.
	line := string(peek)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	comma := strings.Count(line, ",")
	semi := strings.Count(line, ";")
	if semi > comma {
		return ';'
	}
	return ','
}

func parseTemplatesXLSX(r io.Reader) ([]importTemplateRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx has no sheets")
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read xlsx rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty xlsx")
	}

	idx := mapHeaders(rows[0])
	required := []string{"day_of_week", "week_parity", "pair_number", "subject", "location", "teacher_name"}
	for _, k := range required {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}

	out := make([]importTemplateRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		rec := rows[i]
		// skip empty lines
		empty := true
		for _, v := range rec {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}

		parsed, err := parseTemplateRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseCurriculumItemsCSV(r io.Reader) ([]importCurriculumItemRow, error) {
	br := bufio.NewReader(r)
	peek, _ := br.Peek(4096)
	del := detectCSVDelimiter(peek)

	cr := csv.NewReader(br)
	cr.Comma = del
	cr.FieldsPerRecord = -1

	head, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := mapHeaders(head)
	for _, k := range []string{"discipline", "course", "semester", "hours_total"} {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}

	var out []importCurriculumItemRow
	rowNum := 1
	for {
		rowNum++
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		if recordIsEmpty(rec) {
			continue
		}
		parsed, err := parseCurriculumItemRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseCurriculumItemsXLSX(r io.Reader) ([]importCurriculumItemRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read xlsx rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty xlsx")
	}
	idx := mapHeaders(rows[0])
	for _, k := range []string{"discipline", "course", "semester", "hours_total"} {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}

	out := make([]importCurriculumItemRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		rec := rows[i]
		if recordIsEmpty(rec) {
			continue
		}
		parsed, err := parseCurriculumItemRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseStudyCalendarCSV(r io.Reader) ([]importStudyCalendarRow, error) {
	br := bufio.NewReader(r)
	peek, _ := br.Peek(4096)
	del := detectCSVDelimiter(peek)

	cr := csv.NewReader(br)
	cr.Comma = del
	cr.FieldsPerRecord = -1

	head, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := mapHeaders(head)
	for _, k := range []string{"week_number", "allows_lessons"} {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}
	if _, ok := idx["group_id"]; !ok {
		if _, ok := idx["group"]; !ok {
			return nil, fmt.Errorf("missing required column: group or group_id")
		}
	}
	if _, ok := idx["activity_code"]; !ok {
		if _, ok := idx["activity_name"]; !ok {
			return nil, fmt.Errorf("missing required column: activity_code or activity_name")
		}
	}

	var out []importStudyCalendarRow
	rowNum := 1
	for {
		rowNum++
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		if recordIsEmpty(rec) {
			continue
		}
		parsed, err := parseStudyCalendarRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseStudyCalendarXLSX(r io.Reader) ([]importStudyCalendarRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read xlsx rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty xlsx")
	}
	idx := mapHeaders(rows[0])
	for _, k := range []string{"week_number", "allows_lessons"} {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column: %s", k)
		}
	}
	if _, ok := idx["group_id"]; !ok {
		if _, ok := idx["group"]; !ok {
			return nil, fmt.Errorf("missing required column: group or group_id")
		}
	}
	if _, ok := idx["activity_code"]; !ok {
		if _, ok := idx["activity_name"]; !ok {
			return nil, fmt.Errorf("missing required column: activity_code or activity_name")
		}
	}

	out := make([]importStudyCalendarRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		rec := rows[i]
		if recordIsEmpty(rec) {
			continue
		}
		parsed, err := parseStudyCalendarRecord(func(key string) string {
			j, ok := idx[key]
			if !ok {
				return ""
			}
			if j < 0 || j >= len(rec) {
				return ""
			}
			return rec[j]
		})
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func mapHeaders(head []string) map[string]int {
	m := make(map[string]int, len(head))
	for i, h := range head {
		k := normalizeHeader(h)
		switch k {
		case "day_of_week", "day", "dow", "день", "день_недели":
			m["day_of_week"] = i
		case "week_parity", "parity", "week", "четность", "неделя":
			m["week_parity"] = i
		case "pair_number", "pair", "пара", "номер_пары":
			m["pair_number"] = i
		case "subject", "subject_name", "предмет", "дисциплина":
			m["subject"] = i
			if k == "дисциплина" || k == "предмет" {
				m["discipline"] = i
			}
		case "location", "room", "кабинет", "аудитория":
			m["location"] = i
		case "teacher_name", "teacher", "преподаватель":
			m["teacher_name"] = i
		case "subgroup", "подгруппа":
			m["subgroup"] = i
		case "flow_key", "flow", "stream", "РїРѕС‚РѕРє", "РєР»СЋС‡_РїРѕС‚РѕРєР°":
			m["flow_key"] = i
		case "discipline", "discipline_name", "наименование_дисциплины":
			m["discipline"] = i
		case "course", "курс":
			m["course"] = i
		case "semester", "sem", "семестр":
			m["semester"] = i
		case "hours", "hours_total", "total_hours", "часы", "количество_часов", "кол_во_часов":
			m["hours_total"] = i
		case "item_type", "тип", "тип_строки":
			m["item_type"] = i
		case "group_id", "группа_id", "id_группы":
			m["group_id"] = i
		case "group", "group_name", "группа":
			m["group"] = i
		case "week_number", "week_num", "номер_недели", "неделя_номер":
			m["week_number"] = i
		case "week_start_date", "week_start", "дата_начала_недели":
			m["week_start_date"] = i
		case "activity_code", "activity", "код_занятости", "занятость":
			m["activity_code"] = i
		case "activity_name", "activity_full_name", "название_занятости", "полное_наименование":
			m["activity_name"] = i
		case "activity_kind", "вид_занятости":
			m["activity_kind"] = i
		case "allows_lessons", "allow_lessons", "можно_ставить_пары", "разрешены_ли_пары":
			m["allows_lessons"] = i
		case "comment", "комментарий", "примечание":
			m["comment"] = i
		}
	}
	return m
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func parseTemplateRecord(get func(string) string) (importTemplateRow, error) {
	day, err := parseInt16(get("day_of_week"))
	if err != nil {
		return importTemplateRow{}, fmt.Errorf("day_of_week: %w", err)
	}
	// accept 1..6 as Mon..Sat too
	if day >= 1 && day <= 6 {
		day = day - 1
	}
	if day < 0 || day > 5 {
		return importTemplateRow{}, fmt.Errorf("day_of_week must be 0..5 (or 1..6)")
	}

	par := schedule.WeekParity(strings.TrimSpace(strings.ToLower(get("week_parity"))))
	switch par {
	case schedule.WeekParityNumerator, schedule.WeekParityDenominator, schedule.WeekParityBoth:
	default:
		return importTemplateRow{}, fmt.Errorf("week_parity must be numerator/denominator/both")
	}

	pair, err := parseInt16(get("pair_number"))
	if err != nil {
		return importTemplateRow{}, fmt.Errorf("pair_number: %w", err)
	}
	if pair < 1 || pair > 8 {
		return importTemplateRow{}, fmt.Errorf("pair_number must be 1..8")
	}

	subject := strings.TrimSpace(get("subject"))
	location := strings.TrimSpace(get("location"))
	teacher := strings.TrimSpace(get("teacher_name"))
	if subject == "" {
		return importTemplateRow{}, fmt.Errorf("subject required")
	}
	if location == "" {
		return importTemplateRow{}, fmt.Errorf("location required")
	}

	var subgroup *int16
	if sgRaw := strings.TrimSpace(get("subgroup")); sgRaw != "" {
		sg, err := parseInt16(sgRaw)
		if err != nil {
			return importTemplateRow{}, fmt.Errorf("subgroup: %w", err)
		}
		if sg != 1 && sg != 2 {
			return importTemplateRow{}, fmt.Errorf("subgroup must be 1/2")
		}
		subgroup = &sg
	}
	var flowKey *string
	if raw := strings.TrimSpace(get("flow_key")); raw != "" {
		flowKey = &raw
	}

	return importTemplateRow{
		DayOfWeek:   day,
		WeekParity:  par,
		PairNumber:  pair,
		SubjectName: subject,
		Location:    location,
		TeacherName: teacher,
		Subgroup:    subgroup,
		FlowKey:     flowKey,
	}, nil
}

func parseCurriculumItemRecord(get func(string) string) (importCurriculumItemRow, error) {
	discipline := strings.TrimSpace(get("discipline"))
	if discipline == "" {
		return importCurriculumItemRow{}, fmt.Errorf("discipline required")
	}
	course, err := parseInt16(get("course"))
	if err != nil {
		return importCurriculumItemRow{}, fmt.Errorf("course: %w", err)
	}
	if course < 1 || course > 6 {
		return importCurriculumItemRow{}, fmt.Errorf("course must be 1..6")
	}
	semester, err := parseInt16(get("semester"))
	if err != nil {
		return importCurriculumItemRow{}, fmt.Errorf("semester: %w", err)
	}
	if semester < 1 || semester > 12 {
		return importCurriculumItemRow{}, fmt.Errorf("semester must be 1..12")
	}
	hours, err := strconv.Atoi(strings.TrimSpace(get("hours_total")))
	if err != nil {
		return importCurriculumItemRow{}, fmt.Errorf("hours_total: invalid int")
	}
	if hours < 0 || hours > 10000 {
		return importCurriculumItemRow{}, fmt.Errorf("hours_total must be 0..10000")
	}
	itemType := strings.ToUpper(strings.TrimSpace(get("item_type")))
	if itemType == "" {
		itemType = "DISCIPLINE"
	}
	subject := strings.TrimSpace(get("subject"))
	if subject == "" {
		subject = discipline
	}

	return importCurriculumItemRow{
		Discipline:  discipline,
		Course:      course,
		Semester:    semester,
		HoursTotal:  hours,
		ItemType:    itemType,
		SubjectName: subject,
	}, nil
}

func parseStudyCalendarRecord(get func(string) string) (importStudyCalendarRow, error) {
	var groupID *int
	if raw := strings.TrimSpace(get("group_id")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return importStudyCalendarRow{}, fmt.Errorf("group_id: invalid int")
		}
		groupID = &v
	}
	groupName := strings.TrimSpace(get("group"))
	if groupID == nil && groupName == "" {
		return importStudyCalendarRow{}, fmt.Errorf("group_id or group required")
	}

	week, err := parseInt16(get("week_number"))
	if err != nil {
		return importStudyCalendarRow{}, fmt.Errorf("week_number: %w", err)
	}
	if week < 1 || week > 60 {
		return importStudyCalendarRow{}, fmt.Errorf("week_number must be 1..60")
	}

	var weekStart *time.Time
	if raw := strings.TrimSpace(get("week_start_date")); raw != "" {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return importStudyCalendarRow{}, fmt.Errorf("week_start_date must be YYYY-MM-DD")
		}
		d = mondayOfWeekHTTP(d)
		weekStart = &d
	}

	code := strings.TrimSpace(get("activity_code"))
	name := strings.TrimSpace(get("activity_name"))
	if code == "" && name == "" {
		return importStudyCalendarRow{}, fmt.Errorf("activity_code or activity_name required")
	}
	if name == "" {
		name = code
	}
	if code == "" {
		code = name
	}
	allows, err := parseBoolFlexible(get("allows_lessons"))
	if err != nil {
		return importStudyCalendarRow{}, fmt.Errorf("allows_lessons: %w", err)
	}
	var comment *string
	if raw := strings.TrimSpace(get("comment")); raw != "" {
		comment = &raw
	}

	return importStudyCalendarRow{
		GroupID:       groupID,
		GroupName:     groupName,
		WeekNumber:    week,
		WeekStartDate: weekStart,
		ActivityCode:  code,
		ActivityName:  name,
		ActivityKind:  strings.ToUpper(strings.TrimSpace(get("activity_kind"))),
		AllowsLessons: allows,
		Comment:       comment,
	}, nil
}

func parseInt16(s string) (int16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("required")
	}
	v, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid int")
	}
	return int16(v), nil
}

func parseBoolFlexible(s string) (bool, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "true", "1", "yes", "y", "да", "д", "можно":
		return true, nil
	case "false", "0", "no", "n", "нет", "н", "нельзя":
		return false, nil
	case "":
		return false, fmt.Errorf("required")
	default:
		return false, fmt.Errorf("invalid bool")
	}
}

func recordIsEmpty(rec []string) bool {
	for _, v := range rec {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func mondayOfWeekHTTP(t time.Time) time.Time {
	y, m, d := t.Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}
