package schedule

type WeekParity string

const (
	WeekParityNumerator   WeekParity = "numerator"
	WeekParityDenominator WeekParity = "denominator"
	WeekParityBoth        WeekParity = "both"
)

type OverrideAction string

const (
	OverrideCancel  OverrideAction = "CANCEL"
	OverrideReplace OverrideAction = "REPLACE"
	OverrideAdd     OverrideAction = "ADD"
)
