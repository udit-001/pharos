package render

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

var ldquo = "\u201c" // "
var rdquo = "\u201d" // "

// renderComponent renders a templ Component to a string.
func renderComponent(c templ.Component) string {
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		return ""
	}
	return buf.String()
}

func Dashboard(d DashboardData) string      { return renderComponent(dashboard(d)) }
func About() string                         { return renderComponent(about()) }
func WorkspacePage(d WorkspaceData) string  { return renderComponent(workspacePage(d)) }
func Lesson(d LessonData) string            { return renderComponent(lesson(d)) }
func Record(d RecordData) string            { return renderComponent(record(d)) }
func Ref(d RefData) string                  { return renderComponent(ref(d)) }
func QuizLibrary(d QuizLibraryData) string  { return renderComponent(quizLibrary(d)) }
func Quiz(d QuizData) string                { return renderComponent(quiz(d)) }
func Document(d DocumentData) string        { return renderComponent(documentView(d)) }
func NotFound(title, message string) string { return renderComponent(notFound(title, message)) }
func Search(d SearchData) string            { return renderComponent(search(d)) }

// workspaceMeta formats the lesson/record/ref counts for a dashboard card.
func workspaceMeta(w WorkspaceCard) string {
	counts := make([]string, 0, 3)
	if w.LessonCount > 0 {
		counts = append(counts, fmt.Sprintf("%d lessons", w.LessonCount))
	}
	if w.RecordCount > 0 {
		counts = append(counts, fmt.Sprintf("%d records", w.RecordCount))
	}
	if w.RefCount > 0 {
		counts = append(counts, fmt.Sprintf("%d refs", w.RefCount))
	}
	meta := strings.Join(counts, " · ")
	if meta == "" {
		meta = "empty"
	}
	return meta
}

type glossaryGroup struct {
	Category string
	Terms    []GlossaryTermRow
}

// groupGlossaryTerms groups consecutive terms sharing the same Category,
// preserving input order. An empty Category renders as "Other".
func groupGlossaryTerms(terms []GlossaryTermRow) []glossaryGroup {
	var groups []glossaryGroup
	for _, t := range terms {
		if len(groups) > 0 && groups[len(groups)-1].Category == t.Category {
			groups[len(groups)-1].Terms = append(groups[len(groups)-1].Terms, t)
		} else {
			groups = append(groups, glossaryGroup{Category: t.Category, Terms: []GlossaryTermRow{t}})
		}
	}
	return groups
}

// onboardingBlock renders a guided empty state for a fresh workspace with
// no lessons or learning records. It tells the learner how pharos is driven:
// set a mission, install the teach skill, then ask the agent to teach.
func onboardingBlock(displayName, mission string) string {
	var missionLine string
	if mission == "" {
		missionLine = fmt.Sprintf(`<div class="flex items-center gap-3 py-2">
			<code class="bg-slate-100 px-2 py-0.5 rounded text-xs text-slate-600 shrink-0">"I want to master %s"</code>
			<span class="text-sm text-slate-500">Tell your agent what you want to learn</span>
		</div>`, esc(displayName))
	}

	return fmt.Sprintf(`<div class="bg-white rounded-lg border border-slate-200 p-6">
		<div class="flex items-center gap-3 mb-4">
			<div class="shrink-0 text-slate-400">%s</div>
			<p class="text-sm font-medium text-slate-600">Ask your AI agent to start teaching.</p>
		</div>
		%s
		<div class="flex items-center gap-3 py-2">
			<code class="bg-slate-100 px-2 py-0.5 rounded text-xs text-slate-600 shrink-0">"teach me about %s"</code>
		</div>
	</div>`, iconCompass(), missionLine, esc(displayName))
}

// bigIcon returns an svg with its size attributes set to px×px, used for
// empty-state illustrations where the default 16px is too small.
func bigIcon(svg string, px int) string {
	size := strconv.Itoa(px)
	s := strings.Replace(svg, `width="16" height="16"`, `width="`+size+`" height="`+size+`"`, 1)
	return strings.Replace(s, `width="20" height="20"`, `width="`+size+`" height="`+size+`"`, 1)
}

func emptyStateAction(kind string) string {
	switch kind {
	case "mission":
		return `"Set a learning goal for me"`
	case "resources":
		return `"Add some reference materials"`
	case "glossary":
		return `"Define the key terms"`
	case "notes":
		return "Your agent records learning preferences here"
	}
	return ""
}

func docHint(kind string) string {
	switch kind {
	case "mission":
		return "Why you're learning this — every lesson should trace back to it"
	case "resources":
		return "Curated knowledge sources and communities"
	case "glossary":
		return "Canonical terminology for this workspace"
	case "notes":
		return "Scratchpad for preferences and working notes"
	}
	return ""
}

// orEmpty returns s if non-empty, otherwise fallback.
func orEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func pluralResults(n int) string {
	if n == 1 {
		return "result"
	}
	return "results"
}

func statusTag(status string) string {
	if status == "superseded" {
		return `<span class="inline-flex items-center bg-red-100 text-red-600 text-xs font-medium px-2 py-0.5 rounded">superseded</span>`
	}
	return `<span class="inline-flex items-center bg-emerald-100 text-emerald-600 text-xs font-medium px-2 py-0.5 rounded">active</span>`
}
