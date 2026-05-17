package httpapi

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"ispo-schedule/internal/pdf"
	"ispo-schedule/internal/schedule"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type twoWeekScheduleExportData struct {
	Title       string
	Subtitle    string
	GeneratedAt string
	Pairs       []int
	Weeks       []scheduleExportWeek
}

type scheduleExportWeek struct {
	Title      string
	RangeLabel string
	Days       []scheduleExportDay
}

type scheduleExportDay struct {
	DayName   string
	Date      string
	DateLabel string
	Note      string
	Cells     []scheduleExportCell
}

type scheduleExportCell struct {
	PairNumber int
	Lessons    []scheduleExportLesson
}

type scheduleExportLesson struct {
	Subject   string
	Primary   string
	Secondary string
	Location  string
	Badge     string
	Comment   string
	IsChanged bool
	IsAdded   bool
}

type overridesExportData struct {
	Title       string
	Subtitle    string
	GeneratedAt string
	RowsCount   int
	Rows        []overrideExportRow
}

type overrideExportRow struct {
	Date            string
	Pair            string
	GroupName       string
	ActionLabel     string
	SourceText      string
	ReplacementText string
	Reason          string
}

var twoWeekScheduleExportTpl = pdf.MustParseTemplate("two_week_schedule_export", twoWeekScheduleExportHTMLTemplate)
var scheduleOverridesExportTpl = pdf.MustParseTemplate("schedule_overrides_export", scheduleOverridesExportHTMLTemplate)

func buildGroupTwoWeekScheduleExportData(svc *schedule.Service, repo *schedule.Repository, groupID int, refDate time.Time) (*twoWeekScheduleExportData, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	group, err := repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}
	week1Start, week2Start, endDate := twoWeekExportBounds(refDate)
	week1, err := svc.GetRange(groupID, week1Start, week1Start.AddDate(0, 0, 5))
	if err != nil {
		return nil, err
	}
	week2, err := svc.GetRange(groupID, week2Start, endDate)
	if err != nil {
		return nil, err
	}

	return &twoWeekScheduleExportData{
		Title:       "Расписание группы " + group.Name,
		Subtitle:    formatDateRange(week1Start, endDate),
		GeneratedAt: "Сформировано: " + time.Now().UTC().Format("02.01.2006 15:04 UTC"),
		Pairs:       exportPairs(),
		Weeks: []scheduleExportWeek{
			buildGroupExportWeek("Числитель", week1Start, week1.Days),
			buildGroupExportWeek("Знаменатель", week2Start, week2.Days),
		},
	}, nil
}

func buildTeacherTwoWeekScheduleExportData(svc *schedule.Service, repo *schedule.Repository, teacherID int, refDate time.Time) (*twoWeekScheduleExportData, error) {
	if teacherID <= 0 {
		return nil, fmt.Errorf("teacher_id required")
	}
	teacher, err := repo.GetTeacher(teacherID)
	if err != nil {
		return nil, err
	}
	week1Start, _, endDate := twoWeekExportBounds(refDate)
	view, err := svc.GetScheduleView(schedule.ScheduleViewFilter{Scope: "teacher", TeacherID: &teacherID}, week1Start, endDate)
	if err != nil {
		return nil, err
	}

	return &twoWeekScheduleExportData{
		Title:       "Расписание преподавателя " + teacher.Name,
		Subtitle:    formatDateRange(week1Start, endDate),
		GeneratedAt: "Сформировано: " + time.Now().UTC().Format("02.01.2006 15:04 UTC"),
		Pairs:       exportPairs(),
		Weeks:       buildTeacherExportWeeks(view.Days, week1Start),
	}, nil
}

func buildScheduleOverridesExportData(repo *schedule.Repository, filters schedule.ScheduleOverrideFilters, startDate, endDate *time.Time) (*overridesExportData, error) {
	rows, err := repo.ListScheduleOverrideReportRows(filters)
	if err != nil {
		return nil, err
	}
	subtitle := "Все примененные операции"
	if startDate != nil || endDate != nil {
		from := "..."
		to := "..."
		if startDate != nil {
			from = formatDate(*startDate)
		}
		if endDate != nil {
			to = formatDate(*endDate)
		}
		subtitle = from + " - " + to
	}
	out := make([]overrideExportRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, overrideExportRow{
			Date:            formatDate(row.LessonDate),
			Pair:            formatPair(row.PairNumber, row.Subgroup),
			GroupName:       fallback(row.GroupName, strconv.Itoa(row.GroupID)),
			ActionLabel:     overrideActionLabel(row.ActionType),
			SourceText:      overrideSideText(row.SourceSubjectName, row.SourceTeacherName, row.SourceLocationName, row.SourceLessonFormat),
			ReplacementText: overrideSideText(row.ReplacementSubjectName, row.ReplacementTeacherName, row.ReplacementLocationName, row.ReplacementLessonFormat),
			Reason:          strings.TrimSpace(stringValue(row.Reason)),
		})
	}
	return &overridesExportData{
		Title:       "Журнал замен расписания",
		Subtitle:    subtitle,
		GeneratedAt: "Сформировано: " + time.Now().UTC().Format("02.01.2006 15:04 UTC"),
		RowsCount:   len(out),
		Rows:        out,
	}, nil
}

func buildTwoWeekSchedulePDFHTML(data *twoWeekScheduleExportData) (string, error) {
	return pdf.RenderTemplate(twoWeekScheduleExportTpl, data)
}

func buildScheduleOverridesPDFHTML(data *overridesExportData) (string, error) {
	return pdf.RenderTemplate(scheduleOverridesExportTpl, data)
}

func buildTwoWeekScheduleXLSX(data *twoWeekScheduleExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	styles, err := newExportWorkbookStyles(f)
	if err != nil {
		return nil, err
	}

	sheet := "2 недели"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	if err := configureReportPrintLayout(f, sheet); err != nil {
		return nil, err
	}
	if err := writeScheduleWorkbookHeader(f, styles, sheet, data); err != nil {
		return nil, err
	}
	nextRow := 4
	for _, week := range data.Weeks {
		var err error
		nextRow, err = writeScheduleWeekBlock(f, styles, sheet, week, nextRow)
		if err != nil {
			return nil, err
		}
		nextRow += 2
	}
	f.SetActiveSheet(0)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildScheduleOverridesXLSX(data *overridesExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	styles, err := newExportWorkbookStyles(f)
	if err != nil {
		return nil, err
	}
	sheet := "Замены"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	if err := configureSchedulePrintLayout(f, sheet); err != nil {
		return nil, err
	}
	if err := f.MergeCell(sheet, "A1", "G1"); err != nil {
		return nil, err
	}
	if err := f.SetCellValue(sheet, "A1", data.Title); err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(sheet, "A1", "G1", styles.title); err != nil {
		return nil, err
	}
	if err := f.MergeCell(sheet, "A2", "G2"); err != nil {
		return nil, err
	}
	if err := f.SetCellValue(sheet, "A2", data.Subtitle+" | "+data.GeneratedAt); err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(sheet, "A2", "G2", styles.subtitle); err != nil {
		return nil, err
	}

	headers := []string{"Дата", "Пара", "Группа", "Операция", "Было", "Стало", "Причина"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	if err := f.SetCellStyle(sheet, "A4", "G4", styles.header); err != nil {
		return nil, err
	}
	widths := []float64{13, 10, 18, 18, 38, 38, 32}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return nil, err
		}
	}
	for i, row := range data.Rows {
		r := i + 5
		values := []any{row.Date, row.Pair, row.GroupName, row.ActionLabel, row.SourceText, row.ReplacementText, row.Reason}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return nil, err
			}
		}
		if err := f.SetRowHeight(sheet, r, 46); err != nil {
			return nil, err
		}
	}
	if len(data.Rows) > 0 {
		end := len(data.Rows) + 4
		if err := f.SetCellStyle(sheet, "A5", fmt.Sprintf("G%d", end), styles.body); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type exportWorkbookStyles struct {
	title    int
	subtitle int
	header   int
	day      int
	body     int
	lesson   int
	changed  int
	empty    int
}

func newExportWorkbookStyles(f *excelize.File) (exportWorkbookStyles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "B7C3C9", Style: 1},
		{Type: "right", Color: "B7C3C9", Style: 1},
		{Type: "top", Color: "B7C3C9", Style: 1},
		{Type: "bottom", Color: "B7C3C9", Style: 1},
	}
	title, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 15, Color: "111111"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	subtitle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "111111"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	day, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "111111"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	body, err := f.NewStyle(&excelize.Style{
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "top", WrapText: true},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	lesson, err := f.NewStyle(&excelize.Style{
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "top", WrapText: true},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	changed, err := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "111111", Style: 2},
			{Type: "right", Color: "B7C3C9", Style: 1},
			{Type: "top", Color: "B7C3C9", Style: 1},
			{Type: "bottom", Color: "B7C3C9", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "top", WrapText: true},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	empty, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "9AA6AC"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return exportWorkbookStyles{}, err
	}
	return exportWorkbookStyles{title: title, subtitle: subtitle, header: header, day: day, body: body, lesson: lesson, changed: changed, empty: empty}, nil
}

func configureSchedulePrintLayout(f *excelize.File, sheet string) error {
	orientation := "landscape"
	paperSize := 9
	fitToWidth := 1
	fitToHeight := 1
	blackAndWhite := true
	fitToPage := true
	if err := f.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation:   &orientation,
		Size:          &paperSize,
		FitToWidth:    &fitToWidth,
		FitToHeight:   &fitToHeight,
		BlackAndWhite: &blackAndWhite,
	}); err != nil {
		return err
	}
	return f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fitToPage})
}

func configureReportPrintLayout(f *excelize.File, sheet string) error {
	orientation := "landscape"
	paperSize := 9
	fitToWidth := 1
	fitToHeight := 0
	blackAndWhite := true
	fitToPage := true
	if err := f.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation:   &orientation,
		Size:          &paperSize,
		FitToWidth:    &fitToWidth,
		FitToHeight:   &fitToHeight,
		BlackAndWhite: &blackAndWhite,
	}); err != nil {
		return err
	}
	return f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fitToPage})
}

func writeScheduleWorkbookHeader(f *excelize.File, styles exportWorkbookStyles, sheet string, data *twoWeekScheduleExportData) error {
	if err := f.MergeCell(sheet, "A1", "I1"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "A1", data.Title); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", "I1", styles.title); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "A2", "I2"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "A2", data.Subtitle+" | 2 недели на одном листе | "+data.GeneratedAt); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A2", "I2", styles.subtitle); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet, "A", "A", 13); err != nil {
		return err
	}
	return f.SetColWidth(sheet, "B", "I", 16)
}

func writeScheduleWeekBlock(f *excelize.File, styles exportWorkbookStyles, sheet string, week scheduleExportWeek, startRow int) (int, error) {
	if err := f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("I%d", startRow)); err != nil {
		return startRow, err
	}
	if err := f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), week.Title+" | "+week.RangeLabel); err != nil {
		return startRow, err
	}
	if err := f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("I%d", startRow), styles.subtitle); err != nil {
		return startRow, err
	}

	headerRow := startRow + 1
	headers := []string{"День", "1 пара", "2 пара", "3 пара", "4 пара", "5 пара", "6 пара", "7 пара", "8 пара"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return startRow, err
		}
	}
	if err := f.SetCellStyle(sheet, fmt.Sprintf("A%d", headerRow), fmt.Sprintf("I%d", headerRow), styles.header); err != nil {
		return startRow, err
	}
	for i, day := range week.Days {
		row := headerRow + 1 + i
		if err := f.SetRowHeight(sheet, row, 42); err != nil {
			return startRow, err
		}
		dayText := day.DayName + "\n" + day.DateLabel
		if day.Note != "" {
			dayText += "\n" + day.Note
		}
		if err := f.SetCellValue(sheet, fmt.Sprintf("A%d", row), dayText); err != nil {
			return startRow, err
		}
		if err := f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styles.day); err != nil {
			return startRow, err
		}
		for _, cell := range day.Cells {
			addr, _ := excelize.CoordinatesToCellName(cell.PairNumber+1, row)
			text := scheduleCellText(cell.Lessons)
			if text == "" {
				text = "-"
			}
			if err := f.SetCellValue(sheet, addr, text); err != nil {
				return startRow, err
			}
			style := styles.lesson
			if len(cell.Lessons) == 0 {
				style = styles.empty
			} else if hasChangedLesson(cell.Lessons) {
				style = styles.changed
			}
			if err := f.SetCellStyle(sheet, addr, addr, style); err != nil {
				return startRow, err
			}
		}
	}
	return headerRow + 1 + len(week.Days), nil
}

func buildGroupExportWeek(title string, start time.Time, days []schedule.DaySchedule) scheduleExportWeek {
	out := scheduleExportWeek{
		Title:      title,
		RangeLabel: formatDateRange(start, start.AddDate(0, 0, 5)),
		Days:       make([]scheduleExportDay, 0, len(days)),
	}
	for _, d := range days {
		out.Days = append(out.Days, scheduleExportDayFromGroupDay(d))
	}
	return out
}

func buildTeacherExportWeeks(days []schedule.ScheduleViewDay, week1Start time.Time) []scheduleExportWeek {
	week1End := week1Start.AddDate(0, 0, 5)
	week2Start := week1Start.AddDate(0, 0, 7)
	week2End := week2Start.AddDate(0, 0, 5)
	weeks := []scheduleExportWeek{
		{Title: "Числитель", RangeLabel: formatDateRange(week1Start, week1End)},
		{Title: "Знаменатель", RangeLabel: formatDateRange(week2Start, week2End)},
	}
	for _, day := range days {
		d, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			continue
		}
		if day.DayOfWeek > 5 || d.After(week2End) || (d.After(week1End) && d.Before(week2Start)) {
			continue
		}
		target := 0
		if !d.Before(week2Start) {
			target = 1
		}
		weeks[target].Days = append(weeks[target].Days, scheduleExportDayFromTeacherDay(day))
	}
	for i := range weeks {
		sort.SliceStable(weeks[i].Days, func(a, b int) bool { return weeks[i].Days[a].Date < weeks[i].Days[b].Date })
	}
	return weeks
}

func scheduleExportDayFromGroupDay(d schedule.DaySchedule) scheduleExportDay {
	cells := emptyExportCells()
	byPair := map[int][]scheduleExportLesson{}
	for _, lesson := range d.Lessons {
		pair := int(lesson.PairNumber)
		byPair[pair] = append(byPair[pair], scheduleExportLesson{
			Subject:   fallback(lesson.SubjectName, "Без дисциплины"),
			Primary:   lesson.TeacherName,
			Secondary: subgroupLabel(lesson.Subgroup),
			Location:  displayLocation(lesson.LocationName, lesson.LessonFormat),
			Badge:     changedBadge(lesson.IsChanged, lesson.IsAdded),
			Comment:   strings.TrimSpace(stringValue(lesson.Comment)),
			IsChanged: lesson.IsChanged,
			IsAdded:   lesson.IsAdded,
		})
	}
	for i := range cells {
		cells[i].Lessons = byPair[cells[i].PairNumber]
	}
	return scheduleExportDay{
		DayName:   ruDayName(d.DayOfWeek),
		Date:      d.Date,
		DateLabel: formatDateString(d.Date),
		Note:      dayScheduleNote(d),
		Cells:     cells,
	}
}

func scheduleExportDayFromTeacherDay(d schedule.ScheduleViewDay) scheduleExportDay {
	cells := emptyExportCells()
	byPair := map[int][]scheduleExportLesson{}
	for _, lesson := range d.Lessons {
		pair := int(lesson.PairNumber)
		byPair[pair] = append(byPair[pair], scheduleExportLesson{
			Subject:   fallback(lesson.SubjectName, "Без дисциплины"),
			Primary:   lesson.GroupName,
			Secondary: subgroupLabel(lesson.Subgroup),
			Location:  displayLocation(lesson.LocationName, lesson.LessonFormat),
			Badge:     changedBadge(lesson.IsChanged, lesson.IsAdded),
			Comment:   strings.TrimSpace(stringValue(lesson.Comment)),
			IsChanged: lesson.IsChanged,
			IsAdded:   lesson.IsAdded,
		})
	}
	for i := range cells {
		cells[i].Lessons = byPair[cells[i].PairNumber]
	}
	return scheduleExportDay{
		DayName:   ruDayName(d.DayOfWeek),
		Date:      d.Date,
		DateLabel: formatDateString(d.Date),
		Cells:     cells,
	}
}

func emptyExportCells() []scheduleExportCell {
	cells := make([]scheduleExportCell, 0, 8)
	for _, pair := range exportPairs() {
		cells = append(cells, scheduleExportCell{PairNumber: pair})
	}
	return cells
}

func exportPairs() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8}
}

func twoWeekExportBounds(ref time.Time) (time.Time, time.Time, time.Time) {
	week1Start := scheduleMonday(ref)
	week2Start := week1Start.AddDate(0, 0, 7)
	return week1Start, week2Start, week2Start.AddDate(0, 0, 5)
}

func formatDateRange(start, end time.Time) string {
	return formatDate(start) + " - " + formatDate(end)
}

func formatDate(d time.Time) string {
	return dateOnlyUTC(d).Format("02.01.2006")
}

func formatDateString(v string) string {
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		return v
	}
	return formatDate(d)
}

func dateOnlyUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func dayScheduleNote(d schedule.DaySchedule) string {
	notes := make([]string, 0, 3)
	if d.GlobalDayConstraint != nil {
		notes = append(notes, d.GlobalDayConstraint.Title)
	}
	if d.StudyDayState != nil && !d.StudyDayState.IsTeaching {
		label := strings.TrimSpace(stringValue(d.StudyDayState.ActivityName))
		if label == "" {
			label = d.StudyDayState.ActivityCode
		}
		if label != "" {
			notes = append(notes, label)
		}
	}
	if d.OverlayText != nil && strings.TrimSpace(*d.OverlayText) != "" {
		notes = append(notes, strings.TrimSpace(*d.OverlayText))
	}
	return strings.Join(notes, "; ")
}

func scheduleCellText(lessons []scheduleExportLesson) string {
	parts := make([]string, 0, len(lessons))
	for _, lesson := range lessons {
		lines := []string{lesson.Subject}
		for _, v := range []string{lesson.Primary, lesson.Secondary, lesson.Location, lesson.Badge, lesson.Comment} {
			if strings.TrimSpace(v) != "" {
				lines = append(lines, strings.TrimSpace(v))
			}
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func hasChangedLesson(lessons []scheduleExportLesson) bool {
	for _, lesson := range lessons {
		if lesson.IsChanged || lesson.IsAdded {
			return true
		}
	}
	return false
}

func displayLocation(locationName string, lessonFormat string) string {
	locationName = strings.TrimSpace(locationName)
	if locationName != "" {
		return locationName
	}
	if strings.EqualFold(strings.TrimSpace(lessonFormat), "online") {
		return "Дистант"
	}
	return ""
}

func subgroupLabel(subgroup *int16) string {
	if subgroup == nil {
		return ""
	}
	return fmt.Sprintf("%d подгруппа", *subgroup)
}

func changedBadge(isChanged bool, isAdded bool) string {
	if isAdded {
		return "добавлено"
	}
	if isChanged {
		return "замена"
	}
	return ""
}

func overrideActionLabel(action schedule.OverrideAction) string {
	switch action {
	case schedule.OverrideAdd:
		return "Добавление"
	case schedule.OverrideReplace:
		return "Замена"
	case schedule.OverrideCancel:
		return "Отмена"
	case schedule.OverrideRestore:
		return "Восстановление"
	default:
		return string(action)
	}
}

func overrideSideText(subject, teacher, location string, lessonFormat *string) string {
	lines := make([]string, 0, 4)
	for _, v := range []string{subject, teacher, location} {
		if strings.TrimSpace(v) != "" {
			lines = append(lines, strings.TrimSpace(v))
		}
	}
	if len(lines) == 0 && lessonFormat != nil && strings.EqualFold(*lessonFormat, "online") {
		lines = append(lines, "Дистант")
	}
	return strings.Join(lines, "\n")
}

func fallback(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func formatPair(pair int16, subgroup *int16) string {
	if subgroup == nil {
		return strconv.Itoa(int(pair))
	}
	return fmt.Sprintf("%d / %d пгр.", pair, *subgroup)
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeASCIIFileName(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "\\", "_")
	v = unsafeFilenameChars.ReplaceAllString(v, "_")
	v = strings.Trim(v, "._-")
	if v == "" {
		return "export"
	}
	return v
}

func contentDisposition(filename string) string {
	ascii := safeASCIIFileName(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(filename))
}
