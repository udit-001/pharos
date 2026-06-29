package render

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/udit-001/pharos/internal/urls"
)

// Page renders the full HTML document: the frame (sidebar + topbar + wrapper)
// wrapped around the given content HTML.
func Page(f Frame, content string) string {
	var buf bytes.Buffer
	if err := document(f, content).Render(context.Background(), &buf); err != nil {
		return "<!DOCTYPE html><html><body>" + content + "</body></html>"
	}
	return buf.String()
}

func sidebarOverlay() string {
	return `<div id="sidebar-overlay" class="fixed inset-0 bg-black/30 z-30 hidden md:hidden" onclick="toggleSidebar()"></div>`
}

func sidebarHeader(f Frame) string {
	return `<div class="flex items-center gap-2.5 px-5 py-3 border-b border-slate-200">
      <a href="/" class="flex items-center gap-2 text-sm font-semibold text-slate-800 hover:text-slate-600 no-underline">
        ` + logoSVG() + `
        Pharos
      </a>
    </div>`
}

func sidebarSection(label, sectionID, itemsHTML string, count int, sectionActive bool) string {
	chevron := `<svg class="sidebar-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`
	collapsed := ""
	if !sectionActive {
		collapsed = ` collapsed`
		chevron = `<svg class="sidebar-chevron" style="transform:rotate(90deg)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`
	}
	countHTML := ""
	if count > 0 {
		countHTML = ` <span class="sidebar-section-count">` + fmt.Sprintf("%d", count) + `</span>`
	}
	return `<div class="sidebar-section">` +
		`<div class="sidebar-section-label" data-section="` + sectionID + `" onclick="toggleSection(this)">` +
		chevron + `<span>` + esc(label) + countHTML + `</span></div>` +
		`<div class="sidebar-section-items` + collapsed + `">` + itemsHTML + `</div></div>`
}

func sidebarBody(f Frame) string {
	// Dashboard / search: sidebar is not needed — main content handles it
	if f.Sidebar.Workspace == nil {
		return `<div class="px-4 py-8 text-center text-slate-400 text-sm">
			<p class="text-xs">Select a workspace to begin</p>
		</div>`
	}

	var b strings.Builder
	ws := f.Sidebar.Workspace
	at := f.ActiveType

	// Lessons first (primary content)
	if len(f.Sidebar.Lessons) > 0 {
		var items strings.Builder
		for _, l := range f.Sidebar.Lessons {
			active := at == "lesson" && f.ActiveSeq == l.Seq
			items.WriteString(sidebarLink(urls.Lesson(ws.Name, l.Seq), iconBook(), l.Title, active))
		}
		b.WriteString(sidebarSection("Lessons", "lessons", items.String(), len(f.Sidebar.Lessons), at == "lesson"))
	}
	// Quizzes: single link to the library page (not per-item, to avoid
	// clutter as the collection grows — matches the Glossary pattern).
	quizActive := at == "quiz" || at == "quiz-library"
	b.WriteString(sidebarSection("Quizzes", "quizzes", sidebarLink(urls.QuizLibrary(ws.Name), iconClipboardList(), "All quizzes", quizActive), len(f.Sidebar.Quizzes), quizActive))
	if len(f.Sidebar.Records) > 0 {
		var items strings.Builder
		for _, r := range f.Sidebar.Records {
			active := at == "record" && f.ActiveSeq == r.Seq
			ico := iconNote()
			if r.Status == "superseded" {
				ico = iconArchive()
			}
			items.WriteString(sidebarLink(urls.Record(ws.Name, r.Seq), ico, r.Title, active))
		}
		b.WriteString(sidebarSection("Records", "records", items.String(), len(f.Sidebar.Records), at == "record"))
	}
	if len(f.Sidebar.Refs) > 0 {
		var items strings.Builder
		for _, ref := range f.Sidebar.Refs {
			active := at == "ref" && f.ActiveSlug == ref.Slug
			items.WriteString(sidebarLink(urls.Ref(ws.Name, ref.Slug), iconBookmark(), ref.Title, active))
		}
		b.WriteString(sidebarSection("References", "refs", items.String(), len(f.Sidebar.Refs), at == "ref"))
	}

	// Workspace docs at the bottom
	docs := []struct{ kind, label, icon string }{
		{"mission", "Mission", iconTarget()},
		{"resources", "Resources", iconLink()},
		{"glossary", "Glossary", iconBookOpen()},
		{"notes", "Notes", iconPencil()},
	}
	var wsItems strings.Builder
	for _, doc := range docs {
		active := at == doc.kind
		wsItems.WriteString(sidebarLink(urls.Doc(ws.Name, doc.kind), doc.icon, doc.label, active))
	}
	wsActive := at == "mission" || at == "resources" || at == "glossary" || at == "notes"
	b.WriteString(sidebarSection("Workspace", "workspace", wsItems.String(), 0, wsActive))

	return b.String()
}

func sidebarLink(href, icon, label string, active bool) string {
	cls := "sidebar-link"
	if active {
		cls = "sidebar-link-active"
	}
	return fmt.Sprintf(`<a href="%s" class="%s">%s<span class="sidebar-link-label">%s</span></a>`, href, cls, icon, esc(label))
}

func navLinkClass(isActive bool) string {
	if isActive {
		return "sidebar-link-active"
	}
	return "sidebar-link"
}

func breadcrumbs(f Frame) string {
	if f.ActiveWS == "" {
		return ""
	}
	// On the workspace landing page there's no item crumb, and the
	// "Dashboard / Workspace" trail is redundant — the sidebar logo
	// links to Dashboard and the workspace link is a self-link.
	if f.ActiveType == "" {
		return ""
	}
	wsLabel := f.ActiveWS
	if f.Sidebar.Workspace != nil {
		wsLabel = displayName(f.Sidebar.Workspace.Name, f.Sidebar.Workspace.Topic)
	}
	wsURL := urls.Workspace(f.ActiveWS)

	// Build trail: Workspace / Item (Dashboard is reachable via the
	// sidebar logo, so it doesn't earn a crumb).
	sep := `<span class="text-slate-300 mx-1 shrink-0">/</span>`
	wsLink := fmt.Sprintf(`<a href="%s" class="text-slate-400 hover:text-slate-600 no-underline text-sm truncate max-w-[40vw] block">%s</a>`, wsURL, esc(wsLabel))

	// If there's a page-level item, add it as the third crumb
	var pageCrumb string
	switch f.ActiveType {
	case "lesson":
		title := ""
		for _, l := range f.Sidebar.Lessons {
			if l.Seq == f.ActiveSeq {
				title = l.Title
				break
			}
		}
		if title == "" {
			title = fmt.Sprintf("Lesson %d", f.ActiveSeq)
		}
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block">%s</span>`, esc(title))
	case "record":
		title := ""
		for _, r := range f.Sidebar.Records {
			if r.Seq == f.ActiveSeq {
				title = r.Title
				break
			}
		}
		if title == "" {
			title = fmt.Sprintf("Record %d", f.ActiveSeq)
		}
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block">%s</span>`, esc(title))
	case "ref":
		title := ""
		for _, ref := range f.Sidebar.Refs {
			if ref.Slug == f.ActiveSlug {
				title = ref.Title
				break
			}
		}
		if title == "" {
			title = f.ActiveSlug
		}
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block">%s</span>`, esc(title))
	case "quiz":
		title := ""
		for _, q := range f.Sidebar.Quizzes {
			if q.Slug == f.ActiveSlug {
				title = q.Title
				break
			}
		}
		if title == "" {
			title = f.ActiveSlug
		}
		quizzesLink := fmt.Sprintf(`<a href="%s" class="text-slate-400 hover:text-slate-600 no-underline text-sm truncate max-w-[40vw] block">Quizzes</a>`, urls.QuizLibrary(f.ActiveWS))
		quizLink := fmt.Sprintf(`<a href="%s" class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block hover:text-slate-800 no-underline">%s</a>`, urls.Quiz(f.ActiveWS, f.ActiveSlug), esc(title))
		pageCrumb = sep + quizzesLink + sep + quizLink
	case "mission", "resources", "glossary", "notes":
		docLabels := map[string]string{"mission": "Mission", "resources": "Resources", "glossary": "Glossary", "notes": "Notes"}
		if label, ok := docLabels[f.ActiveType]; ok {
			pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block">%s</span>`, label)
		}
	case "quiz-library":
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium truncate max-w-[40vw] block">%s</span>`, "Quizzes")
	}

	return fmt.Sprintf(`<nav class="flex items-center gap-0 text-sm min-w-0">%s%s</nav>`,
		wsLink, pageCrumb)
}

func frameContentClass(isFrame bool) string {
	if isFrame {
		return " flex flex-col overflow-hidden h-full"
	}
	return ""
}

// topbarCenterBranding returns the Pharos branding centered in the topbar,
// only on the dashboard where the sidebar is hidden.
func topbarCenterBranding(f Frame) string {
	if f.ActiveWS != "" {
		return ""
	}
	return `<a href="/" class="topbar-brand flex items-center gap-2 text-sm font-semibold text-slate-800 hover:text-slate-600 no-underline">` + logoSVG() + `Pharos</a>`
}

// topbarMenuButton returns the mobile hamburger button, only when a sidebar exists.
func topbarMenuButton(f Frame) string {
	if f.ActiveWS == "" {
		return ""
	}
	return `<button class="md:hidden p-1.5 rounded-md hover:bg-slate-200 text-slate-600 cursor-pointer inline-flex items-center justify-center" onclick="toggleSidebar()" aria-label="Toggle sidebar">` + iconMenu() + `</button>`
}

// sidebarBlock returns the full sidebar HTML when inside a workspace,
// or empty string on the dashboard where the sidebar is hidden.
func sidebarBlock(f Frame) string {
	if f.ActiveWS == "" {
		return ""
	}
	ws := ""
	if f.Sidebar.Workspace != nil {
		ws = f.Sidebar.Workspace.Name
	}
	return `<aside id="sidebar" class="fixed md:relative z-40 md:z-auto flex flex-col border-r border-slate-200 shadow-sm bg-slate-100 w-60 min-w-60 overflow-hidden transition-[left] duration-200 left-0 sidebar-hidden h-full">` +
		sidebarHeader(f) +
		`<nav class="flex flex-col flex-1 overflow-y-auto pb-6" data-workspace="` + esc(ws) + `">` +
		sidebarDashLink(f) +
		sidebarBody(f) +
		`</nav></aside>`
}

// sidebarDashLink returns the Dashboard nav link, hidden when inside a workspace
// since breadcrumbs handle back-navigation.
func sidebarDashLink(f Frame) string {
	if f.ActiveWS != "" {
		return ""
	}
	cls := navLinkClass(f.ActiveWS == "" && f.ActiveType == "")
	return fmt.Sprintf(`<a href="/" class="flex items-center gap-2 px-4 py-2 text-sm no-underline cursor-pointer %s hover:bg-slate-200 hover:text-slate-900 transition-colors">%s<span>Dashboard</span></a>`, cls, iconHome())
}

// contentPaddingClass returns the padding class for the content wrapper.
// Frame pages (lessons, references) use no padding so the iframe fills
// the container edge-to-edge; other pages get standard reading padding.
func contentPaddingClass(isFrame bool) string {
	if isFrame {
		return "p-0"
	}
	return "p-6"
}

// frameMaxWidthClass returns the max-width class for the content container.
// Frame pages (lessons, references) get a wider column to give the iframe
// more room; other pages use the standard reading width.
func frameMaxWidthClass(isFrame bool) string {
	if isFrame {
		return ""
	}
	return "max-w-4xl"
}
