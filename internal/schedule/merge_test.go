package schedule

import (
	"testing"
	"time"
)

func TestMergeLessons_CancelReplaceAdd(t *testing.T) {
	sg1 := int16(1)

	tpls := []TemplateView{
		{
			PairNumber:   1,
			SubjectID:    10,
			SubjectName:  "Math",
			LocationID:   100,
			LocationName: "101",
			TeacherName:  "A",
			Subgroup:     nil,
		},
		{
			PairNumber:   2,
			SubjectID:    20,
			SubjectName:  "Physics",
			LocationID:   200,
			LocationName: "202",
			TeacherName:  "B",
			Subgroup:     &sg1,
		},
	}

	newSub := 21
	newLoc := 201
	newTeacher := "B2"
	comment := "changed"

	ovrs := []OverrideView{
		{PairNumber: 1, ActionType: OverrideCancel, UpdatedAt: time.Now().UTC()},
		{
			PairNumber:      2,
			ActionType:      OverrideReplace,
			NewSubjectID:    &newSub,
			NewSubjectName:  "Physics2",
			NewLocationID:   &newLoc,
			NewLocationName: "203",
			NewTeacherName:  &newTeacher,
			Comment:         &comment,
			Subgroup:        &sg1,
			UpdatedAt:       time.Now().UTC(),
		},
		{
			PairNumber:      3,
			ActionType:      OverrideAdd,
			NewSubjectID:    &newSub,
			NewSubjectName:  "Added",
			NewLocationID:   &newLoc,
			NewLocationName: "X",
			UpdatedAt:       time.Now().UTC(),
		},
	}

	lessons, err := mergeLessons(tpls, ovrs)
	if err != nil {
		t.Fatalf("mergeLessons err: %v", err)
	}

	// pair 1 should be removed
	for _, l := range lessons {
		if l.PairNumber == 1 {
			t.Fatalf("expected pair 1 canceled, got %+v", l)
		}
	}

	// pair 2 should be replaced
	var got2 *Lesson
	for i := range lessons {
		if lessons[i].PairNumber == 2 {
			got2 = &lessons[i]
			break
		}
	}
	if got2 == nil {
		t.Fatalf("expected pair 2 present")
	}
	if got2.SubjectID == nil || *got2.SubjectID != newSub {
		t.Fatalf("expected subject_id=%d, got %+v", newSub, got2)
	}
	if got2.LocationID == nil || *got2.LocationID != newLoc {
		t.Fatalf("expected location_id=%d, got %+v", newLoc, got2)
	}
	if got2.TeacherName != newTeacher {
		t.Fatalf("expected teacher_name=%q, got %+v", newTeacher, got2)
	}
	if !got2.IsChanged {
		t.Fatalf("expected IsChanged=true, got %+v", got2)
	}
	if got2.IsAdded {
		t.Fatalf("expected IsAdded=false on replace, got %+v", got2)
	}
	if got2.Comment == nil || *got2.Comment != comment {
		t.Fatalf("expected comment=%q, got %+v", comment, got2)
	}

	// pair 3 should be added
	var got3 *Lesson
	for i := range lessons {
		if lessons[i].PairNumber == 3 {
			got3 = &lessons[i]
			break
		}
	}
	if got3 == nil {
		t.Fatalf("expected pair 3 present")
	}
	if !got3.IsAdded {
		t.Fatalf("expected IsAdded=true, got %+v", got3)
	}
}

func TestMergeLessons_Normalization_PriorityBeatsRecency(t *testing.T) {
	// Same key (pair/subgroup=nil): CANCEL should beat ADD even if ADD is newer.
	newer := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)

	newSub := 1
	newLoc := 1

	ovrs := []OverrideView{
		{ID: 1, PairNumber: 1, ActionType: OverrideAdd, NewSubjectID: &newSub, NewSubjectName: "X", NewLocationID: &newLoc, NewLocationName: "Y", UpdatedAt: newer},
		{ID: 2, PairNumber: 1, ActionType: OverrideCancel, UpdatedAt: older},
	}

	lessons, err := mergeLessons(nil, ovrs)
	if err != nil {
		t.Fatalf("mergeLessons err: %v", err)
	}
	if len(lessons) != 0 {
		t.Fatalf("expected CANCEL to win and remove lesson, got %+v", lessons)
	}
}

func TestMergeLessons_Normalization_UpdatedAtThenID(t *testing.T) {
	sg1 := int16(1)
	base := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)

	newTeacher1 := "T1"
	newTeacher2 := "T2"

	// Two REPLACE for same key: later UpdatedAt should win.
	ovrs := []OverrideView{
		{ID: 1, PairNumber: 2, ActionType: OverrideReplace, NewTeacherName: &newTeacher1, Subgroup: &sg1, UpdatedAt: base},
		{ID: 2, PairNumber: 2, ActionType: OverrideReplace, NewTeacherName: &newTeacher2, Subgroup: &sg1, UpdatedAt: later},
	}
	lessons, err := mergeLessons(nil, ovrs)
	if err != nil {
		t.Fatalf("mergeLessons err: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %+v", lessons)
	}
	if lessons[0].TeacherName != newTeacher2 {
		t.Fatalf("expected newer override to win: %q, got %q", newTeacher2, lessons[0].TeacherName)
	}

	// Same UpdatedAt: larger ID should win.
	newTeacher3 := "T3"
	newTeacher4 := "T4"
	ovrs2 := []OverrideView{
		{ID: 10, PairNumber: 2, ActionType: OverrideReplace, NewTeacherName: &newTeacher3, Subgroup: &sg1, UpdatedAt: base},
		{ID: 11, PairNumber: 2, ActionType: OverrideReplace, NewTeacherName: &newTeacher4, Subgroup: &sg1, UpdatedAt: base},
	}
	lessons2, err := mergeLessons(nil, ovrs2)
	if err != nil {
		t.Fatalf("mergeLessons err: %v", err)
	}
	if len(lessons2) != 1 {
		t.Fatalf("expected 1 lesson, got %+v", lessons2)
	}
	if lessons2[0].TeacherName != newTeacher4 {
		t.Fatalf("expected larger ID override to win: %q, got %q", newTeacher4, lessons2[0].TeacherName)
	}
}
