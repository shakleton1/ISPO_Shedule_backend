package schedule

import "strings"

type WeekParity string

const (
	WeekParityNumerator   WeekParity = "numerator"
	WeekParityDenominator WeekParity = "denominator"
	WeekParityBoth        WeekParity = "both"
)

type OverrideAction string

const (
	OverrideAdd     OverrideAction = "add"
	OverrideReplace OverrideAction = "replace"
	OverrideCancel  OverrideAction = "cancel"
	OverrideRestore OverrideAction = "restore"
)

func normalizeLessonFormat(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "online":
		return "online"
	case "hybrid":
		return "hybrid"
	default:
		return "offline"
	}
}
