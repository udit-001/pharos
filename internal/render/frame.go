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
<link rel="stylesheet" href="/css/app.css">
</head>
<body class="font-sans">
<div class="flex h-screen overflow-hidden bg-slate-100 text-slate-800">

  %s

  <!-- Sidebar -->
  <aside id="sidebar" class="fixed md:relative z-40 md:z-auto flex flex-col border-r border-slate-200 bg-slate-100 w-60 min-w-60 overflow-hidden transition-all duration-200 left-0 sidebar-hidden h-full">
    %s
    <nav class="flex flex-col flex-1 overflow-y-auto">
      <a href="/" class="flex items-center gap-2 px-4 py-2 text-sm no-underline cursor-pointer %s hover:bg-slate-200 hover:text-slate-900 transition-colors">
        %s
        <span>Dashboard</span>
      </a>
      <div class="border-t border-slate-200 my-1"></div>
      %s
    </nav>
  </aside>

  <main class="flex flex-col flex-1 overflow-hidden">
    <header class="flex items-center justify-between gap-4 min-h-10 px-4 md:px-6 py-1.5 bg-stone-50 border-b border-slate-200">
      <div class="flex items-center gap-3 min-w-0">
        <button class="md:hidden p-1 rounded hover:bg-slate-200 text-slate-600 cursor-pointer inline-flex items-center justify-center" onclick="toggleSidebar()" aria-label="Toggle sidebar">
          %s
        </button>
        %s
        <h2 class="text-base font-semibold text-slate-800 whitespace-nowrap truncate">%s</h2>
      </div>
      <div class="flex items-center gap-2">
        <div class="flex items-center bg-slate-100 rounded-lg px-2">
          <form action="/search" method="GET" class="flex items-center">
            <input type="text" name="q" placeholder="Search lessons & records..." aria-label="Search" class="bg-transparent border-none outline-none w-48 py-1.5 px-2 text-sm text-slate-700 placeholder-slate-400 focus:w-64 transition-all">
          </form>
        </div>
      </div>
    </header>

    <div class="flex-1 p-6 overflow-y-auto">
      <div class="max-w-4xl mx-auto%s">
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
</script>
</body>
</html>`,
		esc(f.Title),
		sidebarOverlay(),
		sidebarHeader(f),
		navLinkClass(f.ActiveWS == "" && f.ActiveType == ""),
		iconHome(),
		sidebarBody(f),
		iconMenu(),
		breadcrumbs(f),
		esc(f.Title),
		frameContentClass(f.FrameContent()),
		content,
	)
}

func sidebarOverlay() string {
	return `<div id="sidebar-overlay" class="fixed inset-0 bg-black/30 z-30 hidden md:hidden" onclick="toggleSidebar()"></div>`
}

func sidebarHeader(f Frame) string {
	return `<div class="flex items-center gap-2.5 px-5 py-2 border-b border-slate-200">
      <a href="/" class="flex items-center gap-2 text-sm font-semibold text-slate-800 hover:text-slate-600 no-underline">
        ` + logoSVG() + `
        Pharos
      </a>
    </div>`
}

func sidebarBody(f Frame) string {
	if f.Sidebar.Workspace == nil {
		return `<div class="px-4 py-6 text-center text-slate-400 text-sm"><p>Select a workspace</p><p class="text-xs mt-1">from the dashboard</p></div>`
	}
	var b strings.Builder
		ws := f.Sidebar.Workspace
	b.WriteString(fmt.Sprintf(`<a href="/workspace/%s" class="flex items-center gap-2 px-4 py-2 border-b border-slate-200 text-sm font-semibold text-slate-800 hover:text-slate-600 no-underline">%s%s</a>`,
		urlPathEscape(ws.Name), iconFolder(), esc(displayName(ws.Name, ws.Topic))))

	if len(f.Sidebar.Lessons) > 0 {
		b.WriteString(`<div class="block px-4 pt-4 pb-1 text-xs font-semibold uppercase tracking-wider text-slate-400">Lessons</div>`)
		for _, l := range f.Sidebar.Lessons {
			active := f.ActiveType == "lesson" && f.ActiveSeq == l.SequenceNumber
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(ws.Name), l.SequenceNumber), iconBook(), l.Title, active))
		}
	}
	if len(f.Sidebar.Records) > 0 {
		b.WriteString(`<div class="block px-4 pt-4 pb-1 text-xs font-semibold uppercase tracking-wider text-slate-400">Records</div>`)
		for _, r := range f.Sidebar.Records {
			active := f.ActiveType == "record" && f.ActiveSeq == r.SequenceNumber
			ico := iconNote()
			if r.Status == "superseded" {
				ico = iconArchive()
			}
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(ws.Name), r.SequenceNumber), ico, r.Title, active))
		}
	}
	if len(f.Sidebar.Refs) > 0 {
		b.WriteString(`<div class="block px-4 pt-4 pb-1 text-xs font-semibold uppercase tracking-wider text-slate-400">References</div>`)
		for _, ref := range f.Sidebar.Refs {
			active := f.ActiveType == "ref" && f.ActiveSlug == ref.Slug
			b.WriteString(sidebarLink(fmt.Sprintf("/workspace/%s/ref/%s", urlPathEscape(ws.Name), urlPathEscape(ref.Slug)), iconBook(), ref.Title, active))
		}
	}
	return b.String()
}

func sidebarLink(href, icon, label string, active bool) string {
	cls := "flex items-center gap-2 px-4 py-1.5 text-sm no-underline cursor-pointer transition-colors"
	if active {
		cls += " border-l-2 border-slate-700 bg-slate-200/60 text-slate-800 font-medium"
	} else {
		cls += " text-slate-600 hover:bg-slate-200 hover:text-slate-900 border-l-2 border-transparent"
	}
	return fmt.Sprintf(`<a href="%s" class="%s">%s<span>%s</span></a>`, href, cls, icon, esc(label))
}

func navLinkClass(isActive bool) string {
	if isActive {
		return "border-l-2 border-slate-700 bg-slate-200/60 text-slate-800 font-medium"
	}
	return "text-slate-600 border-l-2 border-transparent"
}

func breadcrumbs(f Frame) string {
	if f.ActiveWS == "" {
		return ""
	}
	// Prefer the friendly Topic from the sidebar workspace; fall back to the slug.
	label := f.ActiveWS
	if f.Sidebar.Workspace != nil {
		label = displayName(f.Sidebar.Workspace.Name, f.Sidebar.Workspace.Topic)
	}
	return fmt.Sprintf(`<nav class="flex items-center gap-1.5 text-sm min-w-0">
		<a href="/" class="text-slate-500 hover:text-slate-700 cursor-pointer bg-transparent border-none p-0 text-sm no-underline">Dashboard</a>
		<span class="text-slate-300 mx-0.5 shrink-0">/</span>
		<a href="/workspace/%s" class="text-slate-500 hover:text-slate-700 cursor-pointer bg-transparent border-none p-0 text-sm no-underline">%s</a>
	</nav>`, urlPathEscape(f.ActiveWS), esc(label))
}

func frameContentClass(isFrame bool) string {
	if isFrame {
		return " flex flex-col overflow-hidden h-full"
	}
	return ""
}
