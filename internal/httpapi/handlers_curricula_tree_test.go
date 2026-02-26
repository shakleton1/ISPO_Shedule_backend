package httpapi

import (
	"testing"

	"ispo-schedule/internal/schedule"
)

func strPtr(s string) *string { return &s }
func int64Ptr(v int64) *int64 { return &v }

func TestBuildCurriculumItemTree(t *testing.T) {
	items := []schedule.CurriculumItem{
		{ID: 1, CurriculumID: 10, ParentID: nil, IndexCode: strPtr("ПЦ.01"), ItemType: "OTHER", Name: "ПЦ"},
		{ID: 2, CurriculumID: 10, ParentID: int64Ptr(1), IndexCode: strPtr("ПМ.01"), ItemType: "OTHER", Name: "ПМ.01"},
		{ID: 3, CurriculumID: 10, ParentID: int64Ptr(2), IndexCode: strPtr("МДК.01.01"), ItemType: "DISCIPLINE", Name: "МДК"},
		{ID: 4, CurriculumID: 10, ParentID: int64Ptr(1), IndexCode: nil, ItemType: "OTHER", Name: "Без индекса"},
		{ID: 5, CurriculumID: 10, ParentID: int64Ptr(99), IndexCode: nil, ItemType: "OTHER", Name: "Сирота"},
	}

	tree := buildCurriculumItemTree(items)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
	if tree[0].ID != 1 {
		t.Fatalf("expected root[0].id=1, got %d", tree[0].ID)
	}
	if tree[1].ID != 5 {
		t.Fatalf("expected root[1].id=5, got %d", tree[1].ID)
	}

	if len(tree[0].Children) != 2 {
		t.Fatalf("expected root[0] to have 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].ID != 2 {
		t.Fatalf("expected first child id=2, got %d", tree[0].Children[0].ID)
	}
	if tree[0].Children[1].ID != 4 {
		t.Fatalf("expected second child id=4, got %d", tree[0].Children[1].ID)
	}

	if len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("expected node 2 to have 1 child, got %d", len(tree[0].Children[0].Children))
	}
	if tree[0].Children[0].Children[0].ID != 3 {
		t.Fatalf("expected node 2 child id=3, got %d", tree[0].Children[0].Children[0].ID)
	}
}
