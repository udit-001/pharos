package db

import "testing"

func TestTotals(t *testing.T) {
	ws := []Workspace{
		{Name: "a", LessonCount: 2, RecordCount: 1, RefCount: 3},
		{Name: "b", LessonCount: 0, RecordCount: 4, RefCount: 1},
	}
	got := Totals(ws)
	want := Stats{Workspaces: 2, Lessons: 2, Records: 5, Refs: 4}
	if got != want {
		t.Errorf("Totals = %+v, want %+v", got, want)
	}
}

func TestContinueItemLesson(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "alpha")
	if _, err := ws.AddLesson(Lesson{Title: "Intro", Filename: "0001-intro.html"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.SetLastViewed("lesson", 1); err != nil {
		t.Fatal(err)
	}

	ci, err := store.ContinueItem()
	if err != nil {
		t.Fatal(err)
	}
	if ci == nil {
		t.Fatal("expected continue item, got nil")
	}
	if want := "/workspace/alpha/lesson/1"; ci.URL != want {
		t.Errorf("URL = %q, want %q", ci.URL, want)
	}
	if want := "alpha — Lesson: Intro"; ci.Label != want {
		t.Errorf("Label = %q, want %q", ci.Label, want)
	}
}

// TestContinueItemRef covers the LastRefSeq branch that was previously
// untested (it lived inline in the HTTP handler). This is the branch the
// report flagged as "exactly the one that breaks silently."
func TestContinueItemRef(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "alpha")
	if _, err := ws.AddRef(Reference{Title: "SQL Joins", Slug: "sql-joins", Filename: "sql-joins.html"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.SetLastViewed("ref", 0); err != nil {
		t.Fatal(err)
	}

	ci, err := store.ContinueItem()
	if err != nil {
		t.Fatal(err)
	}
	if ci == nil {
		t.Fatal("expected continue item, got nil")
	}
	if want := "/workspace/alpha/ref/sql-joins"; ci.URL != want {
		t.Errorf("URL = %q, want %q", ci.URL, want)
	}
	if want := "alpha — Reference: SQL Joins"; ci.Label != want {
		t.Errorf("Label = %q, want %q", ci.Label, want)
	}
}

func TestContinueItemNone(t *testing.T) {
	store := newTestStore(t)
	seedWorkspace(t, store, "alpha")

	ci, _ := store.ContinueItem()
	if ci != nil {
		t.Errorf("expected nil for workspace with no activity, got %+v", ci)
	}
}
