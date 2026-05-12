package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationNeedsAutofill(t *testing.T) {
	virtualID := 10
	classroomID := 11
	meta := map[int]LocationMeta{
		virtualID:   {ID: virtualID, LocationKind: "virtual"},
		classroomID: {ID: classroomID, LocationKind: "classroom"},
	}

	assert.True(t, locationNeedsAutofill(Lesson{}, meta, true))
	assert.True(t, locationNeedsAutofill(Lesson{LocationID: &virtualID}, meta, true))
	assert.False(t, locationNeedsAutofill(Lesson{LocationID: &virtualID}, meta, false))
	assert.False(t, locationNeedsAutofill(Lesson{LocationID: &classroomID}, meta, true))
}

func TestChooseFreeLocation(t *testing.T) {
	candidates := []Location{{ID: 1, Name: "101"}, {ID: 2, Name: "102"}}
	occupied := map[locationAutofillSlot]bool{
		{Date: "2026-09-07", PairNumber: 1, LocationID: 1}: true,
	}

	got := chooseFreeLocation("2026-09-07", 1, candidates, occupied)
	if assert.NotNil(t, got) {
		assert.Equal(t, 2, got.ID)
	}

	assert.Nil(t, chooseFreeLocation("2026-09-07", 1, candidates[:1], occupied))
}
