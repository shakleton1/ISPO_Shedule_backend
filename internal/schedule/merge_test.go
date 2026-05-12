package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOverrides_EmptyList(t *testing.T) {
	result := normalizeOverrides([]OverrideView{})

	assert.Empty(t, result)
}

func TestNormalizeOverrides_SingleOverride(t *testing.T) {
	now := time.Now().UTC()
	input := []OverrideView{
		{
			ID:         1,
			PairNumber: 1,
			ActionType: OverrideCancel,
			UpdatedAt:  now,
		},
	}

	result := normalizeOverrides(input)

	require.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].ID)
}

func TestNormalizeOverrides_MultipleSubgroups(t *testing.T) {
	now := time.Now().UTC()
	sg1 := int16(1)
	sg2 := int16(2)

	input := []OverrideView{
		{ID: 1, PairNumber: 1, ActionType: OverrideCancel, Subgroup: nil, UpdatedAt: now},
		{ID: 2, PairNumber: 1, ActionType: OverrideCancel, Subgroup: &sg1, UpdatedAt: now},
		{ID: 3, PairNumber: 1, ActionType: OverrideCancel, Subgroup: &sg2, UpdatedAt: now},
	}

	result := normalizeOverrides(input)

	// Должно остаться 3 override (разные subgroups)
	require.Len(t, result, 3)
}

func TestNormalizeOverrides_DuplicateKey_KeepsBest(t *testing.T) {
	base := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)

	input := []OverrideView{
		{ID: 1, PairNumber: 1, ActionType: OverrideReplace, UpdatedAt: base},
		{ID: 2, PairNumber: 1, ActionType: OverrideReplace, UpdatedAt: later},
	}

	result := normalizeOverrides(input)

	require.Len(t, result, 1)
	assert.Equal(t, int64(2), result[0].ID) // Более новый UpdatedAt
}

func TestOverrideBetterThan_SamePriority_DifferentUpdatedAt(t *testing.T) {
	base := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)

	a := OverrideView{ID: 1, ActionType: OverrideReplace, UpdatedAt: later}
	b := OverrideView{ID: 2, ActionType: OverrideReplace, UpdatedAt: base}

	// a более новый, должен быть "лучше"
	assert.True(t, overrideBetterThan(a, b))
	assert.False(t, overrideBetterThan(b, a))
}

func TestOverrideBetterThan_SameUpdatedAt_LargerIDWins(t *testing.T) {
	sameTime := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)

	a := OverrideView{ID: 10, ActionType: OverrideReplace, UpdatedAt: sameTime}
	b := OverrideView{ID: 20, ActionType: OverrideReplace, UpdatedAt: sameTime}

	// Больший ID должен выиграть
	assert.False(t, overrideBetterThan(a, b))
	assert.True(t, overrideBetterThan(b, a))
}

func TestOverridePriority(t *testing.T) {
	tests := []struct {
		action   OverrideAction
		expected int
	}{
		{OverrideCancel, 0},
		{OverrideReplace, 1},
		{OverrideAdd, 2},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			assert.Equal(t, tt.expected, overridePriority(tt.action))
		})
	}
}

func TestSubgroupMatch_AllCombinations(t *testing.T) {
	sg1 := int16(1)
	sg2 := int16(2)

	tests := []struct {
		name             string
		lessonSubgroup   *int16
		overrideSubgroup *int16
		expected         bool
	}{
		{"both nil", nil, nil, true},
		{"lesson nil, override 1", nil, &sg1, true},
		{"lesson nil, override 2", nil, &sg2, true},
		{"lesson 1, override nil", &sg1, nil, true},
		{"lesson 2, override nil", &sg2, nil, true},
		{"both 1", &sg1, &sg1, true},
		{"both 2", &sg2, &sg2, true},
		{"lesson 1, override 2", &sg1, &sg2, false},
		{"lesson 2, override 1", &sg2, &sg1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := subgroupMatch(tt.lessonSubgroup, tt.overrideSubgroup)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterOutPair_ByPairNumber(t *testing.T) {
	lessons := []Lesson{
		{PairNumber: 1, SubjectName: "Math"},
		{PairNumber: 2, SubjectName: "Physics"},
		{PairNumber: 3, SubjectName: "Chemistry"},
	}

	result := filterOutPair(lessons, 2, nil)

	require.Len(t, result, 2)
	assert.Equal(t, int16(1), result[0].PairNumber)
	assert.Equal(t, int16(3), result[1].PairNumber)
}

func TestFilterOutPair_ByPairAndSubgroup(t *testing.T) {
	sg1 := int16(1)
	sg2 := int16(2)

	lessons := []Lesson{
		{PairNumber: 1, Subgroup: nil, SubjectName: "Math"},
		{PairNumber: 1, Subgroup: &sg1, SubjectName: "Math SG1"},
		{PairNumber: 1, Subgroup: &sg2, SubjectName: "Math SG2"},
		{PairNumber: 2, Subgroup: nil, SubjectName: "Physics"},
	}

	// Удаляем пару 1 с subgroup=1
	// Логика функции: если subgroup!=nil в аргументах:
	// - l.Subgroup=nil -> drop (не проходит if)
	// - l.Subgroup=sg1 -> drop (равно, не проходит if)
	// - l.Subgroup=sg2 -> keep (не равно, проходит if)
	result := filterOutPair(lessons, 1, &sg1)

	require.Len(t, result, 2)
	// Остаются: пара 1 с sg2 и пара 2

	assert.Equal(t, int16(1), result[0].PairNumber)
	assert.Equal(t, sg2, *result[0].Subgroup)
	assert.Equal(t, "Math SG2", result[0].SubjectName)

	assert.Equal(t, int16(2), result[1].PairNumber)
	assert.Nil(t, result[1].Subgroup)
	assert.Equal(t, "Physics", result[1].SubjectName)
}

func TestFilterOutPair_NoMatch(t *testing.T) {
	lessons := []Lesson{
		{PairNumber: 1, SubjectName: "Math"},
		{PairNumber: 2, SubjectName: "Physics"},
	}

	result := filterOutPair(lessons, 3, nil)

	// Ничего не должно удалиться
	require.Len(t, result, 2)
	assert.Equal(t, int16(1), result[0].PairNumber)
	assert.Equal(t, int16(2), result[1].PairNumber)
}

func TestApplyOverrideReplace_FullReplace(t *testing.T) {
	newSubID := 10
	newLocID := 20
	newTeacher := "New Teacher"
	comment := "Test comment"
	flow := "flow-1"
	sg1 := int16(1)

	lesson := &Lesson{
		PairNumber:   1,
		SubjectID:    nil,
		SubjectName:  "Old Subject",
		LocationID:   nil,
		LocationName: "Old Location",
		TeacherName:  "Old Teacher",
		Subgroup:     nil,
		IsChanged:    false,
	}

	override := OverrideView{
		PairNumber:      1,
		ActionType:      OverrideReplace,
		NewSubjectID:    &newSubID,
		NewSubjectName:  "New Subject",
		NewLocationID:   &newLocID,
		NewLocationName: "New Location",
		NewTeacherName:  &newTeacher,
		Comment:         &comment,
		Subgroup:        &sg1,
		FlowKey:         &flow,
	}

	applyOverrideReplace(lesson, override)

	assert.Equal(t, newSubID, *lesson.SubjectID)
	assert.Equal(t, "New Subject", lesson.SubjectName)
	assert.Equal(t, newLocID, *lesson.LocationID)
	assert.Equal(t, "New Location", lesson.LocationName)
	assert.Equal(t, newTeacher, lesson.TeacherName)
	assert.Equal(t, sg1, *lesson.Subgroup)
	assert.Equal(t, flow, *lesson.FlowKey)
	assert.True(t, lesson.IsChanged)
	assert.Equal(t, &comment, lesson.Comment)
}

func TestApplyOverrideReplace_PartialReplace(t *testing.T) {
	newTeacher := "New Teacher"

	lesson := &Lesson{
		PairNumber:   1,
		SubjectID:    intPtr(10),
		SubjectName:  "Old Subject",
		LocationID:   intPtr(20),
		LocationName: "Old Location",
		TeacherName:  "",
		IsChanged:    false,
	}

	override := OverrideView{
		PairNumber:     1,
		ActionType:     OverrideReplace,
		NewTeacherName: &newTeacher,
		// NewSubjectID и NewLocationID nil
	}

	applyOverrideReplace(lesson, override)

	// Subject и Location должны остаться unchanged
	assert.Equal(t, 10, *lesson.SubjectID)
	assert.Equal(t, "Old Subject", lesson.SubjectName)
	assert.Equal(t, 20, *lesson.LocationID)
	assert.Equal(t, "Old Location", lesson.LocationName)
	// Teacher должен обновиться
	assert.Equal(t, newTeacher, lesson.TeacherName)
	assert.True(t, lesson.IsChanged)
}

func TestBuildLessonFromOverride_FullData(t *testing.T) {
	newSubID := 10
	newLocID := 20
	newTeacher := "Teacher"
	comment := "Comment"
	flow := "flow-2"
	sg1 := int16(1)

	override := OverrideView{
		PairNumber:      3,
		ActionType:      OverrideAdd,
		NewSubjectID:    &newSubID,
		NewSubjectName:  "New Subject",
		NewLocationID:   &newLocID,
		NewLocationName: "New Location",
		NewTeacherName:  &newTeacher,
		Comment:         &comment,
		Subgroup:        &sg1,
		FlowKey:         &flow,
	}

	lesson := buildLessonFromOverride(override, true)

	assert.Equal(t, int16(3), lesson.PairNumber)
	assert.Equal(t, newSubID, *lesson.SubjectID)
	assert.Equal(t, "New Subject", lesson.SubjectName)
	assert.Equal(t, newLocID, *lesson.LocationID)
	assert.Equal(t, "New Location", lesson.LocationName)
	assert.Equal(t, newTeacher, lesson.TeacherName)
	assert.Equal(t, sg1, *lesson.Subgroup)
	assert.Equal(t, flow, *lesson.FlowKey)
	assert.True(t, lesson.IsAdded)
	assert.False(t, lesson.IsChanged) // ADD не меняет, а добавляет
	assert.Equal(t, &comment, lesson.Comment)
}

func TestBuildLessonFromOverride_MinimalData(t *testing.T) {
	override := OverrideView{
		PairNumber: 5,
		ActionType: OverrideAdd,
		// Все остальные поля nil/пустые
	}

	lesson := buildLessonFromOverride(override, true)

	assert.Equal(t, int16(5), lesson.PairNumber)
	assert.Nil(t, lesson.SubjectID)
	assert.Empty(t, lesson.SubjectName)
	assert.Nil(t, lesson.LocationID)
	assert.Empty(t, lesson.LocationName)
	assert.Empty(t, lesson.TeacherName)
	assert.Nil(t, lesson.Subgroup)
	assert.True(t, lesson.IsAdded)
	assert.Nil(t, lesson.Comment)
}

func TestBuildLessonFromOverride_ReplaceAction(t *testing.T) {
	override := OverrideView{
		PairNumber: 2,
		ActionType: OverrideReplace,
	}

	lesson := buildLessonFromOverride(override, false)

	assert.Equal(t, int16(2), lesson.PairNumber)
	assert.False(t, lesson.IsAdded)
	assert.True(t, lesson.IsChanged) // REPLACE должен быть IsChanged=true
}

// Helper function
func intPtr(i int) *int {
	return &i
}
