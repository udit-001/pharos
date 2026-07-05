package render

import (
	"strings"
	"testing"
)

// TestDashboardRendersStatsAndWorkspaces proves the render module's output is
// a pure function of its view model — the seam created in LEARN-10.
func TestDashboardRendersStatsAndWorkspaces(t *testing.T) {
	d := DashboardData{
		Stats: Stats{Workspaces: 2, Lessons: 5, Records: 3, Refs: 1, Quizzes: 2},
		Workspaces: []WorkspaceCard{
			{Name: "sql-basics", LessonCount: 3, RecordCount: 1, RefCount: 1, QuizCount: 1, LastStudied: "2026-06-21"},
			{Name: "golang", LessonCount: 2, RecordCount: 2, RefCount: 0, QuizCount: 1, LastStudied: "2026-06-20"},
		},
	}
	out := Dashboard(d)

	for _, want := range []string{
		"workspaces",
		">2<", // workspaces count
		">5<", // lessons count
		">3<", // records count
		">1<", // refs count
		">2<", // quizzes count
		"sql-basics",
		"golang",
		"3 lessons",
		"1 quizzes",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestDashboardEmptyState(t *testing.T) {
	out := Dashboard(DashboardData{})
	if !strings.Contains(out, "Your learning dashboard") {
		t.Errorf("expected empty state heading, got:\n%s", out)
	}
	if !strings.Contains(out, "Teach me about") {
		t.Errorf("expected teach skill hint, got:\n%s", out)
	}
}

func TestDashboardContinueCard(t *testing.T) {
	d := DashboardData{
		Continue: &ContinueItem{URL: "/workspace/sql-basics/lesson/2", Label: "sql-basics — Lesson: JOINs"},
	}
	out := Dashboard(d)
	if !strings.Contains(out, "Continue:") {
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
	if !strings.Contains(strings.ToLower(out), "<!doctype html>") {
		t.Errorf("expected doctype, got:\n%s", out)
	}
	if !strings.Contains(out, "<title>My Page — Pharos</title>") {
		t.Errorf("expected title tag, got:\n%s", out)
	}
	if !strings.Contains(out, "<p>BODY</p>") {
		t.Errorf("expected body content preserved, got:\n%s", out)
	}
}

func TestPageDashboardNoSidebar(t *testing.T) {
	f := Frame{Title: "Dashboard", ActiveWS: ""}
	out := Page(f, "x")
	if strings.Contains(out, "<aside") {
		t.Errorf("expected no sidebar on dashboard, got:\n%s", out)
	}
	if !strings.Contains(out, "Pharos") {
		t.Errorf("expected branding in topbar, got:\n%s", out)
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

// TestPaletteDataScriptEmpty verifies the dashboard (no active workspace)
// emits an empty array so the palette JS always parses valid JSON.
func TestPaletteDataScriptEmpty(t *testing.T) {
	got := Frame{}.PaletteDataScript()
	if !strings.Contains(got, `<script type="application/json" id="pharos-palette-data">[]</script>`) {
		t.Errorf("expected empty data script tag, got:\n%s", got)
	}
}

// TestPaletteDataScriptFromSidebar verifies the palette data module maps
// every sidebar item kind to its dashboard URL — the deep behaviour behind
// the one-method interface.
func TestPaletteDataScriptFromSidebar(t *testing.T) {
	f := Frame{
		ActiveWS: "sql-basics",
		Sidebar: Sidebar{
			Workspace: &Workspace{Name: "sql-basics"},
			Lessons:   []LessonEntry{{Seq: 1, Title: "JOINs"}},
			Records:   []RecordEntry{{Seq: 2, Title: "Notes on indexes"}},
			Refs:      []RefEntry{{Slug: "ddl-cheatsheet", Title: "DDL cheatsheet"}},
			Quizzes:   []QuizEntry{{Slug: "q1", Title: "Quiz 1"}},
		},
	}
	got := f.PaletteDataScript()
	for _, want := range []string{
		`"type":"lesson"`, `"title":"JOINs"`, `"url":"/workspace/sql-basics/lesson/1"`,
		`"type":"record"`, `"url":"/workspace/sql-basics/record/2"`,
		`"type":"ref"`, `"url":"/workspace/sql-basics/ref/ddl-cheatsheet"`,
		`"type":"quiz"`, `"url":"/workspace/sql-basics/quiz/q1"`,
		`"type":"doc"`, `"url":"/workspace/sql-basics/mission"`,
		`"url":"/workspace/sql-basics/glossary"`,
		`"workspace":"sql-basics"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in palette data, got:\n%s", want, got)
		}
	}
}

// TestPaletteDataScriptEscapesHTML guards against a title containing
// "</script>" breaking out of the embedded JSON script tag.
func TestPaletteDataScriptEscapesHTML(t *testing.T) {
	f := Frame{
		Sidebar: Sidebar{
			Workspace: &Workspace{Name: "ws"},
			Lessons:   []LessonEntry{{Seq: 1, Title: `</script><img src=x>`}},
		},
	}
	got := f.PaletteDataScript()
	if strings.Contains(got, "</script><img") {
		t.Errorf("expected </script> escaped in JSON, got:\n%s", got)
	}
	if !strings.Contains(got, `\u003c/script\u003e`) {
		t.Errorf("expected \\u003c escaping for </script>, got:\n%s", got)
	}
}

func TestQuizLibraryRendersQuizzes(t *testing.T) {
	d := QuizLibraryData{
		Workspace: Workspace{Name: "alpha"},
		Quizzes: []QuizEntry{
			{Slug: "genetics-foundations", Title: "Genetics foundations", Description: "Core factors", ItemCount: 5},
		},
	}
	out := QuizLibrary(d)
	for _, want := range []string{
		"Genetics foundations",
		"Core factors",
		"5 questions",
		"/workspace/alpha/quiz/genetics-foundations",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestQuizLibraryEmptyState(t *testing.T) {
	out := QuizLibrary(QuizLibraryData{Workspace: Workspace{Name: "alpha"}})
	for _, want := range []string{
		"Quizzes test what you've learned.",
		`"Quiz me on what I've learned"`,
		"bg-white rounded-lg border border-slate-200", // card, matching documentView empty state
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected empty state to contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "0 quizzes") {
		t.Errorf("empty state should not show a zero count, got:\n%s", out)
	}
}

func TestQuizDetailRendersStartButton(t *testing.T) {
	d := QuizData{
		Workspace:   Workspace{Name: "alpha"},
		Slug:        "genetics-foundations",
		Title:       "Genetics foundations",
		Description: "Core genetic factors in ASD",
		ItemCount:   5,
	}
	out := Quiz(d)
	for _, want := range []string{
		"Genetics foundations",
		"Core genetic factors in ASD",
		"5 questions",
		"Start quiz",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}
