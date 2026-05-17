package httpapi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestBuildTwoWeekSchedulePDFHTML(t *testing.T) {
	data := sampleTwoWeekExportData()

	html, err := buildTwoWeekSchedulePDFHTML(data)

	require.NoError(t, err)
	require.Contains(t, html, "Расписание группы 22290907/1095")
	require.Contains(t, html, "МДК.02.03 Математическое моделирование")
	require.Contains(t, html, "Числитель")
	require.Contains(t, html, "Знаменатель")
	require.Contains(t, html, "403 П")
}

func TestBuildTwoWeekScheduleXLSX(t *testing.T) {
	data := sampleTwoWeekExportData()

	body, err := buildTwoWeekScheduleXLSX(data)

	require.NoError(t, err)
	require.NotEmpty(t, body)

	f, err := excelize.OpenReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	title, err := f.GetCellValue("2 недели", "A1")
	require.NoError(t, err)
	require.Equal(t, data.Title, title)

	cell, err := f.GetCellValue("2 недели", "B6")
	require.NoError(t, err)
	require.Contains(t, cell, "МДК.02.03 Математическое моделирование")
	require.Contains(t, cell, "Зернова Е.Н.")
}

func TestBuildScheduleOverridesXLSX(t *testing.T) {
	data := &overridesExportData{
		Title:       "Журнал замен расписания",
		Subtitle:    "23.03.2026 - 04.04.2026",
		GeneratedAt: "Сформировано: test",
		RowsCount:   1,
		Rows: []overrideExportRow{
			{
				Date:            "24.03.2026",
				Pair:            "2",
				GroupName:       "22290907/1095",
				ActionLabel:     "Замена",
				SourceText:      "Ин. яз.\nКузнецова Л.И.\n441",
				ReplacementText: "Физ. культура\nСмирнов А.Н.\nск5",
				Reason:          "Больничный",
			},
		},
	}

	body, err := buildScheduleOverridesXLSX(data)

	require.NoError(t, err)
	require.NotEmpty(t, body)

	f, err := excelize.OpenReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	title, err := f.GetCellValue("Замены", "A1")
	require.NoError(t, err)
	require.Equal(t, data.Title, title)

	replacement, err := f.GetCellValue("Замены", "F5")
	require.NoError(t, err)
	require.True(t, strings.Contains(replacement, "Физ. культура"))
}

func sampleTwoWeekExportData() *twoWeekScheduleExportData {
	return &twoWeekScheduleExportData{
		Title:       "Расписание группы 22290907/1095",
		Subtitle:    "23.03.2026 - 04.04.2026",
		GeneratedAt: "Сформировано: test",
		Pairs:       exportPairs(),
		Weeks: []scheduleExportWeek{
			{
				Title:      "Числитель",
				RangeLabel: "23.03.2026 - 28.03.2026",
				Days: []scheduleExportDay{
					{
						DayName:   "ПН",
						Date:      "2026-03-23",
						DateLabel: "23.03.2026",
						Cells: []scheduleExportCell{
							{PairNumber: 1, Lessons: []scheduleExportLesson{
								{
									Subject:   "МДК.02.03 Математическое моделирование",
									Primary:   "Зернова Е.Н.",
									Location:  "403 П",
									IsChanged: true,
									Badge:     "замена",
								},
							}},
							{PairNumber: 2},
							{PairNumber: 3},
							{PairNumber: 4},
							{PairNumber: 5},
							{PairNumber: 6},
							{PairNumber: 7},
							{PairNumber: 8},
						},
					},
				},
			},
			{
				Title:      "Знаменатель",
				RangeLabel: "30.03.2026 - 04.04.2026",
				Days: []scheduleExportDay{
					{
						DayName:   "ПН",
						Date:      "2026-03-30",
						DateLabel: "30.03.2026",
						Cells:     emptyExportCells(),
					},
				},
			},
		},
	}
}
