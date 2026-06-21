package render

import (
	"html"
	"strings"
)

// esc HTML-escapes a string.
func esc(s string) string { return html.EscapeString(s) }

// urlPathEscape replaces spaces for URL path segments.
func urlPathEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

// shortDate returns the YYYY-MM-DD prefix of a timestamp.
func shortDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// displayName picks the friendly Topic if set, else the slug Name. Used for
// visible text only — URLs and keys always use the raw Name.
func displayName(name, topic string) string {
	if topic != "" {
		return topic
	}
	return name
}

// ── Lucide icons as inline SVGs ──

func iconBook() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="shrink-0"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H19a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H6.5a1 1 0 0 1 0-5H20"/></svg>`
}

func iconNote() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="shrink-0"><path d="M15.5 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V8.5Z"/><path d="M15 3v6h6"/></svg>`
}

func iconArchive() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="shrink-0"><rect width="20" height="5" x="2" y="4" rx="2"/><path d="M4 9v9a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9"/><path d="M10 13h4"/></svg>`
}

func iconFolder() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="shrink-0"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>`
}

func iconSearch() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`
}

func iconMenu() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="12" y2="12"/><line x1="4" x2="20" y1="6" y2="6"/><line x1="4" x2="20" y1="18" y2="18"/></svg>`
}

func iconHome() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`
}

// logoSVG returns the Learn logo.
func logoSVG() string {
	return `<svg class="shrink-0" viewBox="0 0 100 100" width="28" height="28" aria-hidden="true">
		<g fill="none" stroke="currentColor" stroke-linecap="round">
			<circle cx="50" cy="50" r="32" stroke-width="4"/>
			<circle cx="50" cy="50" r="16" stroke-width="1.5" stroke-dasharray="4 6" opacity="0.4"/>
			<circle cx="50" cy="18" r="4" fill="currentColor" stroke="none"/>
			<circle cx="82" cy="50" r="4" fill="currentColor" stroke="none"/>
			<circle cx="50" cy="82" r="4" fill="currentColor" stroke="none"/>
			<circle cx="18" cy="50" r="4" fill="currentColor" stroke="none"/>
			<polygon points="50,40 60,50 50,60 40,50" fill="currentColor" stroke="none"/>
			<path d="M 50 18 A 32 32 0 0 1 82 50" stroke-width="2.5" opacity="0.5"/>
			<circle cx="50" cy="50" r="2" fill="currentColor" stroke="none"/>
		</g>
	</svg>`
}
