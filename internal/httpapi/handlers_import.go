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
}

func handleAdminImportTemplatesCSV(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := parseGroupIDFromRequest(c)
		if !ok {
			return
		}

		f, err := getUploadedFile(c, "file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()

		rows, err := parseTemplatesCSV(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		inserted, ver, err := importTemplatesReplace(c, repo, groupID, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
		}
		writeAudit(c, repo, "import", "schedule_templates", fmt.Sprintf("group:%d", groupID), gin.H{"inserted": inserted})
		c.JSON(http.StatusOK, gin.H{"inserted": inserted, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func handleAdminImportTemplatesXLSX(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := parseGroupIDFromRequest(c)
		if !ok {
			return
		}

		f, err := getUploadedFile(c, "file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()

		rows, err := parseTemplatesXLSX(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		inserted, ver, err := importTemplatesReplace(c, repo, groupID, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
		}
		writeAudit(c, repo, "import", "schedule_templates", fmt.Sprintf("group:%d", groupID), gin.H{"inserted": inserted})
		c.JSON(http.StatusOK, gin.H{"inserted": inserted, "schedule_version": ver.UTC().Format(time.RFC3339Nano)})
	}
}

func parseGroupIDFromRequest(c *gin.Context) (int, bool) {
	if q := strings.TrimSpace(c.Query("group_id")); q != "" {
		gid, err := strconv.Atoi(q)
		if err != nil || gid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return 0, false
		}
		return gid, true
	}
	gidStr := strings.TrimSpace(c.PostForm("group_id"))
	if gidStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_id required"})
		return 0, false
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil || gid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return 0, false
	}
	return gid, true
}

func getUploadedFile(c *gin.Context, field string) (io.ReadCloser, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("missing %s file", field)
	}
	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("cannot open upload")
	}
	return file, nil
}

func importTemplatesReplace(c *gin.Context, repo *schedule.Repository, groupID int, rows []importTemplateRow) (int, time.Time, error) {
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

		if err := tx.Where("group_id = ?", groupID).Delete(&schedule.ScheduleTemplate{}).Error; err != nil {
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
			templates = append(templates, schedule.ScheduleTemplate{
				GroupID:     groupID,
				DayOfWeek:   r.DayOfWeek,
				WeekParity:  r.WeekParity,
				PairNumber:  r.PairNumber,
				SubjectID:   subID,
				LocationID:  locID,
				TeacherName: r.TeacherName,
				Subgroup:    r.Subgroup,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
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

	ver, err := bumpScheduleVersionAndGet(repo)
	if err != nil {
		return inserted, time.Time{}, err
	}
	return inserted, ver, nil
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
	l = schedule.Location{Name: name, IsVirtual: false}
	if err := tx.Create(&l).Error; err != nil {
		return 0, err
	}
	return l.ID, nil
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
			j := idx[key]
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
			j := idx[key]
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
		case "location", "room", "кабинет", "аудитория":
			m["location"] = i
		case "teacher_name", "teacher", "преподаватель":
			m["teacher_name"] = i
		case "subgroup", "подгруппа":
			m["subgroup"] = i
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

	return importTemplateRow{
		DayOfWeek:   day,
		WeekParity:  par,
		PairNumber:  pair,
		SubjectName: subject,
		Location:    location,
		TeacherName: teacher,
		Subgroup:    subgroup,
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
