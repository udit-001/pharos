package render

import (
	"strings"
	"testing"
)

// TestDashboardRendersStatsAndWorkspaces proves the render module's output is
// a pure function of its view model — the seam created in LEARN-10.
func TestDashboardRendersStatsAndWorkspaces(t *testing.T) {
	d := DashboardData{
		Stats:      Stats{Workspaces: 2, Lessons: 5, Records: 3, Refs: 1},
		Workspaces: []WorkspaceCard{
			{Name: "sql-basics", LessonCount: 3, RecordCount: 1, RefCount: 1, LastStudied: "2026-06-21"},
			{Name: "golang", LessonCount: 2, RecordCount: 2, RefCount: 0, LastStudied: "2026-06-20"},
		},
	}
	out := Dashboard(d)

	for _, want := range []string{
		"Dashboard",
		">2<", // workspaces count
		">5<", // lessons count
		">3<", // records count
		">1<", // refs count
		"sql-basics",
		"golang",
		"3 lessons · 1 records · 1 refs",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestDashboardEmptyState(t *testing.T) {
	out := Dashboard(DashboardData{})
	if !strings.Contains(out, "No workspaces yet") {
		t.Errorf("expected empty state, got:\n%s", out)
	}
	if !strings.Contains(out, `learn init "topic"`) {
		t.Errorf("expected init hint, got:\n%s", out)
	}
}

func TestDashboardContinueCard(t *testing.T) {
	d := DashboardData{
		Continue: &ContinueItem{URL: "/workspace/sql-basics/lesson/2", Label: "sql-basics — Lesson: JOINs"},
	}
	out := Dashboard(d)
	if !strings.Contains(out, "Continue where you left off") {
		t.Errorf("expected continue label, got:\n%s", out)
	}
	if !strings.Contains(out, "/workspace/sql-basics/lesson/2") {
		t.Errorf("expected continue URL, got:\n%s", out)
	}
}

func TestRecordRendersStatusAndBody(t *testing.T) {
	out := Record(RecordData{Title: "x", Status: "superseded", BodyHTML: "<p>hello</p>"})
	if !strings.Contains(out, "superseded") {
		t.Errorf("expected superseded tag, got:\n%s", out)
	}
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("expected body HTML, got:\n%s", out)
	}
}

func TestRecordActiveStatusTag(t *testing.T) {
	out := Record(RecordData{Status: "active"})
	if !strings.Contains(out, "active") {
		t.Errorf("expected active tag, got:\n%s", out)
	}
	if strings.Contains(out, "superseded") {
		t.Errorf("did not expect superseded tag, got:\n%s", out)
	}
}

func TestPageWrapsContentInFrame(t *testing.T) {
	f := Frame{Title: "My Page"}
	out := Page(f, "<p>BODY</p>")
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("expected doctype, got:\n%s", out)
	}
	if !strings.Contains(out, "<title>My Page — Learn</title>") {
		t.Errorf("expected title tag, got:\n%s", out)
	}
	if !strings.Contains(out, "<p>BODY</p>") {
		t.Errorf("expected body content preserved, got:\n%s", out)
	}
}

func TestPageSidebarEmptyState(t *testing.T) {
	f := Frame{Title: "Dashboard", ActiveWS: ""}
	out := Page(f, "x")
	if !strings.Contains(out, "Select a workspace") {
		t.Errorf("expected sidebar empty state, got:\n%s", out)
	}
}

// TestPageEscapesTitle guards against HTML injection through the title.
func TestPageEscapesTitle(t *testing.T) {
	f := Frame{Title: `<script>alert(1)</script>`}
	out := Page(f, "x")
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Errorf("expected title to be HTML-escaped, got:\n%s", out)
	}
}
