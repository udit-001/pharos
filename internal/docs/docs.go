package docs

import "strings"

// defaultNotesProse is the seed content of NOTES.md — an unedited Notes
// file is treated as empty so the learner gets guidance instead of
// boilerplate. Matches internal/db/seed/NOTES.md (post-TrimSpace).
const defaultNotesProse = "# Notes\n\nPreferences and working notes for this workspace."

// IsTemplate reports whether content is an unfilled workspace document
// template: empty, containing a {placeholder} marker, or (for notes) still
// the default seed prose. kind is the document kind ("mission",
// "resources", "notes"). An unfilled doc renders as empty so the page
// shows guidance instead of boilerplate.
func IsTemplate(content, kind string) bool {
	if content == "" {
		return true
	}
	if strings.Contains(content, "{") {
		return true
	}
	if kind == "notes" && strings.HasPrefix(content, defaultNotesProse) {
		return true
	}
	return false
}

// StripH1 removes a leading "# ..." heading (the first line if it starts
// with "# "). Document templates start with an H1 that duplicates the
// navbar title. Returns the content unchanged if it has no leading H1;
// returns "" if the H1 was the only content.
func StripH1(content string) string {
	if !strings.HasPrefix(content, "# ") {
		return content
	}
	nl := strings.IndexByte(content, '\n')
	if nl < 0 {
		return ""
	}
	return strings.TrimSpace(content[nl+1:])
}
