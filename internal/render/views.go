package render

import (
	"fmt"
	"strconv"
	"strings"
)

// Dashboard renders the dashboard page body.
func Dashboard(d DashboardData) string {
	var statsBlock string
	if len(d.Workspaces) > 0 {
		statsBlock = fmt.Sprintf(`<div class="flex items-center gap-6 mb-8 text-sm text-slate-400">
			<span><span class="font-semibold text-slate-600 tabular-nums">%d</span> workspaces</span>
			<span class="text-slate-200">·</span>
			<span><span class="font-semibold text-blue-700 tabular-nums">%d</span> lessons</span>
			<span class="text-slate-200">·</span>
			<span><span class="font-semibold text-emerald-600 tabular-nums">%d</span> records</span>
			<span class="text-slate-200">·</span>
			<span><span class="font-semibold text-amber-600 tabular-nums">%d</span> references</span>
		</div>`, d.Stats.Workspaces, d.Stats.Lessons, d.Stats.Records, d.Stats.Refs)
	}

	var continueBlock string
	if d.Continue != nil {
		continueBlock = fmt.Sprintf(`<div class="mb-6">
			<a href="%s" class="group inline-flex items-center gap-2 text-sm no-underline py-2 px-3 -mx-3 rounded-md hover:bg-slate-100 transition-colors">
				<span class="text-slate-300 group-hover:text-blue-700 transition-colors">&rarr;</span>
				<span class="text-slate-400 group-hover:text-slate-600 transition-colors">Continue:</span>
				<span class="font-medium text-blue-700 group-hover:text-blue-900 transition-colors">%s</span>
			</a>
		</div>`, esc(d.Continue.URL), esc(d.Continue.Label))
	}

	var listBlock string
	if len(d.Workspaces) > 0 {
		items := make([]string, len(d.Workspaces))
		for i, w := range d.Workspaces {
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

			items[i] = fmt.Sprintf(`<a href="/workspace/%s" class="group block py-3 px-3 -mx-3 no-underline rounded-md hover:bg-slate-100 transition-colors">
				<div class="flex items-center justify-between">
					<div class="font-medium text-slate-800 group-hover:text-blue-700 transition-colors">%s</div>
					<span class="text-slate-300 group-hover:text-blue-700 transition-colors text-sm">&rsaquo;</span>
				</div>
				<div class="flex items-center gap-2 text-xs text-slate-400 mt-1">
					<span>%s</span>
					<span class="text-slate-200">·</span>
					<span>last %s</span>
				</div>
			</a>`, urlPathEscape(w.Name), esc(displayName(w.Name, w.Topic)), meta, shortDate(w.LastStudied))
		}
		listBlock = `<div class="divide-y divide-slate-100">` + strings.Join(items, "") + `</div>`
	} else {
		listBlock = `<div class="text-center py-16 text-slate-400">
			<div class="text-4xl mb-4 opacity-50">📚</div>
			<h3 class="text-lg font-semibold text-slate-600 mb-2">No workspaces yet</h3>
			<p class="max-w-sm mx-auto mb-4 leading-relaxed">Create your first workspace from the terminal:</p>
			<code class="bg-slate-100 px-3 py-1.5 rounded text-sm">pharos workspace create "topic"</code>
		</div>`
	}

	return fmt.Sprintf(`
		%s
		%s
		%s
	`, statsBlock, continueBlock, listBlock)
}

// Workspace renders a workspace landing page body.
func Workspace(d WorkspaceData) string {
	ws := d.Workspace
	var missionBlock string
	if d.Mission != "" {
		missionBlock = fmt.Sprintf(`<h2 class="text-xs font-medium text-slate-400 mt-6 mb-3">Mission</h2>
		<div class="prose">%s</div>`, d.Mission)
	}

	// Fresh workspace: no lessons or records yet. Show onboarding guidance
	// instead of two empty lists — the learner needs to know how to start,
	// and that pharos is meant to be driven by their AI agent.
	if len(d.Lessons) == 0 && len(d.Records) == 0 {
		return fmt.Sprintf(`
		<h1 class="text-xl font-semibold text-slate-800 tracking-tight">%s</h1>
		<p class="text-sm text-slate-400 mt-0.5 mb-5">Workspace ready</p>
		%s
		%s
	`, esc(displayName(ws.Name, ws.Topic)), missionBlock, onboardingBlock(displayName(ws.Name, ws.Topic), d.Mission))
	}

	var lessonsBlock string
	if len(d.Lessons) == 0 {
		lessonsBlock = `<p class="text-sm text-slate-400">No lessons yet.</p>`
	} else {
		rows := make([]string, len(d.Lessons))
		for i, l := range d.Lessons {
			rows[i] = fmt.Sprintf(`<div class="py-2 border-b border-slate-100 last:border-0">
				<a href="/workspace/%s/lesson/%d" class="no-underline text-slate-800 flex justify-between items-center group">
					<span class="font-medium text-sm group-hover:text-blue-700 transition-colors">%d. %s</span>
					<span class="text-xs text-slate-400 tabular-nums">L%d</span>
				</a>
			</div>`, urlPathEscape(ws.Name), l.SequenceNumber, l.SequenceNumber, esc(l.Title), l.SequenceNumber)
		}
		lessonsBlock = strings.Join(rows, "")
	}

	var recordsBlock string
	if len(d.Records) == 0 {
		recordsBlock = `<p class="text-sm text-slate-400">No learning records yet.</p>`
	} else {
		rows := make([]string, len(d.Records))
		for i, rec := range d.Records {
			rows[i] = fmt.Sprintf(`<div class="py-2 border-b border-slate-100 last:border-0">
				<a href="/workspace/%s/record/%d" class="no-underline text-slate-800 block group">
					<div class="flex items-center justify-between">
						<span class="font-medium text-sm group-hover:text-blue-700 transition-colors">%d. %s</span>
						%s
					</div>
					<div class="text-xs text-slate-400 mt-0.5">%s</div>
				</a>
			</div>`, urlPathEscape(ws.Name), rec.SequenceNumber, rec.SequenceNumber, esc(rec.Title), statusTag(rec.Status), esc(rec.Summary))
		}
		recordsBlock = strings.Join(rows, "")
	}

	var refsBlock string
	if len(d.Refs) == 0 {
		refsBlock = `<p class="text-sm text-slate-400">No references yet.</p>`
	} else {
		rows := make([]string, len(d.Refs))
		for i, ref := range d.Refs {
			rows[i] = fmt.Sprintf(`<div class="py-2 border-b border-slate-100 last:border-0">
				<a href="/workspace/%s/ref/%s" class="no-underline text-slate-800 flex items-center gap-2 group">
					<span class="shrink-0 text-slate-300">%s</span>
					<span class="font-medium text-sm group-hover:text-blue-700 transition-colors">%s</span>
				</a>
			</div>`, urlPathEscape(ws.Name), urlPathEscape(ref.Slug), iconBookmark(), esc(ref.Title))
		}
		refsBlock = strings.Join(rows, "")
	}

	return fmt.Sprintf(`
		<h1 class="text-xl font-semibold text-slate-800 tracking-tight">%s</h1>
		<p class="text-sm text-slate-400 mt-0.5 mb-5">%d lessons · %d records · %d refs</p>
		%s
		<h2 class="text-xs font-medium text-slate-400 mt-6 mb-3">Lessons</h2>
		%s
		<h2 class="text-xs font-medium text-slate-400 mt-6 mb-3">Learning Records</h2>
		%s
		<h2 class="text-xs font-medium text-slate-400 mt-6 mb-3">References</h2>
		%s
	`, esc(displayName(ws.Name, ws.Topic)), ws.LessonCount, ws.RecordCount, ws.RefCount, missionBlock, lessonsBlock, recordsBlock, refsBlock)
}

// onboardingBlock renders a guided empty state for a fresh workspace with
// no lessons or learning records. It tells the learner how pharos is driven:
// set a mission, install the teach skill, then ask the agent to teach.
func onboardingBlock(displayName, mission string) string {
	var missionLine string
	if mission == "" {
		missionLine = fmt.Sprintf(`<div class="flex items-center gap-3 py-2">
			<code class="bg-slate-100 px-2 py-0.5 rounded text-xs text-slate-600 shrink-0">pharos mission --edit</code>
			<span class="text-sm text-slate-500">Set a goal for learning %s</span>
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

// biggerChevron swaps the 16x16 svg dimensions for 24x24 so the icon
// scales with the larger tap target without redefining the path data.
func biggerChevron(svg string) string {
	s := strings.Replace(svg, `width="16" height="16"`, `width="24" height="24"`, 1)
	return strings.Replace(s, `width="20" height="20"`, `width="24" height="24"`, 1)
}

// bigIcon returns an svg with its size attributes set to px×px, used for
// empty-state illustrations where the default 16px is too small.
func bigIcon(svg string, px int) string {
	size := strconv.Itoa(px)
	s := strings.Replace(svg, `width="16" height="16"`, `width="`+size+`" height="`+size+`"`, 1)
	return strings.Replace(s, `width="20" height="20"`, `width="`+size+`" height="`+size+`"`, 1)
}

// Lesson renders a lesson detail page body (iframe) flanked by inline
// prev/next navigation buttons so they don't consume vertical reading space.
func Lesson(d LessonData) string {
	return fmt.Sprintf(`
		<div class="flex-1 min-h-0 flex items-center gap-2">
			%s
			<iframe src="%s" class="flex-1 h-full border border-slate-200 rounded-lg bg-white" title="%s"></iframe>
			%s
		</div>
	`, lessonSideNav(d.Prev, false), esc(d.RawURL), esc(d.Title), lessonSideNav(d.Next, true))
}

// lessonSideNav renders a slim vertical nav button on the side of the
// iframe. nil target = no nav on that side; emits an invisible spacer
// to keep the iframe from shifting.
func lessonSideNav(nav *LessonNav, isNext bool) string {
	if nav == nil {
		return `<div class="w-10 shrink-0"></div>`
	}
	icon := iconChevronLeft()
	if isNext {
		icon = iconChevronRight()
	}
	return fmt.Sprintf(`<a href="%s" title="%s" class="shrink-0 self-stretch flex items-center justify-center w-10 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors no-underline">%s</a>`,
		esc(nav.URL), esc(nav.Title), biggerChevron(icon))
}

// Record renders a learning-record detail page body (rendered markdown).
func Record(d RecordData) string {
	return fmt.Sprintf(`
		<div class="mb-3">%s</div>
		<div class="prose">%s</div>
	`, statusTag(d.Status), d.BodyHTML)
}

// Ref renders a reference detail page body (iframe).
func Ref(d RefData) string {
	return fmt.Sprintf(`
		<div class="flex-1 min-h-0 flex flex-col">
			<iframe src="%s" class="w-full flex-1 border border-slate-200 rounded-lg bg-white" title="%s"></iframe>
		</div>
	`, esc(d.RawURL), esc(d.Title))
}

// Document renders a workspace document page (Mission, Resources, Glossary,
// Notes). Empty documents get a guided state pointing to the CLI command —
// the dashboard is read-only, so the learner needs to know how to edit.
func Document(d DocumentData) string {
	if d.Empty {
		cmd := docCommand(d.Kind)
		hint := docHint(d.Kind)
		return fmt.Sprintf(`
		<p class="text-sm text-slate-400 mt-0.5 mb-5">%s</p>
		<div class="bg-white rounded-lg border border-slate-200 p-10 text-center">
			<div class="flex justify-center mb-4 text-slate-300">%s</div>
			<p class="text-sm text-slate-500 mb-4">%s</p>
			<code class="bg-slate-100 px-2 py-1 rounded text-xs text-slate-700">%s</code>
		</div>
	`, hint, bigIcon(iconBookOpen(), 48), hint, esc(cmd))
	}
	return fmt.Sprintf(`
		<div class="prose mt-4">%s</div>
	`, d.BodyHTML)
}

// docCommand returns the CLI command to edit a workspace document.
func docCommand(kind string) string {
	switch kind {
	case "mission":
		return "pharos mission --edit"
	case "resources":
		return "pharos resources --edit"
	case "glossary":
		return "pharos glossary --edit"
	case "notes":
		return "# edit NOTES.md directly"
	}
	return ""
}

// docHint returns the descriptive subtitle for a document page.
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

// Search renders the search page body.
func Search(d SearchData) string {
	body := fmt.Sprintf(`
		<h1 class="text-xl font-semibold text-slate-800 tracking-tight">Search</h1>
		<p class="text-sm text-slate-400 mt-0.5 mb-5">Search across all lessons and learning records</p>
		<form action="/search" method="GET" class="mb-5">
			<input type="text" name="q" value="%s" placeholder="Search..." class="w-full py-2 px-3 border border-slate-200 rounded-lg text-sm focus:border-slate-400 focus:outline-none transition-colors">
		</form>`, esc(d.Query))

	if d.Query == "" {
		return body
	}

	if len(d.Results) == 0 {
		body += fmt.Sprintf(`<div class="text-center py-12 text-slate-400">
			<div class="text-4xl mb-4 opacity-50">%s</div>
			<p class="text-sm">No results for &ldquo;%s&rdquo;</p>
		</div>`, iconSearch(), esc(d.Query))
		return body
	}

	var results []string
	for _, r := range d.Results {
		ico := iconBook()
		typeLabel := "Lesson"
		switch r.Type {
		case "record":
			ico = iconNote()
			typeLabel = "Record"
		case "ref":
			ico = iconBookmark()
			typeLabel = "Reference"
		}
		results = append(results, fmt.Sprintf(`<div class="py-3 border-b border-slate-100 last:border-0">
			<a href="%s" class="font-medium text-blue-700 hover:text-blue-900 no-underline flex items-center gap-2">%s %s</a>
			<div class="text-sm text-slate-400 mt-0.5">%s in <strong class="text-slate-500">%s</strong>%s</div>
		</div>`, esc(r.URL), ico, esc(r.Title), typeLabel, esc(r.Workspace), withSummary(r.Summary)))
	}
	body += `<div class="bg-white rounded-lg border border-slate-200 p-4">` + strings.Join(results, "") + `</div>`
	return body
}

// statusTag renders an active/superseded badge.
func statusTag(status string) string {
	if status == "superseded" {
		return `<span class="inline-flex items-center bg-red-100 text-red-600 text-xs font-medium px-2 py-0.5 rounded">superseded</span>`
	}
	return `<span class="inline-flex items-center bg-emerald-100 text-emerald-600 text-xs font-medium px-2 py-0.5 rounded">active</span>`
}

func withSummary(s string) string {
	if s == "" {
		return ""
	}
	return " — " + esc(s)
}
