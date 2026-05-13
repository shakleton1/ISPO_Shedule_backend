package httpapi

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"ispo-schedule/internal/schedule"
)

func TestParseTemplatesCSV_Comma(t *testing.T) {
	csvData := strings.Join([]string{
		"day_of_week,week_parity,pair_number,subject,location,teacher_name,subgroup,flow_key",
		"1,numerator,2,Math,101,Dr A,1,math-flow-1",
		"6,both,8,Physics,Lab,,,",
		"",
	}, "\n")

	rows, err := parseTemplatesCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].DayOfWeek != 0 { // 1 => 0 (Mon)
		t.Fatalf("expected day 0, got %d", rows[0].DayOfWeek)
	}
	if rows[0].WeekParity != schedule.WeekParityNumerator {
		t.Fatalf("expected numerator, got %s", rows[0].WeekParity)
	}
	if rows[0].PairNumber != 2 {
		t.Fatalf("expected pair 2, got %d", rows[0].PairNumber)
	}
	if rows[0].SubjectName != "Math" || rows[0].Location != "101" || rows[0].TeacherName != "Dr A" {
		t.Fatalf("unexpected row[0] payload: %+v", rows[0])
	}
	if rows[0].Subgroup == nil || *rows[0].Subgroup != 1 {
		t.Fatalf("expected subgroup=1, got %+v", rows[0].Subgroup)
	}
	if rows[0].FlowKey == nil || *rows[0].FlowKey != "math-flow-1" {
		t.Fatalf("expected flow_key=math-flow-1, got %+v", rows[0].FlowKey)
	}

	if rows[1].DayOfWeek != 5 { // 6 => 5 (Sat)
		t.Fatalf("expected day 5, got %d", rows[1].DayOfWeek)
	}
	if rows[1].WeekParity != schedule.WeekParityBoth {
		t.Fatalf("expected both, got %s", rows[1].WeekParity)
	}
	if rows[1].PairNumber != 8 {
		t.Fatalf("expected pair 8, got %d", rows[1].PairNumber)
	}
	if rows[1].TeacherName != "" {
		t.Fatalf("expected empty teacher, got %q", rows[1].TeacherName)
	}
	if rows[1].Subgroup != nil {
		t.Fatalf("expected nil subgroup, got %+v", rows[1].Subgroup)
	}
}

func TestParseTemplatesCSV_SemicolonDelimiterAutoDetect(t *testing.T) {
	csvData := strings.Join([]string{
		"день_недели;четность;номер_пары;предмет;аудитория;преподаватель;подгруппа",
		"2;denominator;1;История;201;Иванов И.И.;2",
	}, "\n")

	rows, err := parseTemplatesCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].DayOfWeek != 1 { // 2 => 1
		t.Fatalf("expected day 1, got %d", rows[0].DayOfWeek)
	}
	if rows[0].WeekParity != schedule.WeekParityDenominator {
		t.Fatalf("expected denominator, got %s", rows[0].WeekParity)
	}
	if rows[0].PairNumber != 1 {
		t.Fatalf("expected pair 1, got %d", rows[0].PairNumber)
	}
	if rows[0].SubjectName != "История" {
		t.Fatalf("unexpected subject: %q", rows[0].SubjectName)
	}
	if rows[0].Subgroup == nil || *rows[0].Subgroup != 2 {
		t.Fatalf("expected subgroup=2, got %+v", rows[0].Subgroup)
	}
}

func TestParseTemplatesCSV_MissingRequiredColumn(t *testing.T) {
	csvData := strings.Join([]string{
		"day_of_week,week_parity,pair_number,subject,teacher_name",
		"1,numerator,2,Math,Dr A",
	}, "\n")

	_, err := parseTemplatesCSV(strings.NewReader(csvData))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing required column") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestParseTemplatesXLSX_Basic(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	// Headers
	_ = f.SetCellValue(sheet, "A1", "day_of_week")
	_ = f.SetCellValue(sheet, "B1", "week_parity")
	_ = f.SetCellValue(sheet, "C1", "pair_number")
	_ = f.SetCellValue(sheet, "D1", "subject")
	_ = f.SetCellValue(sheet, "E1", "location")
	_ = f.SetCellValue(sheet, "F1", "teacher_name")
	_ = f.SetCellValue(sheet, "G1", "subgroup")

	// Row 2
	_ = f.SetCellValue(sheet, "A2", 0)
	_ = f.SetCellValue(sheet, "B2", "numerator")
	_ = f.SetCellValue(sheet, "C2", 3)
	_ = f.SetCellValue(sheet, "D2", "Math")
	_ = f.SetCellValue(sheet, "E2", "101")
	_ = f.SetCellValue(sheet, "F2", "Dr A")
	_ = f.SetCellValue(sheet, "G2", 1)

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}

	rows, err := parseTemplatesXLSX(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].DayOfWeek != 0 || rows[0].PairNumber != 3 || rows[0].WeekParity != schedule.WeekParityNumerator {
		t.Fatalf("unexpected parsed row: %+v", rows[0])
	}
}

func TestParseTemplatesXLSX_InvalidRow(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "day_of_week")
	_ = f.SetCellValue(sheet, "B1", "week_parity")
	_ = f.SetCellValue(sheet, "C1", "pair_number")
	_ = f.SetCellValue(sheet, "D1", "subject")
	_ = f.SetCellValue(sheet, "E1", "location")
	_ = f.SetCellValue(sheet, "F1", "teacher_name")

	// Invalid pair_number
	_ = f.SetCellValue(sheet, "A2", 1)
	_ = f.SetCellValue(sheet, "B2", "both")
	_ = f.SetCellValue(sheet, "C2", 99)
	_ = f.SetCellValue(sheet, "D2", "Math")
	_ = f.SetCellValue(sheet, "E2", "101")
	_ = f.SetCellValue(sheet, "F2", "")

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}

	_, err = parseTemplatesXLSX(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "pair_number") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestParseCurriculumItemsCSV_Semicolon(t *testing.T) {
	csvData := strings.Join([]string{
		"дисциплина;курс;семестр;часы",
		"Математика;1;1;72",
	}, "\n")

	rows, err := parseCurriculumItemsCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Discipline != "Математика" || rows[0].Course != 1 || rows[0].Semester != 1 || rows[0].HoursTotal != 72 {
		t.Fatalf("unexpected parsed row: %+v", rows[0])
	}
	if rows[0].ItemType != "DISCIPLINE" || rows[0].SubjectName != "Математика" {
		t.Fatalf("unexpected defaults: %+v", rows[0])
	}
}

func TestParseCurriculumItemsCSV_SemesterCourseMismatchDetectedByImportRecord(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "discipline":
			return "History"
		case "course":
			return "1"
		case "semester":
			return "3"
		case "hours_total":
			return "40"
		default:
			return ""
		}
	}
	row, err := parseCurriculumItemRecord(get)
	if err != nil {
		t.Fatalf("parse record should only validate field ranges: %v", err)
	}
	if row.Semester != 3 {
		t.Fatalf("expected semester 3, got %d", row.Semester)
	}
}

func TestParseStudyCalendarCSV_RussianBooleans(t *testing.T) {
	csvData := strings.Join([]string{
		"группа;неделя_номер;занятость;полное_наименование;разрешены_ли_пары",
		"22290907/1095;1;PRACTICE;Производственная практика;Нет",
		"22290907/1095;8;EXAM;Экзамен;Да",
	}, "\n")

	rows, err := parseStudyCalendarCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].GroupName != "22290907/1095" || rows[0].WeekNumber != 1 || rows[0].ActivityCode != "PRACTICE" || rows[0].AllowsLessons {
		t.Fatalf("unexpected row[0]: %+v", rows[0])
	}
	if rows[1].ActivityName != "Экзамен" || !rows[1].AllowsLessons {
		t.Fatalf("unexpected row[1]: %+v", rows[1])
	}
}

func TestParseStudyCalendarXLSX_Basic(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "group_id")
	_ = f.SetCellValue(sheet, "B1", "week_number")
	_ = f.SetCellValue(sheet, "C1", "activity_code")
	_ = f.SetCellValue(sheet, "D1", "activity_name")
	_ = f.SetCellValue(sheet, "E1", "allows_lessons")
	_ = f.SetCellValue(sheet, "A2", 10)
	_ = f.SetCellValue(sheet, "B2", 2)
	_ = f.SetCellValue(sheet, "C2", "PRACTICE")
	_ = f.SetCellValue(sheet, "D2", "Practice")
	_ = f.SetCellValue(sheet, "E2", "no")

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	rows, err := parseStudyCalendarXLSX(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(rows) != 1 || rows[0].GroupID == nil || *rows[0].GroupID != 10 || rows[0].AllowsLessons {
		t.Fatalf("unexpected parsed rows: %+v", rows)
	}
}

func TestParsePLXCurriculumXLSX_Basic(t *testing.T) {
	f := newPLXTestWorkbook(t)
	defer func() { _ = f.Close() }()

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	academicStart, _ := time.Parse("2006-01-02", "2026-09-01")
	parsed, err := parsePLXCurriculumXLSX(bytes.NewReader(buf.Bytes()), plxCurriculumImportOptions{
		Filename:          "99.02.01_26_11.plx.xlsx",
		AcademicYearStart: academicStart,
		Replace:           true,
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if parsed.SpecialtyCode != "99.02.01" || parsed.AdmissionYear != 2026 || parsed.BaseGrade == nil || *parsed.BaseGrade != 11 {
		t.Fatalf("unexpected metadata: %+v", parsed)
	}
	if parsed.Variant != "base_11" {
		t.Fatalf("expected base_11 variant, got %q", parsed.Variant)
	}
	if len(parsed.Items) != 3 {
		t.Fatalf("expected 3 items, got %d: %+v", len(parsed.Items), parsed.Items)
	}
	discipline := parsed.Items[2]
	if discipline.Name != "История России" || discipline.ItemType != "DISCIPLINE" || discipline.IndexCode == nil || *discipline.IndexCode != "ОГСЭ.01" {
		t.Fatalf("unexpected discipline item: %+v", discipline)
	}
	if discipline.ParentIndex != 1 {
		t.Fatalf("expected discipline parent index 1, got %d", discipline.ParentIndex)
	}
	if len(discipline.Allocations) != 2 {
		t.Fatalf("expected 2 allocations, got %d: %+v", len(discipline.Allocations), discipline.Allocations)
	}
	if discipline.Allocations[0].Semester != 1 || discipline.Allocations[0].HoursTotal == nil || *discipline.Allocations[0].HoursTotal != 34 {
		t.Fatalf("unexpected semester 1 allocation: %+v", discipline.Allocations[0])
	}
	if discipline.Allocations[1].Semester != 2 || discipline.Allocations[1].AssessmentType == nil || *discipline.Allocations[1].AssessmentType != "GRADED_CREDIT" {
		t.Fatalf("unexpected semester 2 allocation: %+v", discipline.Allocations[1])
	}
	if len(parsed.CalendarWeeks) != 52 {
		t.Fatalf("expected 52 calendar weeks, got %d", len(parsed.CalendarWeeks))
	}
	if parsed.CalendarWeeks[1].ActivityCode != "PRACTICE" || parsed.CalendarWeeks[1].IsTeaching {
		t.Fatalf("unexpected week 2 calendar cell: %+v", parsed.CalendarWeeks[1])
	}
	if parsed.CalendarWeeks[2].ActivityCode != "EXAM" || !parsed.CalendarWeeks[2].IsTeaching {
		t.Fatalf("unexpected week 3 calendar cell: %+v", parsed.CalendarWeeks[2])
	}
	if parsed.CalendarWeeks[3].ActivityCode != "VACATION" || parsed.CalendarWeeks[3].IsTeaching {
		t.Fatalf("unexpected week 4 calendar cell: %+v", parsed.CalendarWeeks[3])
	}
}

func TestParsePLXCurriculumXLSX_InferFourDigitAdmissionYear(t *testing.T) {
	f := newPLXTestWorkbook(t)
	defer func() { _ = f.Close() }()

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	academicStart, _ := time.Parse("2006-01-02", "2026-09-01")
	parsed, err := parsePLXCurriculumXLSX(bytes.NewReader(buf.Bytes()), plxCurriculumImportOptions{
		Filename:          "09.02.07 Инф.сист. и прогр. 2023 (4 курс).plx.xlsx",
		BaseGrade:         int16Ptr(9),
		AcademicYearStart: academicStart,
		Replace:           true,
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if parsed.AdmissionYear != 2023 {
		t.Fatalf("expected admission year 2023, got %d", parsed.AdmissionYear)
	}
	if parsed.BaseGrade == nil || *parsed.BaseGrade != 9 {
		t.Fatalf("expected base grade 9, got %+v", parsed.BaseGrade)
	}
}

func TestParsePLXCurriculumXLSX_SessionBlocks(t *testing.T) {
	f := newPLXSessionWorkbook(t)
	defer func() { _ = f.Close() }()

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	academicStart, _ := time.Parse("2006-01-02", "2026-09-01")
	parsed, err := parsePLXCurriculumXLSX(bytes.NewReader(buf.Bytes()), plxCurriculumImportOptions{
		Filename:          "38.02.01_26_11.plx.xlsx",
		AcademicYearStart: academicStart,
		Replace:           true,
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(parsed.Items), parsed.Items)
	}
	discipline := parsed.Items[1]
	if discipline.Name != "История России" {
		t.Fatalf("unexpected discipline: %+v", discipline)
	}
	if len(discipline.Allocations) != 2 {
		t.Fatalf("expected 2 session allocations, got %d: %+v", len(discipline.Allocations), discipline.Allocations)
	}
	if discipline.Allocations[0].Semester != 1 || discipline.Allocations[0].HoursTotal == nil || *discipline.Allocations[0].HoursTotal != 10 {
		t.Fatalf("unexpected first session allocation: %+v", discipline.Allocations[0])
	}
	if discipline.Allocations[1].Semester != 2 || discipline.Allocations[1].HoursTotal == nil || *discipline.Allocations[1].HoursTotal != 12 {
		t.Fatalf("unexpected second session allocation: %+v", discipline.Allocations[1])
	}
}

func newPLXTestWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	plan := f.GetSheetName(0)
	if err := f.SetSheetName(plan, "План"); err != nil {
		t.Fatalf("SetSheetName: %v", err)
	}
	plan = "План"
	if _, err := f.NewSheet("График"); err != nil {
		t.Fatalf("NewSheet: %v", err)
	}

	_ = f.SetCellValue(plan, "Q1", "Курс 1")
	_ = f.SetCellValue(plan, "Q2", "Семестр 1 [16 нед]")
	_ = f.SetCellValue(plan, "Y2", "Семестр 2 [21 нед]")
	_ = f.SetCellValue(plan, "A3", "Считать в плане")
	_ = f.SetCellValue(plan, "B3", "Индекс")
	_ = f.SetCellValue(plan, "C3", "Наименование")
	_ = f.SetCellValue(plan, "D3", "Экза мен")
	_ = f.SetCellValue(plan, "E3", "Зачет")
	_ = f.SetCellValue(plan, "F3", "Зачет с оц.")
	_ = f.SetCellValue(plan, "H3", "Др")
	_ = f.SetCellValue(plan, "Q3", "Итого")
	_ = f.SetCellValue(plan, "R3", "Ауд.")
	_ = f.SetCellValue(plan, "S3", "Лек")
	_ = f.SetCellValue(plan, "T3", "Лаб")
	_ = f.SetCellValue(plan, "U3", "Пр")
	_ = f.SetCellValue(plan, "V3", "СР")
	_ = f.SetCellValue(plan, "W3", "ПАтт")
	_ = f.SetCellValue(plan, "Y3", "Итого")
	_ = f.SetCellValue(plan, "Z3", "Ауд.")
	_ = f.SetCellValue(plan, "AA3", "Лек")
	_ = f.SetCellValue(plan, "AB3", "Пр")
	_ = f.SetCellValue(plan, "AC3", "ПАтт")
	_ = f.SetCellValue(plan, "A4", "ПП.ПРОФЕССИОНАЛЬНАЯ ПОДГОТОВКА")
	_ = f.SetCellValue(plan, "A5", "СГЦ.Социально-гуманитарный цикл")
	_ = f.SetCellValue(plan, "A6", "+")
	_ = f.SetCellValue(plan, "B6", "ОГСЭ.01")
	_ = f.SetCellValue(plan, "C6", "История России")
	_ = f.SetCellValue(plan, "F6", 2)
	_ = f.SetCellValue(plan, "Q6", 34)
	_ = f.SetCellValue(plan, "R6", 34)
	_ = f.SetCellValue(plan, "S6", 34)
	_ = f.SetCellValue(plan, "Y6", 32)
	_ = f.SetCellValue(plan, "Z6", 32)
	_ = f.SetCellValue(plan, "AA6", 32)

	graph := "График"
	for week := 1; week <= 52; week++ {
		cellName, err := excelize.CoordinatesToCellName(week+1, 4)
		if err != nil {
			t.Fatalf("CoordinatesToCellName: %v", err)
		}
		_ = f.SetCellValue(graph, cellName, week)
	}
	_ = f.SetCellValue(graph, "A13", "I")
	_ = f.SetCellValue(graph, "B13", "=")
	_ = f.SetCellValue(graph, "C13", "У")
	_ = f.SetCellValue(graph, "D13", "Э")
	_ = f.SetCellValue(graph, "E13", "К")
	return f
}

func newPLXSessionWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	plan := f.GetSheetName(0)
	if err := f.SetSheetName(plan, "План"); err != nil {
		t.Fatalf("SetSheetName: %v", err)
	}
	plan = "План"
	if _, err := f.NewSheet("График"); err != nil {
		t.Fatalf("NewSheet: %v", err)
	}

	_ = f.SetCellValue(plan, "Q1", "Курс 1")
	_ = f.SetCellValue(plan, "R2", "Установочная сессия")
	_ = f.SetCellValue(plan, "AD2", "Зимняя сессия")
	_ = f.SetCellValue(plan, "A3", "Считать в плане")
	_ = f.SetCellValue(plan, "B3", "Индекс")
	_ = f.SetCellValue(plan, "C3", "Наименование")
	_ = f.SetCellValue(plan, "D3", "Экза мен")
	_ = f.SetCellValue(plan, "E3", "Зачет")
	_ = f.SetCellValue(plan, "F3", "Зачет с оц.")
	_ = f.SetCellValue(plan, "R3", "Итого")
	_ = f.SetCellValue(plan, "S3", "Ауд.")
	_ = f.SetCellValue(plan, "T3", "Лек")
	_ = f.SetCellValue(plan, "AD3", "Итого")
	_ = f.SetCellValue(plan, "AE3", "Ауд.")
	_ = f.SetCellValue(plan, "AF3", "Пр")
	_ = f.SetCellValue(plan, "A4", "ПП.ПРОФЕССИОНАЛЬНАЯ ПОДГОТОВКА")
	_ = f.SetCellValue(plan, "A5", "+")
	_ = f.SetCellValue(plan, "B5", "СГ.01")
	_ = f.SetCellValue(plan, "C5", "История России")
	_ = f.SetCellValue(plan, "R5", 10)
	_ = f.SetCellValue(plan, "S5", 10)
	_ = f.SetCellValue(plan, "T5", 6)
	_ = f.SetCellValue(plan, "AD5", 12)
	_ = f.SetCellValue(plan, "AE5", 12)
	_ = f.SetCellValue(plan, "AF5", 8)

	graph := "График"
	_ = f.SetCellValue(graph, "B4", 1)
	_ = f.SetCellValue(graph, "C4", 2)
	_ = f.SetCellValue(graph, "A13", "I")
	_ = f.SetCellValue(graph, "B13", "=")
	_ = f.SetCellValue(graph, "C13", "=")
	return f
}
