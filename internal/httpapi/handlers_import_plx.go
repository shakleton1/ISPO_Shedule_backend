package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ispo-schedule/internal/schedule"
)

const defaultPLXAcademicYearStart = "2026-09-01"

var (
	plxSpecialtyCodeRE = regexp.MustCompile(`\d{2}\.\d{2}\.\d{2}`)
	plxSemesterRE      = regexp.MustCompile(`(?i)семестр\s+(\d+)`)
	plxWeeksRE         = regexp.MustCompile(`(\d+)\s*нед`)
	plxDigitsRE        = regexp.MustCompile(`\d+`)
)

type plxCurriculumImportOptions struct {
	Filename          string
	SpecialtyCode     string
	SpecialtyName     string
	AdmissionYear     int16
	BaseGrade         *int16
	Variant           string
	AcademicYearStart time.Time
	Replace           bool
}

type plxParsedCurriculum struct {
	SpecialtyCode     string
	SpecialtyName     string
	AdmissionYear     int16
	BaseGrade         *int16
	Variant           string
	Title             string
	AcademicYearStart time.Time
	Items             []plxParsedItem
	CalendarWeeks     []schedule.AcademicCalendarWeek
	Warnings          []string
}

type plxParsedItem struct {
	ParentIndex int
	IndexCode   *string
	Name        string
	ItemType    string
	IsCounted   bool
	Allocations []schedule.CurriculumItemAllocation
}

type plxSemesterBlock struct {
	Semester int16
	Weeks    *int16
	Start    int
	End      int
	Columns  map[string][]int
}

type plxAssessmentColumns struct {
	Exam         int
	Credit       int
	GradedCredit int
	Other        int
}

// HandleAdminImportPLXCurriculumXLSXForTest exports the PLX import handler for integration tests.
func HandleAdminImportPLXCurriculumXLSXForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminImportPLXCurriculumXLSX(repo)
}

func handleAdminImportPLXCurriculumXLSX(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file", "upload too large")
				return
			}
			writeValidationError(c, "file", "missing file")
			return
		}
		f, err := fh.Open()
		if err != nil {
			writeValidationError(c, "file", "cannot open upload")
			return
		}
		defer f.Close()

		opts, ok := plxOptionsFromRequest(c, fh.Filename)
		if !ok {
			return
		}
		parsed, err := parsePLXCurriculumXLSX(f, opts)
		if err != nil {
			writeValidationError(c, "file", err.Error())
			return
		}
		result, err := importPLXCurriculum(repo, parsed, opts.Replace)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "import", "plx_curriculum", fmt.Sprintf("curriculum:%d", result.CurriculumID), gin.H{
			"items":          result.Items,
			"allocations":    result.Allocations,
			"calendar_weeks": result.CalendarWeeks,
		})
		c.JSON(http.StatusOK, result)
	}
}

type plxImportResult struct {
	Imported        int      `json:"imported"`
	SpecialtyID     int      `json:"specialty_id"`
	CurriculumID    int64    `json:"curriculum_id"`
	Items           int      `json:"items"`
	Allocations     int      `json:"allocations"`
	CalendarID      *int64   `json:"calendar_id,omitempty"`
	CalendarWeeks   int      `json:"calendar_weeks"`
	AcademicYear    string   `json:"academic_year_start"`
	Warnings        []string `json:"warnings,omitempty"`
	ScheduleVersion string   `json:"schedule_version"`
}

func plxOptionsFromRequest(c *gin.Context, filename string) (plxCurriculumImportOptions, bool) {
	opts := plxCurriculumImportOptions{
		Filename: filename,
		Replace:  true,
	}
	if raw := requestValue(c, "academic_year_start"); raw != "" {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeValidationError(c, "academic_year_start", "must be YYYY-MM-DD")
			return opts, false
		}
		opts.AcademicYearStart = d
	} else {
		opts.AcademicYearStart, _ = time.Parse("2006-01-02", defaultPLXAcademicYearStart)
	}
	opts.SpecialtyCode = strings.TrimSpace(requestValue(c, "specialty_code"))
	opts.SpecialtyName = strings.TrimSpace(requestValue(c, "specialty_name"))
	opts.Variant = strings.TrimSpace(requestValue(c, "variant"))

	if raw := strings.TrimSpace(requestValue(c, "admission_year")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 2000 || v > 2100 {
			writeValidationError(c, "admission_year", "must be 2000..2100")
			return opts, false
		}
		opts.AdmissionYear = int16(v)
	}
	if raw := strings.TrimSpace(requestValue(c, "base_grade")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || (v != 9 && v != 11) {
			writeValidationError(c, "base_grade", "must be 9 or 11")
			return opts, false
		}
		grade := int16(v)
		opts.BaseGrade = &grade
	}
	if raw := strings.TrimSpace(requestValue(c, "replace")); raw != "" {
		v, err := parseBoolFlexible(raw)
		if err != nil {
			writeValidationError(c, "replace", "invalid bool")
			return opts, false
		}
		opts.Replace = v
	}
	return opts, true
}

func requestValue(c *gin.Context, key string) string {
	if v := strings.TrimSpace(c.Query(key)); v != "" {
		return v
	}
	return strings.TrimSpace(c.PostForm(key))
}

func parsePLXCurriculumXLSX(r io.Reader, opts plxCurriculumImportOptions) (*plxParsedCurriculum, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	parsed := &plxParsedCurriculum{
		SpecialtyCode:     strings.TrimSpace(opts.SpecialtyCode),
		SpecialtyName:     strings.TrimSpace(opts.SpecialtyName),
		AdmissionYear:     opts.AdmissionYear,
		BaseGrade:         opts.BaseGrade,
		Variant:           strings.TrimSpace(opts.Variant),
		AcademicYearStart: dateOnlyHTTP(opts.AcademicYearStart),
		Warnings:          []string{},
	}
	inferPLXMetadata(parsed, opts.Filename)
	if parsed.SpecialtyCode == "" {
		return nil, fmt.Errorf("specialty_code required or must be present in file name")
	}
	if parsed.SpecialtyName == "" {
		parsed.SpecialtyName = parsed.SpecialtyCode
	}
	if parsed.AdmissionYear == 0 {
		return nil, fmt.Errorf("admission_year required or must be present in file name")
	}
	if parsed.Variant == "" {
		if parsed.BaseGrade != nil {
			parsed.Variant = fmt.Sprintf("base_%d", *parsed.BaseGrade)
		} else {
			parsed.Variant = "plx"
		}
	}
	parsed.Title = fmt.Sprintf("Учебный план %s %d", parsed.SpecialtyCode, parsed.AdmissionYear)
	if parsed.BaseGrade != nil {
		parsed.Title = fmt.Sprintf("%s (%d кл.)", parsed.Title, *parsed.BaseGrade)
	}

	planRows, err := rowsBySheetName(f, "План")
	if err != nil {
		return nil, err
	}
	items, err := parsePLXPlanSheet(planRows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("plan sheet has no curriculum items")
	}
	parsed.Items = items

	graphRows, err := rowsBySheetName(f, "График")
	if err == nil {
		weeks, warnings := parsePLXGraphSheet(graphRows, parsed.AdmissionYear, parsed.AcademicYearStart)
		parsed.CalendarWeeks = weeks
		parsed.Warnings = append(parsed.Warnings, warnings...)
	} else {
		parsed.Warnings = append(parsed.Warnings, err.Error())
	}
	return parsed, nil
}

func inferPLXMetadata(parsed *plxParsedCurriculum, filename string) {
	base := filepath.Base(filename)
	if parsed.SpecialtyCode == "" {
		parsed.SpecialtyCode = plxSpecialtyCodeRE.FindString(base)
	}
	nums := plxDigitsRE.FindAllString(base, -1)
	// Typical PLX file names:
	// 09.02.01_24_11.plx.xlsx -> 09,02,01,24,11
	// 09.02.07 ... 2023 (4 курс).plx.xlsx -> 09,02,07,2023,4
	if parsed.AdmissionYear == 0 {
		for i := 3; i < len(nums); i++ {
			v, err := strconv.Atoi(nums[i])
			if err != nil {
				continue
			}
			if v >= 2000 && v <= 2100 {
				parsed.AdmissionYear = int16(v)
				if parsed.BaseGrade == nil && i+1 < len(nums) {
					if grade, ok := parsePLXBaseGradeToken(nums[i+1]); ok {
						parsed.BaseGrade = &grade
					}
				}
				break
			}
			if v >= 0 && v <= 99 {
				parsed.AdmissionYear = int16(2000 + v)
				if parsed.BaseGrade == nil && i+1 < len(nums) {
					if grade, ok := parsePLXBaseGradeToken(nums[i+1]); ok {
						parsed.BaseGrade = &grade
					}
				}
				break
			}
		}
	}
	if parsed.BaseGrade == nil {
		for i := len(nums) - 1; i >= 0; i-- {
			if grade, ok := parsePLXBaseGradeToken(nums[i]); ok {
				parsed.BaseGrade = &grade
				break
			}
		}
	}
}

func parsePLXBaseGradeToken(raw string) (int16, bool) {
	switch strings.TrimLeft(raw, "0") {
	case "9":
		return 9, true
	case "11":
		return 11, true
	default:
		return 0, false
	}
}

func rowsBySheetName(f *excelize.File, expected string) ([][]string, error) {
	for _, sheet := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(sheet), expected) {
			rows, err := f.GetRows(sheet)
			if err != nil {
				return nil, fmt.Errorf("read sheet %s: %w", expected, err)
			}
			return rows, nil
		}
	}
	for _, sheet := range f.GetSheetList() {
		if strings.Contains(strings.ToLower(sheet), strings.ToLower(expected)) {
			rows, err := f.GetRows(sheet)
			if err != nil {
				return nil, fmt.Errorf("read sheet %s: %w", expected, err)
			}
			return rows, nil
		}
	}
	return nil, fmt.Errorf("sheet %s not found", expected)
}

func parsePLXPlanSheet(rows [][]string) ([]plxParsedItem, error) {
	headerRow := findPLXPlanHeaderRow(rows)
	if headerRow < 0 {
		return nil, fmt.Errorf("plan header row not found")
	}
	if headerRow < 1 {
		return nil, fmt.Errorf("semester header row not found")
	}
	semesterBlocks := parsePLXSemesterBlocks(rows[headerRow-1], rows[headerRow])
	if len(semesterBlocks) == 0 {
		return nil, fmt.Errorf("semester blocks not found")
	}
	assessmentCols := parsePLXAssessmentColumns(rows[headerRow])

	items := []plxParsedItem{}
	currentRoot := -1
	lastHeader := -1
	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]
		if recordIsEmpty(row) {
			continue
		}
		marker := strings.TrimSpace(cell(row, 0))
		indexCode := strings.TrimSpace(cell(row, 1))
		name := strings.TrimSpace(cell(row, 2))

		if marker == "+" {
			if name == "" {
				continue
			}
			item := plxParsedItem{
				ParentIndex: lastHeader,
				IndexCode:   stringPtrIfNotEmpty(indexCode),
				Name:        name,
				ItemType:    inferPLXItemType(indexCode, name, true),
				IsCounted:   true,
				Allocations: parsePLXAllocations(row, semesterBlocks, assessmentCols),
			}
			items = append(items, item)
			continue
		}

		rawHeader := ""
		if indexCode == "" && name == "" {
			rawHeader = strings.TrimSpace(marker)
		} else if name != "" {
			rawHeader = name
		}
		if rawHeader == "" || isPLXSummaryRow(rawHeader) {
			continue
		}
		headerIndex, headerName := splitPLXHeader(rawHeader)
		if indexCode != "" {
			headerIndex = indexCode
		}
		if name != "" {
			headerName = name
		}
		if headerName == "" {
			continue
		}
		parent := currentRoot
		if isPLXRootHeader(headerIndex, headerName) {
			parent = -1
			currentRoot = len(items)
		}
		item := plxParsedItem{
			ParentIndex: parent,
			IndexCode:   stringPtrIfNotEmpty(headerIndex),
			Name:        headerName,
			ItemType:    inferPLXItemType(headerIndex, headerName, false),
			IsCounted:   false,
			Allocations: parsePLXAllocations(row, semesterBlocks, assessmentCols),
		}
		items = append(items, item)
		lastHeader = len(items) - 1
		if isPLXRootHeader(headerIndex, headerName) {
			currentRoot = lastHeader
		}
	}
	return items, nil
}

func findPLXPlanHeaderRow(rows [][]string) int {
	for i, row := range rows {
		hasName := false
		hasSemesterTotal := false
		for _, v := range row {
			n := normalizePLXCell(v)
			if n == "наименование" {
				hasName = true
			}
			if n == "итого" {
				hasSemesterTotal = true
			}
		}
		if hasName && hasSemesterTotal {
			return i
		}
	}
	return -1
}

func parsePLXSemesterBlocks(semesterHeader, columnHeader []string) []plxSemesterBlock {
	starts := []plxSemesterBlock{}
	for i, raw := range semesterHeader {
		m := plxSemesterRE.FindStringSubmatch(strings.ToLower(raw))
		if len(m) != 2 {
			continue
		}
		sem, err := strconv.Atoi(m[1])
		if err != nil || sem <= 0 || sem > 12 {
			continue
		}
		block := plxSemesterBlock{
			Semester: int16(sem),
			Start:    i,
			Columns:  map[string][]int{},
		}
		if wm := plxWeeksRE.FindStringSubmatch(strings.ToLower(raw)); len(wm) == 2 {
			if weeks, err := strconv.Atoi(wm[1]); err == nil {
				w := int16(weeks)
				block.Weeks = &w
			}
		}
		starts = append(starts, block)
	}
	if len(starts) == 0 {
		starts = parsePLXSessionBlocks(semesterHeader)
	}
	for i := range starts {
		end := len(columnHeader)
		if i+1 < len(starts) {
			end = starts[i+1].Start
		}
		starts[i].End = end
		for col := starts[i].Start; col < starts[i].End && col < len(columnHeader); col++ {
			key := normalizePLXCell(columnHeader[col])
			switch key {
			case "итого":
				starts[i].Columns["total"] = append(starts[i].Columns["total"], col)
			case "лек":
				starts[i].Columns["lectures"] = append(starts[i].Columns["lectures"], col)
			case "лаб":
				starts[i].Columns["lab"] = append(starts[i].Columns["lab"], col)
			case "пр":
				starts[i].Columns["practice"] = append(starts[i].Columns["practice"], col)
			case "ср":
				starts[i].Columns["independent"] = append(starts[i].Columns["independent"], col)
			case "патт", "пкэ", "конс":
				starts[i].Columns["exam"] = append(starts[i].Columns["exam"], col)
			}
		}
	}
	return starts
}

func parsePLXSessionBlocks(sessionHeader []string) []plxSemesterBlock {
	starts := []plxSemesterBlock{}
	for i, raw := range sessionHeader {
		key := normalizePLXCell(raw)
		if key == "" || key == "-" || !strings.Contains(key, "сессия") {
			continue
		}
		starts = append(starts, plxSemesterBlock{
			Semester: int16(len(starts) + 1),
			Start:    i,
			Columns:  map[string][]int{},
		})
	}
	return starts
}

func parsePLXAssessmentColumns(header []string) plxAssessmentColumns {
	cols := plxAssessmentColumns{Exam: -1, Credit: -1, GradedCredit: -1, Other: -1}
	for i, raw := range header {
		key := normalizePLXCell(raw)
		switch {
		case strings.Contains(key, "экзам"):
			cols.Exam = i
		case key == "зачет":
			cols.Credit = i
		case strings.Contains(key, "зачетсоц"):
			cols.GradedCredit = i
		case key == "др":
			cols.Other = i
		}
	}
	return cols
}

func parsePLXAllocations(row []string, blocks []plxSemesterBlock, assessments plxAssessmentColumns) []schedule.CurriculumItemAllocation {
	allocs := []schedule.CurriculumItemAllocation{}
	for _, block := range blocks {
		total := sumPLXColumns(row, block.Columns["total"])
		lectures := sumPLXColumns(row, block.Columns["lectures"])
		practice := sumPLXColumns(row, block.Columns["practice"])
		lab := sumPLXColumns(row, block.Columns["lab"])
		independent := sumPLXColumns(row, block.Columns["independent"])
		exam := sumPLXColumns(row, block.Columns["exam"])
		assessment := assessmentForPLXSemester(row, block.Semester, assessments)
		if total == 0 && lectures == 0 && practice == 0 && lab == 0 && independent == 0 && exam == 0 && assessment == nil {
			continue
		}
		allocs = append(allocs, schedule.CurriculumItemAllocation{
			Semester:         block.Semester,
			Weeks:            block.Weeks,
			HoursTotal:       intPtrIfPositive(total),
			HoursLectures:    intPtrIfPositive(lectures),
			HoursPractice:    intPtrIfPositive(practice),
			HoursLab:         intPtrIfPositive(lab),
			HoursIndependent: intPtrIfPositive(independent),
			HoursExam:        intPtrIfPositive(exam),
			AssessmentType:   assessment,
		})
	}
	return allocs
}

func sumPLXColumns(row []string, cols []int) int {
	sum := 0
	for _, col := range cols {
		sum += parsePLXInt(cell(row, col))
	}
	return sum
}

func assessmentForPLXSemester(row []string, semester int16, cols plxAssessmentColumns) *string {
	if cols.Exam >= 0 && plxCellContainsSemester(cell(row, cols.Exam), semester) {
		v := "EXAM"
		return &v
	}
	if cols.Credit >= 0 && plxCellContainsSemester(cell(row, cols.Credit), semester) {
		v := "CREDIT"
		return &v
	}
	if cols.GradedCredit >= 0 && plxCellContainsSemester(cell(row, cols.GradedCredit), semester) {
		v := "GRADED_CREDIT"
		return &v
	}
	if cols.Other >= 0 && plxCellContainsSemester(cell(row, cols.Other), semester) {
		v := "MODULE_EXAM"
		return &v
	}
	return nil
}

func plxCellContainsSemester(raw string, semester int16) bool {
	for _, token := range plxDigitsRE.FindAllString(raw, -1) {
		if len(token) > 1 {
			for _, r := range token {
				if int16(r-'0') == semester {
					return true
				}
			}
			continue
		}
		v, err := strconv.Atoi(token)
		if err == nil && int16(v) == semester {
			return true
		}
	}
	return false
}

func splitPLXHeader(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if idx := strings.Index(raw, "."); idx > 0 && idx < len(raw)-1 {
		prefix := strings.TrimSpace(raw[:idx])
		name := strings.TrimSpace(raw[idx+1:])
		if prefix != "" && name != "" && len([]rune(prefix)) <= 20 {
			return prefix, name
		}
	}
	return "", raw
}

func isPLXRootHeader(indexCode, name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	idx := strings.ToUpper(strings.TrimSpace(indexCode))
	return idx == "ПП" || strings.Contains(n, "профессиональная подготовка")
}

func isPLXSummaryRow(raw string) bool {
	n := strings.ToLower(strings.TrimSpace(raw))
	return n == "всего" || n == "итого" || strings.HasPrefix(n, "в том числе")
}

func inferPLXItemType(indexCode, name string, counted bool) string {
	n := strings.ToLower(indexCode + " " + name)
	switch {
	case strings.Contains(n, "государственная итоговая") || strings.Contains(n, "гиа"):
		return "GIA"
	case strings.Contains(n, "практик") || (counted && (strings.HasPrefix(strings.ToUpper(strings.TrimSpace(indexCode)), "УП") || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(indexCode)), "ПП"))):
		return "PRACTICE"
	case strings.Contains(n, "аттеста") || strings.Contains(n, "экзамен"):
		return "ATTESTATION"
	case counted:
		return "DISCIPLINE"
	default:
		return "OTHER"
	}
}

func parsePLXGraphSheet(rows [][]string, admissionYear int16, academicYearStart time.Time) ([]schedule.AcademicCalendarWeek, []string) {
	warnings := []string{}
	weekHeaderRow := findPLXGraphWeekHeaderRow(rows)
	if weekHeaderRow < 0 {
		return nil, []string{"graph week header row not found"}
	}
	courseRows := findPLXGraphCourseRows(rows)
	if len(courseRows) == 0 {
		return nil, []string{"graph course rows I..VI not found"}
	}

	weeks := []schedule.AcademicCalendarWeek{}
	for course := 1; course <= 6; course++ {
		courseRow, ok := courseRows[course]
		if !ok {
			continue
		}
		weekStart := mondayOfWeekHTTP(academicYearStart.AddDate(course-1, 0, 0))
		for col := 0; col < len(rows[weekHeaderRow]); col++ {
			weekNum := parsePLXInt(cell(rows[weekHeaderRow], col))
			if weekNum <= 0 || weekNum > 60 {
				continue
			}
			symbol := strings.TrimSpace(cell(rows[courseRow], col))
			code, name, isTeaching := mapPLXGraphActivity(symbol)
			activityName := name
			comment := ""
			if symbol != "" && symbol != "=" {
				comment = fmt.Sprintf("PLX code: %s", symbol)
			}
			week := schedule.AcademicCalendarWeek{
				CourseNumber:  int16(course),
				WeekNumber:    int16(weekNum),
				WeekStartDate: weekStart.AddDate(0, 0, (weekNum-1)*7),
				ActivityCode:  code,
				ActivityName:  &activityName,
				IsTeaching:    isTeaching,
			}
			if comment != "" {
				week.Comment = &comment
			}
			weeks = append(weeks, week)
		}
	}
	if len(weeks) == 0 {
		warnings = append(warnings, "graph has no week cells")
	}
	return weeks, warnings
}

func findPLXGraphWeekHeaderRow(rows [][]string) int {
	bestRow := -1
	bestCount := 0
	for i, row := range rows {
		count := 0
		for _, raw := range row {
			v := parsePLXInt(raw)
			if v >= 1 && v <= 60 {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			bestRow = i
		}
	}
	if bestCount >= 20 {
		return bestRow
	}
	return -1
}

func findPLXGraphCourseRows(rows [][]string) map[int]int {
	out := map[int]int{}
	for i, row := range rows {
		for col := 0; col < len(row) && col < 3; col++ {
			if course, ok := parsePLXRomanCourse(strings.TrimSpace(row[col])); ok {
				if _, exists := out[course]; !exists {
					out[course] = i
				}
			}
		}
	}
	return out
}

func findPLXGraphCourseRow(rows [][]string, course int) int {
	target := plxRomanCourse(course)
	for i, row := range rows {
		for col := 0; col < len(row) && col < 3; col++ {
			if strings.EqualFold(strings.TrimSpace(row[col]), target) {
				return i
			}
		}
	}
	return -1
}

func parsePLXRomanCourse(raw string) (int, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "I":
		return 1, true
	case "II":
		return 2, true
	case "III":
		return 3, true
	case "IV":
		return 4, true
	case "V":
		return 5, true
	case "VI":
		return 6, true
	default:
		return 0, false
	}
}

func plxRomanCourse(course int) string {
	switch course {
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	case 4:
		return "IV"
	case 5:
		return "V"
	case 6:
		return "VI"
	default:
		return ""
	}
}

func mapPLXGraphActivity(symbol string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(symbol)) {
	case "у", "уп":
		return "PRACTICE_EDU", "Учебная практика", false
	case "п":
		return "PRACTICE_PROD", "Производственная практика", false
	case "пп", "пд":
		return "PRACTICE_PREGRAD", "Преддипломная практика", false
	case "э":
		return "EXAM", "Экзаменационная сессия", true
	case "г", "гиа":
		return "GIA", "Государственная итоговая аттестация", false
	case "к":
		return "VACATION", "Каникулы", false
	default:
		return "TEACHING", "Учебные занятия", true
	}
}

func importPLXCurriculum(repo *schedule.Repository, parsed *plxParsedCurriculum, replace bool) (plxImportResult, error) {
	result := plxImportResult{
		AcademicYear: parsed.AcademicYearStart.Format("2006-01-02"),
		Warnings:     parsed.Warnings,
	}
	var calendarID *int64
	err := repo.DB().Transaction(func(tx *gorm.DB) error {
		specialtyID, err := upsertPLXSpecialty(tx, parsed.SpecialtyCode, parsed.SpecialtyName)
		if err != nil {
			return err
		}
		curriculumID, err := upsertPLXCurriculum(tx, specialtyID, parsed)
		if err != nil {
			return err
		}
		result.SpecialtyID = specialtyID
		result.CurriculumID = curriculumID
		if replace {
			if err := clearPLXCurriculumData(tx, curriculumID, parsed.AcademicYearStart); err != nil {
				return err
			}
		}
		itemIDs := make([]int64, len(parsed.Items))
		for i, item := range parsed.Items {
			var subjectID *int
			if item.IsCounted {
				id, err := getOrCreateSubjectID(tx, item.Name)
				if err != nil {
					return fmt.Errorf("item %d subject: %w", i+1, err)
				}
				subjectID = &id
			}
			var parentID *int64
			if item.ParentIndex >= 0 && item.ParentIndex < len(itemIDs) && itemIDs[item.ParentIndex] > 0 {
				parentID = &itemIDs[item.ParentIndex]
			}
			row := schedule.CurriculumItem{
				CurriculumID: curriculumID,
				ParentID:     parentID,
				IndexCode:    item.IndexCode,
				ItemType:     item.ItemType,
				Name:         item.Name,
				SubjectID:    subjectID,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("item %d: %w", i+1, err)
			}
			itemIDs[i] = row.ID
			result.Items++
			for _, alloc := range item.Allocations {
				alloc.ItemID = row.ID
				if err := tx.Create(&alloc).Error; err != nil {
					return fmt.Errorf("item %d semester %d: %w", i+1, alloc.Semester, err)
				}
				result.Allocations++
			}
		}
		if len(parsed.CalendarWeeks) > 0 {
			cal := schedule.AcademicCalendar{
				CurriculumID:      curriculumID,
				AcademicYearStart: parsed.AcademicYearStart,
				WeeksTotal:        plxWeeksTotal(parsed.CalendarWeeks),
			}
			note := "Imported from PLX"
			cal.Notes = &note
			if err := tx.Create(&cal).Error; err != nil {
				return err
			}
			calendarID = &cal.ID
			for _, week := range parsed.CalendarWeeks {
				week.CalendarID = cal.ID
				if err := tx.Exec(`
INSERT INTO academic_calendar_weeks
  (calendar_id, course_number, week_number, week_start_date, activity_code, activity_name, is_teaching, comment)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
					week.CalendarID,
					week.CourseNumber,
					week.WeekNumber,
					dateOnlyHTTP(week.WeekStartDate),
					week.ActivityCode,
					week.ActivityName,
					week.IsTeaching,
					week.Comment,
				).Error; err != nil {
					return fmt.Errorf("calendar week %d: %w", week.WeekNumber, err)
				}
				result.CalendarWeeks++
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result.CalendarID = calendarID
	result.Imported = result.Items
	ver, err := bumpScheduleVersionAndGet(repo)
	if err != nil {
		return result, err
	}
	result.ScheduleVersion = ver.UTC().Format(time.RFC3339Nano)
	return result, nil
}

func plxWeeksTotal(weeks []schedule.AcademicCalendarWeek) int16 {
	var maxWeek int16
	for _, week := range weeks {
		if week.WeekNumber > maxWeek {
			maxWeek = week.WeekNumber
		}
	}
	if maxWeek <= 0 {
		return 52
	}
	return maxWeek
}

func upsertPLXSpecialty(tx *gorm.DB, code, name string) (int, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" {
		return 0, fmt.Errorf("specialty code required")
	}
	if name == "" {
		name = code
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := tx.Raw(`
INSERT INTO specialties (code, name)
VALUES (?, ?)
ON CONFLICT (code)
DO UPDATE SET name = EXCLUDED.name
RETURNING id`, code, name).Scan(&out).Error
	return out.ID, err
}

func upsertPLXCurriculum(tx *gorm.DB, specialtyID int, parsed *plxParsedCurriculum) (int64, error) {
	notes := fmt.Sprintf("Imported from PLX; base_grade=%s", plxBaseGradeForNotes(parsed.BaseGrade))
	var out struct {
		ID int64 `gorm:"column:id"`
	}
	err := tx.Raw(`
INSERT INTO curricula (specialty_id, admission_year, variant, title, notes, is_active)
VALUES (?, ?, ?, ?, ?, TRUE)
ON CONFLICT (specialty_id, admission_year, variant)
DO UPDATE SET
  title = EXCLUDED.title,
  notes = EXCLUDED.notes,
  is_active = TRUE,
  deleted_at = NULL
RETURNING id`, specialtyID, parsed.AdmissionYear, parsed.Variant, parsed.Title, notes).Scan(&out).Error
	return out.ID, err
}

func clearPLXCurriculumData(tx *gorm.DB, curriculumID int64, academicYearStart time.Time) error {
	if err := tx.Exec(`
DELETE FROM curriculum_item_allocations
USING curriculum_items
WHERE curriculum_item_allocations.item_id = curriculum_items.id
  AND curriculum_items.curriculum_id = ?`, curriculumID).Error; err != nil {
		return err
	}
	if err := tx.Where("curriculum_id = ?", curriculumID).Delete(&schedule.CurriculumItem{}).Error; err != nil {
		return err
	}
	if err := tx.Where("curriculum_id = ? AND academic_year_start = ?", curriculumID, dateOnlyHTTP(academicYearStart)).Delete(&schedule.AcademicCalendar{}).Error; err != nil {
		return err
	}
	return nil
}

func plxBaseGradeForNotes(v *int16) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(int(*v))
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parsePLXInt(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.ReplaceAll(raw, ",", ".")
	raw = strings.ReplaceAll(raw, " ", "")
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return int(f + 0.5)
	}
	match := plxDigitsRE.FindString(raw)
	if match == "" {
		return 0
	}
	v, _ := strconv.Atoi(match)
	return v
}

func normalizePLXCell(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stringPtrIfNotEmpty(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func intPtrIfPositive(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

func dateOnlyHTTP(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
