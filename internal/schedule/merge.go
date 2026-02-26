package schedule

import "fmt"

func mergeLessons(tpls []TemplateView, ovrs []OverrideView) ([]Lesson, error) {
	// Start with template lessons
	lessons := make([]Lesson, 0, len(tpls))
	for _, t := range tpls {
		l := Lesson{
			PairNumber:   t.PairNumber,
			SubjectID:    &t.SubjectID,
			SubjectName:  t.SubjectName,
			LocationID:   &t.LocationID,
			LocationName: t.LocationName,
			TeacherName:  t.TeacherName,
			Subgroup:     t.Subgroup,
		}
		lessons = append(lessons, l)
	}

	// Apply overrides in priority order: CANCEL -> REPLACE -> ADD
	for _, a := range []OverrideAction{OverrideCancel, OverrideReplace, OverrideAdd} {
		for _, o := range ovrs {
			if o.ActionType != a {
				continue
			}
			switch o.ActionType {
			case OverrideCancel:
				lessons = filterOutPair(lessons, o.PairNumber, o.Subgroup)
			case OverrideReplace:
				var replaced bool
				for i := range lessons {
					if lessons[i].PairNumber != o.PairNumber {
						continue
					}
					if !subgroupMatch(lessons[i].Subgroup, o.Subgroup) {
						continue
					}
					applyOverrideReplace(&lessons[i], o)
					replaced = true
				}
				if !replaced {
					// If nothing to replace, treat as ADD (creates new lesson)
					lessons = append(lessons, buildLessonFromOverride(o, false))
				}
			case OverrideAdd:
				lessons = append(lessons, buildLessonFromOverride(o, true))
			default:
				return nil, fmt.Errorf("unknown action_type: %s", o.ActionType)
			}
		}
	}

	return lessons, nil
}

func subgroupMatch(lessonSubgroup, overrideSubgroup *int16) bool {
	if overrideSubgroup == nil {
		return true
	}
	if lessonSubgroup == nil {
		return true
	}
	return *lessonSubgroup == *overrideSubgroup
}

func filterOutPair(in []Lesson, pair int16, subgroup *int16) []Lesson {
	out := in[:0]
	for _, l := range in {
		if l.PairNumber != pair {
			out = append(out, l)
			continue
		}
		if subgroup != nil && l.Subgroup != nil && *l.Subgroup != *subgroup {
			out = append(out, l)
			continue
		}
		// drop
	}
	return out
}

func applyOverrideReplace(l *Lesson, o OverrideView) {
	if o.NewSubjectID != nil {
		l.SubjectID = o.NewSubjectID
		l.SubjectName = o.NewSubjectName
	}
	if o.NewLocationID != nil {
		l.LocationID = o.NewLocationID
		l.LocationName = o.NewLocationName
	}
	if o.NewTeacherName != nil {
		l.TeacherName = *o.NewTeacherName
	}
	l.IsChanged = true
	l.Comment = o.Comment
	if o.Subgroup != nil {
		sg := *o.Subgroup
		l.Subgroup = &sg
	}
}

func buildLessonFromOverride(o OverrideView, isAdded bool) Lesson {
	lesson := Lesson{
		PairNumber: o.PairNumber,
		Subgroup:   o.Subgroup,
		IsAdded:    isAdded,
		IsChanged:  o.ActionType == OverrideReplace,
		Comment:    o.Comment,
	}
	if o.NewSubjectID != nil {
		lesson.SubjectID = o.NewSubjectID
		lesson.SubjectName = o.NewSubjectName
	}
	if o.NewLocationID != nil {
		lesson.LocationID = o.NewLocationID
		lesson.LocationName = o.NewLocationName
	}
	if o.NewTeacherName != nil {
		lesson.TeacherName = *o.NewTeacherName
	}
	return lesson
}
