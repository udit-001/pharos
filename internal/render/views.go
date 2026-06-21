package render

import (
	"fmt"
	"strings"
)

// Dashboard renders the dashboard page body.
func Dashboard(d DashboardData) string {
	var statsBlock string
	if len(d.Workspaces) > 0 {
		statsBlock = fmt.Sprintf(`<div class="grid grid-cols-4 gap-2.5 mb-6">
			<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors text-center">
				<div class="text-xs uppercase tracking-wide text-slate-400 font-medium">Workspaces</div>
				<div class="text-2xl font-bold text-slate-800 tabular-nums mt-1">%d</div>
			</div>
			<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors text-center">
				<div class="text-xs uppercase tracking-wide text-slate-400 font-medium">Lessons</div>
				<div class="text-2xl font-bold tabular-nums mt-1 text-blue-700">%d</div>
			</div>
			<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors text-center">
				<div class="text-xs uppercase tracking-wide text-slate-400 font-medium">Records</div>
				<div class="text-2xl font-bold tabular-nums mt-1 text-emerald-600">%d</div>
			</div>
			<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors text-center">
				<div class="text-xs uppercase tracking-wide text-slate-400 font-medium">References</div>
				<div class="text-2xl font-bold tabular-nums mt-1 text-amber-600">%d</div>
			</div>
		</div>`, d.Stats.Workspaces, d.Stats.Lessons, d.Stats.Records, d.Stats.Refs)
	}

	var continueBlock string
	if d.Continue != nil {
		continueBlock = fmt.Sprintf(`<div class="bg-white rounded-lg border border-blue-200 p-3 hover:border-slate-400 transition-colors mb-4">
			<div class="text-xs text-slate-400 mb-0.5">Continue where you left off</div>
			<a href="%s" class="font-semibold text-blue-700 hover:text-blue-900 no-underline text-sm">%s</a>
		</div>`, esc(d.Continue.URL), esc(d.Continue.Label))
	}

	var listBlock string
	if len(d.Workspaces) > 0 {
		items := make([]string, len(d.Workspaces))
		for i, w := range d.Workspaces {
			items[i] = fmt.Sprintf(`<a href="/workspace/%s" class="bg-white rounded-lg border border-slate-200 p-4 no-underline text-slate-800 hover:border-slate-400 hover:shadow-sm transition-all block">
				<div class="font-medium text-slate-800">%s</div>
				<div class="text-sm text-slate-400 mt-1">%d lessons · %d records · %d refs</div>
				<div class="text-xs text-slate-300 mt-2">last %s</div>
			</a>`, urlPathEscape(w.Name), esc(displayName(w.Name, w.Topic)), w.LessonCount, w.RecordCount, w.RefCount, shortDate(w.LastStudied))
		}
		listBlock = `<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">` + strings.Join(items, "") + `</div>`
	} else {
		listBlock = `<div class="text-center py-16 text-slate-400">
			<div class="text-4xl mb-4 opacity-50">📚</div>
			<h3 class="text-lg font-semibold text-slate-600 mb-2">No workspaces yet</h3>
			<p class="max-w-sm mx-auto mb-4 leading-relaxed">Create your first workspace from the terminal:</p>
			<code class="bg-slate-100 px-3 py-1.5 rounded text-sm">pharos init "topic"</code>
		</div>`
	}

	return fmt.Sprintf(`
		<h1 class="text-xl font-semibold text-slate-800 tracking-tight">Dashboard</h1>
		<p class="text-sm text-slate-400 mt-0.5 mb-5">Your learning workspaces</p>
		%s
		%s
		%s
	`, statsBlock, continueBlock, listBlock)
}

// Workspace renders a workspace landing page body.
func Workspace(d WorkspaceData) string {
	ws := d.Workspace
	var missionBlock string
	if ws.MissionWhy != "" {
		missionBlock = fmt.Sprintf(`<h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400 mt-6 mb-3">Mission</h2>
		<pre class="bg-white border border-slate-200 rounded-lg p-4 whitespace-pre-wrap font-sans text-sm leading-relaxed text-slate-500">%s</pre>`, esc(ws.MissionWhy))
	}

	var lessonsBlock string
	if len(d.Lessons) == 0 {
		lessonsBlock = `<p class="text-sm text-slate-400">No lessons yet.</p>`
	} else {
		rows := make([]string, len(d.Lessons))
		for i, l := range d.Lessons {
			rows[i] = fmt.Sprintf(`<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors mb-2">
				<a href="/workspace/%s/lesson/%d" class="no-underline text-slate-800 flex justify-between items-center">
					<span class="font-medium">%d. %s</span>
					<span class="inline-flex items-center bg-amber-100 text-amber-700 text-xs font-medium px-2 py-0.5 rounded">L%d</span>
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
			rows[i] = fmt.Sprintf(`<div class="bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-400 transition-colors mb-2">
				<a href="/workspace/%s/record/%d" class="no-underline text-slate-800">
					<div class="font-medium flex items-center gap-2">%d. %s %s</div>
					<div class="text-sm text-slate-400 mt-0.5">%s</div>
				</a>
			</div>`, urlPathEscape(ws.Name), rec.SequenceNumber, rec.SequenceNumber, esc(rec.Title), statusTag(rec.Status), esc(rec.Summary))
		}
		recordsBlock = strings.Join(rows, "")
	}

	return fmt.Sprintf(`
		<h1 class="text-xl font-semibold text-slate-800 tracking-tight">%s</h1>
		<p class="text-sm text-slate-400 mt-0.5 mb-5">%d lessons · %d records · %d refs</p>
		%s
		<h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400 mt-6 mb-3">Lessons</h2>
		%s
		<h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400 mt-6 mb-3">Learning Records</h2>
		%s
	`, esc(displayName(ws.Name, ws.Topic)), ws.LessonCount, ws.RecordCount, ws.RefCount, missionBlock, lessonsBlock, recordsBlock)
}

// Lesson renders a lesson detail page body (iframe).
func Lesson(d LessonData) string {
	return fmt.Sprintf(`
		<div class="flex-1 min-h-0 flex flex-col">
			<iframe src="%s" class="w-full flex-1 border border-slate-200 rounded-lg bg-white" title="%s"></iframe>
		</div>
	`, esc(d.RawURL), esc(d.Title))
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
