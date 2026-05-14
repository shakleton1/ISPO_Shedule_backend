package httpapi

import (
	"strings"
	"testing"
)

func TestBuildMermaidER_Basic(t *testing.T) {
	cols := []dbTableColumn{
		{TableName: "teachers", Column: "id", DataType: "bigint", IsNullable: "NO"},
		{TableName: "teachers", Column: "name", DataType: "text", IsNullable: "NO"},
		{TableName: "schedule_lessons", Column: "id", DataType: "bigint", IsNullable: "NO"},
		{TableName: "schedule_lessons", Column: "teacher_id", DataType: "bigint", IsNullable: "YES"},
	}
	fks := []dbFK{
		{TableName: "schedule_lessons", ColumnName: "teacher_id", RefTableName: "teachers", RefColumnName: "id"},
	}

	out := buildMermaidER(cols, fks)

	if len(out) == 0 {
		t.Fatalf("expected mermaid output")
	}
	if want := "erDiagram"; !strings.Contains(out, want) {
		t.Fatalf("expected %q in output", want)
	}
	if want := "teachers"; !strings.Contains(out, want) {
		t.Fatalf("expected table %q in output", want)
	}
	if want := "schedule_lessons"; !strings.Contains(out, want) {
		t.Fatalf("expected table %q in output", want)
	}
	if want := "schedule_lessons }o--|| teachers"; !strings.Contains(out, want) {
		t.Fatalf("expected FK relation in output")
	}
}

func TestSanitizeMermaidIdent(t *testing.T) {
	if got := sanitizeMermaidIdent("a-b c"); got != "abc" {
		t.Fatalf("expected sanitized ident, got %q", got)
	}
	if got := sanitizeMermaidIdent("---"); got != "x" {
		t.Fatalf("expected fallback ident x, got %q", got)
	}
}
