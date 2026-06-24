package render

import (
	"fmt"
	"strings"
)

// Page renders the full HTML document: the frame (sidebar + topbar + wrapper)
// wrapped around the given content HTML.
func Page(f Frame, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s — Pharos</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/css/app.css?v=3">
<script>(function(){var t=localStorage.getItem('pharos_theme');if(!t){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t})()</script>
</head>
<body class="font-sans">
<div class="flex h-screen overflow-hidden bg-white text-slate-800">

  %s

  %s

  <main class="flex flex-col flex-1 overflow-hidden">
    <header class="relative flex items-center justify-between gap-4 min-h-12 px-4 md:px-6 py-2.5 bg-stone-50 border-b border-slate-200">
      <div class="flex items-center gap-3 min-w-0">
        %s
        %s
        %s
      </div>
      %s
      <div class="flex items-center gap-2">
        <form action="/search" method="GET" class="flex items-center">
          <div class="flex items-center border border-slate-200 rounded-lg px-2.5 py-1.5 bg-white focus-within:border-slate-400 transition-colors">
            <input type="text" name="q" placeholder="Search..." aria-label="Search" value="%s" class="bg-transparent border-none outline-none w-40 text-sm text-slate-700 placeholder-slate-400 focus:w-52 transition-all">
          </div>
        </form>
        <a href="/about" class="p-1.5 rounded hover:bg-slate-200 text-slate-600 hover:text-slate-600 no-underline inline-flex items-center justify-center" title="About Pharos">`+iconHelp()+`</a>
        <button id="theme-toggle" onclick="toggleTheme()" class="p-1.5 rounded hover:bg-slate-200 text-slate-600 cursor-pointer inline-flex items-center justify-center" title="Toggle theme">
          <svg data-theme-icon="moon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
          <svg data-theme-icon="sun" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="hidden"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
        </button>
      </div>
    </header>

    <div class="flex-1 overflow-y-auto %s">
      <div class="%s mx-auto%s">
        %s
      </div>
    </div>
  </main>

</div>

<script>
function toggleSidebar() {
  var sb = document.getElementById('sidebar');
  var ov = document.getElementById('sidebar-overlay');
  sb.classList.toggle('sidebar-hidden');
  ov.classList.toggle('hidden');
}
function toggleTheme() {
  var html = document.documentElement;
  var isDark = html.dataset.theme === 'dark';
  var next = isDark ? 'light' : 'dark';
  html.dataset.theme = next;
  localStorage.setItem('pharos_theme', next);
  document.querySelectorAll('iframe').forEach(function(f) {
    try { f.contentWindow.postMessage({type: 'theme', theme: next}, '*'); } catch(e) {}
  });
  document.querySelector('[data-theme-icon=sun]').classList.toggle('hidden', next !== 'dark');
  document.querySelector('[data-theme-icon=moon]').classList.toggle('hidden', next === 'dark');
}
(function() {
  if (document.documentElement.dataset.theme === 'dark') {
    document.querySelector('[data-theme-icon=sun]').classList.remove('hidden');
    document.querySelector('[data-theme-icon=moon]').classList.add('hidden');
  }
})();
</script>
</body>
</html>`,
		esc(f.Title),
		sidebarOverlay(),
		sidebarBlock(f),
		topbarMenuButton(f),
		breadcrumbs(f),
		topbarTitle(f),
		topbarCenterBranding(f),
		esc(f.SearchQuery),
		contentPaddingClass(f.FrameContent()),
		frameMaxWidthClass(f.FrameContent()),
		frameContentClass(f.FrameContent()),
		content,
	)
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

func sidebarBody(f Frame) string {
	// Dashboard / search: sidebar is not needed — main content handles it
	if f.Sidebar.Workspace == nil {
		return `<div class="px-4 py-8 text-center text-slate-400 text-sm">
			<p class="text-xs">Select a workspace to begin</p>
		</div>`
	}

	var b strings.Builder
	ws := f.Sidebar.Workspace

	// Lessons first (primary content)
	if len(f.Sidebar.Lessons) > 0 {
		b.WriteString(`<div class="sidebar-section-label">Lessons</div>`)
		for _, l := range f.Sidebar.Lessons {
			active := f.ActiveType == "lesson" && f.ActiveSeq == l.Seq
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(ws.Name), l.Seq), iconBook(), l.Title, active))
		}
	}
	if len(f.Sidebar.Records) > 0 {
		b.WriteString(`<div class="sidebar-section-label">Records</div>`)
		for _, r := range f.Sidebar.Records {
			active := f.ActiveType == "record" && f.ActiveSeq == r.Seq
			ico := iconNote()
			if r.Status == "superseded" {
				ico = iconArchive()
			}
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(ws.Name), r.Seq), ico, r.Title, active))
		}
	}
	if len(f.Sidebar.Refs) > 0 {
		b.WriteString(`<div class="sidebar-section-label">References</div>`)
		for _, ref := range f.Sidebar.Refs {
			active := f.ActiveType == "ref" && f.ActiveSlug == ref.Slug
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/ref/%s", urlPathEscape(ws.Name), urlPathEscape(ref.Slug)), iconBookmark(), ref.Title, active))
		}
	}

	// Workspace docs at the bottom
	docs := []struct{ kind, label, icon string }{
		{"mission", "Mission", iconTarget()},
		{"resources", "Resources", iconLink()},
		{"glossary", "Glossary", iconBookOpen()},
		{"notes", "Notes", iconPencil()},
	}
	b.WriteString(`<div class="sidebar-section-label">Workspace</div>`)
	for _, doc := range docs {
		active := f.ActiveType == doc.kind
		b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/%s", urlPathEscape(ws.Name), doc.kind), doc.icon, doc.label, active))
	}

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
	wsLabel := f.ActiveWS
	if f.Sidebar.Workspace != nil {
		wsLabel = displayName(f.Sidebar.Workspace.Name, f.Sidebar.Workspace.Topic)
	}
	wsURL := fmt.Sprintf("/workspace/%s", urlPathEscape(f.ActiveWS))

	// Build trail: always starts with Dashboard / Workspace
	sep := `<span class="text-slate-300 mx-1 shrink-0">/</span>`
	dashLink := `<a href="/" class="text-slate-400 hover:text-slate-600 no-underline text-sm">Dashboard</a>`
	wsLink := fmt.Sprintf(`<a href="%s" class="text-slate-400 hover:text-slate-600 no-underline text-sm">%s</a>`, wsURL, esc(wsLabel))

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
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium">%s</span>`, esc(title))
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
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium">%s</span>`, esc(title))
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
		pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium">%s</span>`, esc(title))
	case "mission", "resources", "glossary", "notes":
		docLabels := map[string]string{"mission": "Mission", "resources": "Resources", "glossary": "Glossary", "notes": "Notes"}
		if label, ok := docLabels[f.ActiveType]; ok {
			pageCrumb = sep + fmt.Sprintf(`<span class="text-slate-600 text-sm font-medium">%s</span>`, label)
		}
	}

	return fmt.Sprintf(`<nav class="flex items-center gap-0 text-sm min-w-0">%s%s%s%s</nav>`,
		dashLink, sep, wsLink, pageCrumb)
}

func frameContentClass(isFrame bool) string {
	if isFrame {
		return " flex flex-col overflow-hidden h-full"
	}
	return ""
}

// topbarTitle returns the page title for the topbar. Currently unused —
// branding is handled by topbarCenterBranding on the dashboard, and
// breadcrumbs show the page name inside workspaces.
func topbarTitle(f Frame) string {
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
	return `<aside id="sidebar" class="fixed md:relative z-40 md:z-auto flex flex-col border-r border-slate-200 shadow-sm bg-slate-100 w-60 min-w-60 overflow-hidden transition-[left] duration-200 left-0 sidebar-hidden h-full">` +
		sidebarHeader(f) +
		`<nav class="flex flex-col flex-1 overflow-y-auto">` +
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
