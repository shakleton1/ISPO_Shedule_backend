package httpapi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"ispo-schedule/internal/schedule"
)

func TestParseTemplatesCSV_Comma(t *testing.T) {
	csvData := strings.Join([]string{
		"day_of_week,week_parity,pair_number,subject,location,teacher_name,subgroup",
		"1,numerator,2,Math,101,Dr A,1",
		"6,both,8,Physics,Lab,,",
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
