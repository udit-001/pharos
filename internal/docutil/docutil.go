package docutil

import "strings"

// defaultNotesProse is the seed content of NOTES.md — an unedited Notes
// file is treated as empty so the learner gets guidance instead of
// boilerplate. Matches internal/db/seed/NOTES.md (post-TrimSpace).
const defaultNotesProse = "# Notes\n\nPreferences and working notes for this workspace."

// IsTemplate reports whether content is an unfilled workspace document
// template: empty, containing only placeholder markers (no real prose),
// or (for notes) still the default seed prose. kind is the document kind
// ("mission", "resources", "notes"). An unfilled doc renders as empty so
// the page shows guidance instead of boilerplate.
//
// For mission and resources, a file is a template if every non-heading,
// non-empty line is a placeholder line (starts with "{" or "- {"). A
// file with real prose alongside a leftover "{...}" placeholder line is
// NOT a template — the user has started editing, so render what they
// wrote. This avoids the false-positive where a single stray brace in
// real content hides the entire page.
//
// Notes are exempt from the placeholder-line check: real notes routinely
// contain "{" in code, JSON, or regex. The notes seed has no placeholder
// markers, so its unedited state is caught by defaultNotesProse instead.
func IsTemplate(content, kind string) bool {
	if content == "" {
		return true
	}
	if kind == "notes" {
		return content == defaultNotesProse
	}
	// mission / resources: template if no real content lines exist.
	return !hasRealContent(content)
}

// hasRealContent reports whether content has at least one non-heading,
// non-empty line that is not a placeholder line. A placeholder line
// starts with "{" or "- {" (after trimming whitespace) — matching the
// seed template markers like "{fill in}" and "- {specific outcome}".
//
// Lines inside fenced code blocks (``` or ~~~) always count as real
// content — a mission/resources doc with an embedded JSON or code
// snippet must not be misclassified as an unfilled template just
// because every line happens to start with "{" or "#".
func hasRealContent(content string) bool {
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			if trimmed != "" {
				return true
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "- {") {
			continue
		}
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
