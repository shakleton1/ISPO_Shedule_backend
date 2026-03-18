package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectCSVDelimiter_Comma(t *testing.T) {
	data := "day_of_week,week_parity,pair_number\n1,numerator,2"
	delimiter := detectCSVDelimiter([]byte(data))

	assert.Equal(t, ',', delimiter)
}

func TestDetectCSVDelimiter_Semicolon(t *testing.T) {
	data := "day_of_week;week_parity;pair_number\n1;numerator;2"
	delimiter := detectCSVDelimiter([]byte(data))

	assert.Equal(t, ';', delimiter)
}

func TestDetectCSVDelimiter_Empty(t *testing.T) {
	data := ""
	delimiter := detectCSVDelimiter([]byte(data))

	// Default to comma
	assert.Equal(t, ',', delimiter)
}

func TestMapHeaders_AllVariants(t *testing.T) {
	tests := []struct {
		headers  []string
		expected map[string]int
	}{
		{
			headers: []string{"day_of_week", "week_parity", "pair_number", "subject", "location", "teacher_name", "subgroup"},
			expected: map[string]int{
				"day_of_week":  0,
				"week_parity":  1,
				"pair_number":  2,
				"subject":      3,
				"location":     4,
				"teacher_name": 5,
				"subgroup":     6,
			},
		},
		{
			headers: []string{"день", "четность", "пара", "предмет", "аудитория", "преподаватель", "подгруппа"},
			expected: map[string]int{
				"day_of_week":  0,
				"week_parity":  1,
				"pair_number":  2,
				"subject":      3,
				"location":     4,
				"teacher_name": 5,
				"subgroup":     6,
			},
		},
		{
			headers: []string{"Day", "Week", "Pair", "Subject_Name", "Room", "Teacher"},
			expected: map[string]int{
				"day_of_week":  0,
				"week_parity":  1,
				"pair_number":  2,
				"subject":      3,
				"location":     4,
				"teacher_name": 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.headers, "_"), func(t *testing.T) {
			result := mapHeaders(tt.headers)

			for key, expectedIdx := range tt.expected {
				idx, ok := result[key]
				assert.True(t, ok, "Key %s should exist", key)
				assert.Equal(t, expectedIdx, idx, "Key %s should have index %d", key, expectedIdx)
			}
		})
	}
}

func TestNormalizeHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Day_Of_Week", "day_of_week"},
		{"day-of-week", "day_of_week"},
		{"DAY OF WEEK", "day_of_week"},
		{"  Week_Parity  ", "week_parity"},
		{"PairNumber", "pairnumber"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHeader(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTemplateRecord_InvalidDayOfWeek(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "day_of_week":
			return "7" // Invalid: must be 0-5 or 1-6
		case "week_parity":
			return "numerator"
		case "pair_number":
			return "1"
		case "subject":
			return "Math"
		case "location":
			return "101"
		case "teacher_name":
			return "Teacher"
		default:
			return ""
		}
	}

	_, err := parseTemplateRecord(get)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "day_of_week")
}

func TestParseTemplateRecord_InvalidWeekParity(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "day_of_week":
			return "0"
		case "week_parity":
			return "invalid"
		case "pair_number":
			return "1"
		case "subject":
			return "Math"
		case "location":
			return "101"
		case "teacher_name":
			return "Teacher"
		default:
			return ""
		}
	}

	_, err := parseTemplateRecord(get)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "week_parity")
}

func TestParseTemplateRecord_InvalidPairNumber(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "day_of_week":
			return "0"
		case "week_parity":
			return "numerator"
		case "pair_number":
			return "9" // Invalid: must be 1-8
		case "subject":
			return "Math"
		case "location":
			return "101"
		case "teacher_name":
			return "Teacher"
		default:
			return ""
		}
	}

	_, err := parseTemplateRecord(get)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pair_number")
}

func TestParseTemplateRecord_InvalidSubgroup(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "day_of_week":
			return "0"
		case "week_parity":
			return "numerator"
		case "pair_number":
			return "1"
		case "subject":
			return "Math"
		case "location":
			return "101"
		case "teacher_name":
			return "Teacher"
		case "subgroup":
			return "3" // Invalid: must be 1 or 2
		default:
			return ""
		}
	}

	_, err := parseTemplateRecord(get)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subgroup")
}

func TestParseTemplateRecord_MissingRequired(t *testing.T) {
	get := func(key string) string {
		switch key {
		case "day_of_week":
			return "0"
		case "week_parity":
			return "numerator"
		case "pair_number":
			return "1"
		case "subject":
			return "" // Missing
		case "location":
			return "101"
		case "teacher_name":
			return "Teacher"
		default:
			return ""
		}
	}

	_, err := parseTemplateRecord(get)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}
