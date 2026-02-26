package httpapi

import (
	"context"
	"time"

	"ispo-schedule/internal/pdf"
	"ispo-schedule/internal/schedule"
)

type schedulePDFEngine interface {
	RenderHTMLToPDF(ctx context.Context, html string) ([]byte, error)
}

type pdfEngineAdapter struct{ e *pdf.Engine }

func (a pdfEngineAdapter) RenderHTMLToPDF(ctx context.Context, html string) ([]byte, error) {
	return a.e.RenderHTMLToPDF(ctx, pdf.RenderInput{HTML: html})
}

func scheduleMonday(t time.Time) time.Time {
	wd := int(t.Weekday())
	offset := (wd + 6) % 7
	y, m, d := t.Date()
	base := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, -offset)
}

// PDF view-models

type pdfDay struct {
	DayName     string
	Date        string
	OverlayText *string
	Lessons     []pdfCell
}

type pdfCell struct {
	PairNumber int
	IsSplit    bool
	Sub1       *pdfLesson
	Sub2       *pdfLesson
	Single     *pdfLesson
}

type pdfLesson struct {
	Subject   string
	Teacher   string
	Location  string
	IsChanged bool
}

type pdfTemplateData struct {
	GroupName  string
	DaysWeek1  []pdfDay
	DaysWeek2  []pdfDay
}

var pdfTpl = pdf.MustParseTemplate("schedule", scheduleHTMLTemplate)

func buildSchedulePDFHTML(groupName string, daysWeek1, daysWeek2 []schedule.DaySchedule) (string, error) {
	data := pdfTemplateData{
		GroupName: groupName,
		DaysWeek1: toPDFDays(daysWeek1),
		DaysWeek2: toPDFDays(daysWeek2),
	}
	return pdf.RenderTemplate(pdfTpl, data)
}

func toPDFDays(days []schedule.DaySchedule) []pdfDay {
	out := make([]pdfDay, 0, len(days))
	for _, d := range days {
		pd := pdfDay{
			DayName:     ruDayName(d.DayOfWeek),
			Date:        d.Date,
			OverlayText: d.OverlayText,
			Lessons:     buildCells(d.Lessons),
		}
		out = append(out, pd)
	}
	return out
}

func buildCells(lessons []schedule.Lesson) []pdfCell {
	// We want fixed 8 pairs.
	byPair := map[int][]schedule.Lesson{}
	for _, l := range lessons {
		byPair[int(l.PairNumber)] = append(byPair[int(l.PairNumber)], l)
	}

	var cells []pdfCell
	for p := 1; p <= 8; p++ {
		ls := byPair[p]
		cell := pdfCell{PairNumber: p}
		// Detect split by subgroup 1+2
		var l1, l2 *schedule.Lesson
		for i := range ls {
			if ls[i].Subgroup == nil {
				continue
			}
			sg := *ls[i].Subgroup
			if sg == 1 {
				copy := ls[i]
				l1 = &copy
			}
			if sg == 2 {
				copy := ls[i]
				l2 = &copy
			}
		}
		if l1 != nil || l2 != nil {
			cell.IsSplit = true
			if l1 != nil {
				cell.Sub1 = &pdfLesson{Subject: l1.SubjectName, Teacher: l1.TeacherName, Location: l1.LocationName, IsChanged: l1.IsChanged}
			}
			if l2 != nil {
				cell.Sub2 = &pdfLesson{Subject: l2.SubjectName, Teacher: l2.TeacherName, Location: l2.LocationName, IsChanged: l2.IsChanged}
			}
			cells = append(cells, cell)
			continue
		}

		// single: pick first lesson with nil subgroup, else first
		var pick *schedule.Lesson
		for i := range ls {
			if ls[i].Subgroup == nil {
				copy := ls[i]
				pick = &copy
				break
			}
		}
		if pick == nil && len(ls) > 0 {
			copy := ls[0]
			pick = &copy
		}
		if pick != nil {
			cell.Single = &pdfLesson{Subject: pick.SubjectName, Teacher: pick.TeacherName, Location: pick.LocationName, IsChanged: pick.IsChanged}
		}
		cells = append(cells, cell)
	}
	return cells
}

func ruDayName(dayOfWeek int16) string {
	switch dayOfWeek {
	case 0:
		return "ПН"
	case 1:
		return "ВТ"
	case 2:
		return "СР"
	case 3:
		return "ЧТ"
	case 4:
		return "ПТ"
	case 5:
		return "СБ"
	default:
		return "?"
	}
}
