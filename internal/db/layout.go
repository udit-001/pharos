package db

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Layout owns the on-disk directory structure of a workspace. One place
// defines where lessons, records, references, and assets live — callers
// ask it instead of reconstructing paths from convention.
type Layout struct {
	Root string // absolute path to the workspace directory
}

// NewLayout creates a Layout for the given workspace root path.
func NewLayout(root string) Layout {
	return Layout{Root: root}
}

// Subdirs returns the workspace subdirectory names in creation order.
func (l Layout) Subdirs() []string {
	return []string{"lessons", "learning-records", "reference", "questions", "assets", "sources"}
}

// LessonPath returns the absolute path for a lesson file.
func (l Layout) LessonPath(filename string) string {
	return filepath.Join(l.Root, "lessons", filename)
}

// RecordPath returns the absolute path for a learning record file.
func (l Layout) RecordPath(filename string) string {
	return filepath.Join(l.Root, "learning-records", filename)
}

// RefPath returns the absolute path for a reference file.
func (l Layout) RefPath(filename string) string {
	return filepath.Join(l.Root, "reference", filename)
}

// AssetPath returns the absolute path for an asset file.
func (l Layout) AssetPath(filename string) string {
	return filepath.Join(l.Root, "assets", filename)
}

// LessonRelPath returns the relative path for a lesson file (stored in DB).
func (l Layout) LessonRelPath(filename string) string {
	return filepath.Join("lessons", filename)
}

// RecordRelPath returns the relative path for a record file (stored in DB).
func (l Layout) RecordRelPath(filename string) string {
	return filepath.Join("learning-records", filename)
}

// RefRelPath returns the relative path for a reference file (stored in DB).
func (l Layout) RefRelPath(filename string) string {
	return filepath.Join("reference", filename)
}

// QuestionPath returns the absolute path for a question stimulus file.
func (l Layout) QuestionPath(filename string) string {
	return filepath.Join(l.Root, "questions", filename)
}

// QuestionRelPath returns the relative path for a question stimulus file (stored in DB).
func (l Layout) QuestionRelPath(filename string) string {
	return filepath.Join("questions", filename)
}

// SourcePath returns the absolute path for a source document file.
func (l Layout) SourcePath(filename string) string {
	return filepath.Join(l.Root, "sources", filename)
}

// SourceRelPath returns the relative path for a source document file (stored in DB).
func (l Layout) SourceRelPath(filename string) string {
	return filepath.Join("sources", filename)
}

// MissionPath returns the absolute path to MISSION.md.
func (l Layout) MissionPath() string {
	return filepath.Join(l.Root, "MISSION.md")
}

// ResourcesPath returns the absolute path to RESOURCES.md.
func (l Layout) ResourcesPath() string {
	return filepath.Join(l.Root, "RESOURCES.md")
}

// GlossaryPath returns the absolute path to GLOSSARY.md.
func (l Layout) GlossaryPath() string {
	return filepath.Join(l.Root, "GLOSSARY.md")
}

// NotesPath returns the absolute path to NOTES.md.
func (l Layout) NotesPath() string {
	return filepath.Join(l.Root, "NOTES.md")
}

// SafeJoin resolves filename into subdir inside the workspace root, rejecting
// traversal (.., absolute paths) and the directory itself. It is the single
// path guard for every user-supplied filename (assets, sources) — one place
// owns the rule so the copies can't drift.
func (l Layout) SafeJoin(subdir, filename string) (string, error) {
	dir := filepath.Join(l.Root, subdir)
	target := filepath.Join(dir, filename)
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(filename) {
		return "", fmt.Errorf("invalid path %q", filename)
	}
	return target, nil
}
