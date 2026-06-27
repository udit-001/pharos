package db

import (
	_ "embed"
	"os"
	"strings"
)

// Workspace seed files — the default content written into a freshly created
// workspace. Real files (lintable, syntax-highlighted, previewable) embedded
// at compile time; one source of truth for what a new workspace contains.
// See PAGE-THEME.md in the teach skill for the design conventions these
// files follow.

//go:embed seed/style.css
var seedStyleCSS string

//go:embed seed/glossary-tooltip.js
var seedGlossaryTooltipJS string

//go:embed seed/MISSION.md
var seedMissionMD string

//go:embed seed/RESOURCES.md
var seedResourcesMD string

//go:embed seed/NOTES.md
var seedNotesMD string

// seedWorkspaceDefaults writes the default workspace content (CSS/JS assets,
// MISSION/RESOURCES/NOTES templates) into the given layout's root. displayName
// is substituted into the two parameterized markdown templates. Existing
// files are preserved — the seed only writes when the target is absent, so
// re-running on an existing workspace won't clobber user edits.
func seedWorkspaceDefaults(layout Layout, displayName string) error {
	files := []struct {
		path    string
		content string
	}{
		{layout.AssetPath("style.css"), seedStyleCSS},
		{layout.AssetPath("glossary-tooltip.js"), seedGlossaryTooltipJS},
		{layout.MissionPath(), strings.ReplaceAll(seedMissionMD, "{{DISPLAY_NAME}}", displayName)},
		{layout.ResourcesPath(), strings.ReplaceAll(seedResourcesMD, "{{DISPLAY_NAME}}", displayName)},
		{layout.NotesPath(), seedNotesMD},
	}
	for _, f := range files {
		if _, err := os.Stat(f.path); err == nil {
			continue // file exists — preserve
		}
		if err := writeToFile(f.path, f.content); err != nil {
			return err
		}
	}
	return nil
}
