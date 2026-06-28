package db

import (
	"strings"
	"testing"
)

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

// TestContinueItemRef covers the LastRefSeq branch. last_ref_seq stores the
// viewed reference's row ID (refs are slug-based — migration 00006 dropped
// sequence_number). This test verifies the continue card shows the viewed
// ref, not just the first one.
func TestContinueItemRef(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "alpha")
	if _, err := ws.AddRef(Reference{Title: "SQL Joins", Slug: "sql-joins", Filename: "sql-joins.html"}); err != nil {
		t.Fatal(err)
	}
	ref2, err := ws.AddRef(Reference{Title: "Index Tuning", Slug: "index-tuning", Filename: "index-tuning.html"})
	if err != nil {
		t.Fatal(err)
	}
	// View the SECOND ref — the continue card must show it, not refs[0]
	if err := ws.SetLastViewed("ref", int(ref2.ID)); err != nil {
		t.Fatal(err)
	}

	ci, err := store.ContinueItem()
	if err != nil {
		t.Fatal(err)
	}
	if ci == nil {
		t.Fatal("expected continue item, got nil")
	}
	if want := "/workspace/alpha/ref/index-tuning"; ci.URL != want {
		t.Errorf("URL = %q, want %q (the viewed ref, not refs[0])", ci.URL, want)
	}
	if want := "alpha — Reference: Index Tuning"; ci.Label != want {
		t.Errorf("Label = %q, want %q", ci.Label, want)
	}
}

func TestContinueItemPicksMostRecentWorkspace(t *testing.T) {
	store := newTestStore(t)
	goWs := seedWorkspace(t, store, "go")
	if _, err := goWs.AddLesson(Lesson{Title: "Go Program", Filename: "0001.html"}); err != nil {
		t.Fatal(err)
	}
	if err := goWs.SetLastViewed("lesson", 1); err != nil {
		t.Fatal(err)
	}

	autismWs := seedWorkspace(t, store, "autism")
	if _, err := autismWs.AddLesson(Lesson{Title: "Autism Lesson", Filename: "0001.html"}); err != nil {
		t.Fatal(err)
	}
	if err := autismWs.SetLastViewed("lesson", 1); err != nil {
		t.Fatal(err)
	}

	ci, err := store.ContinueItem()
	if err != nil {
		t.Fatal(err)
	}
	if ci == nil || !strings.Contains(ci.Label, "autism") {
		t.Fatalf("continue card should show autism (most recent), got %+v", ci)
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
