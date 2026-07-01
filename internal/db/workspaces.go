package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/udit-001/pharos/internal/urls"
)

const wsColumns = `id, name, topic, path, mission_why, created_at, last_studied, last_lesson_seq, last_record_seq, last_ref_seq`

func scanWorkspace(row interface{ Scan(...any) error }) (Workspace, error) {
	var w Workspace
	err := row.Scan(&w.ID, &w.Name, &w.Topic, &w.Path, &w.MissionWhy, &w.CreatedAt, &w.LastStudied,
		&w.LastLessonSeq, &w.LastRecordSeq, &w.LastRefSeq)
	return w, err
}

func scanWorkspaces(rows RowScanner) ([]Workspace, error) {
	return scanRows(rows, "workspace", scanWorkspace)
}

// GetWorkspaces returns all workspaces, newest first.
func (s *Store) GetWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT %s FROM workspaces ORDER BY last_studied DESC", wsColumns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ws, err := scanWorkspaces(rows)
	if err != nil {
		return nil, err
	}
	// Enrich with counts — grouped queries instead of N+1 per-workspace queries.
	lessonCounts, err := s.countByWorkspace("lessons")
	if err != nil {
		return nil, fmt.Errorf("count lessons: %w", err)
	}
	recordCounts, err := s.countByWorkspace("learning_records")
	if err != nil {
		return nil, fmt.Errorf("count records: %w", err)
	}
	refCounts, err := s.countByWorkspace("references_t")
	if err != nil {
		return nil, fmt.Errorf("count refs: %w", err)
	}
	quizCounts, err := s.countByWorkspace("quizzes")
	if err != nil {
		return nil, fmt.Errorf("count quizzes: %w", err)
	}
	for i, w := range ws {
		ws[i].LessonCount = lessonCounts[w.ID]
		ws[i].RecordCount = recordCounts[w.ID]
		ws[i].RefCount = refCounts[w.ID]
		ws[i].QuizCount = quizCounts[w.ID]
	}
	return ws, nil
}

// GetWorkspace returns a single workspace by ID.
func (s *Store) GetWorkspace(id int64) (Workspace, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM workspaces WHERE id = ?", wsColumns), id)
	w, err := scanWorkspace(row)
	if err != nil {
		return w, err
	}
	w.LessonCount = s.countByTable("lessons", w.ID)
	w.RecordCount = s.countByTable("learning_records", w.ID)
	w.RefCount = s.countByTable("references_t", w.ID)
	w.QuizCount = s.countByTable("quizzes", w.ID)
	return w, nil
}

// GetWorkspaceByName returns a workspace by its name.
func (s *Store) GetWorkspaceByName(name string) (Workspace, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM workspaces WHERE name = ?", wsColumns), name)
	w, err := scanWorkspace(row)
	if err != nil {
		return w, err
	}
	w.LessonCount = s.countByTable("lessons", w.ID)
	w.RecordCount = s.countByTable("learning_records", w.ID)
	w.RefCount = s.countByTable("references_t", w.ID)
	w.QuizCount = s.countByTable("quizzes", w.ID)
	return w, nil
}

// AddWorkspace creates a new workspace row only. It does not create the
// on-disk directory or seed templates — for that use CreateWorkspace, which
// owns the full row ⇔ dir tree invariant. Tests use AddWorkspace to set up a
// row without the filesystem; production code uses CreateWorkspace.
func (s *Store) AddWorkspace(w Workspace) (Workspace, error) {
	now := nowTimestamp()
	result, err := s.db.Exec(
		`INSERT INTO workspaces (name, topic, path, mission_why, created_at, last_studied)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		w.Name, w.Topic, w.Path, w.MissionWhy, now, now,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("add workspace: %w", err)
	}
	id, _ := result.LastInsertId()
	w.ID = id
	w.CreatedAt = now
	w.LastStudied = now
	return w, nil
}

// CreateWorkspace owns the full workspace lifecycle: it creates the directory
// tree (root + subdirs), seeds the default files (CSS/JS assets, MISSION/
// RESOURCES/NOTES templates), and inserts the row. The "row ⇔ dir tree"
// invariant lives here — a created workspace always has both. wsPath is
// supplied by the caller (the CLI knows the data dir / --dir override); the
// store owns the scaffold. Mirrors the WorkspaceStore.CreateLesson/Record
// shape for the workspace entity itself.
func (s *Store) CreateWorkspace(name, topic, wsPath string) (Workspace, error) {
	layout := NewLayout(wsPath)

	for _, sub := range layout.Subdirs() {
		if err := os.MkdirAll(filepath.Join(layout.Root, sub), 0o755); err != nil {
			return Workspace{}, fmt.Errorf("create %s directory: %w", sub, err)
		}
	}

	displayName := topic
	if displayName == "" {
		displayName = name
	}
	if err := seedWorkspaceDefaults(layout, displayName); err != nil {
		return Workspace{}, fmt.Errorf("seed workspace: %w", err)
	}

	w := Workspace{Name: name, Topic: topic, Path: wsPath}
	created, err := s.AddWorkspace(w)
	if err != nil {
		// Roll back the directory scaffold on DB failure so a retry
		// doesn't hit a duplicate-name error against an orphaned dir.
		_ = os.RemoveAll(wsPath)
		return Workspace{}, err
	}
	return created, nil
}

// DeleteWorkspaceByName removes a workspace's row (cascading to its lessons,
// records, and references), deletes its on-disk directory, and clears the
// current-workspace setting if it pointed at the deleted one. The inverse of
// CreateWorkspace — the row ⇔ dir tree invariant is torn down in one place.
// Confirmation prompting stays with the caller (a UI concern).
func (s *Store) DeleteWorkspaceByName(name string) error {
	w, err := s.GetWorkspaceByName(name)
	if err != nil {
		return fmt.Errorf("workspace %q not found: %w", name, err)
	}

	if err := s.DeleteWorkspace(w.ID); err != nil {
		return fmt.Errorf("delete workspace row: %w", err)
	}

	if w.Path != "" {
		if err := os.RemoveAll(w.Path); err != nil {
			return fmt.Errorf("remove workspace directory: %w", err)
		}
	}

	if current, _ := s.CurrentWorkspace(); current == w.Name {
		_ = s.SetCurrentWorkspace("")
	}
	return nil
}

// UpdateWorkspaceMission updates the mission_why field.
func (s *Store) UpdateWorkspaceMission(id int64, missionWhy string) error {
	_, err := s.db.Exec("UPDATE workspaces SET mission_why = ? WHERE id = ?", missionWhy, id)
	return err
}

// UpdateWorkspaceTopic updates the topic field.
func (s *Store) UpdateWorkspaceTopic(id int64, topic string) error {
	_, err := s.db.Exec("UPDATE workspaces SET topic = ? WHERE id = ?", topic, id)
	return err
}

// SetLastViewed records which item was last viewed in this workspace.
func (s *Store) SetLastViewed(id int64, itemType string, seq int) error {
	col := ""
	switch itemType {
	case "lesson":
		col = "last_lesson_seq"
	case "record":
		col = "last_record_seq"
	case "ref":
		col = "last_ref_seq"
	default:
		return fmt.Errorf("unknown item type: %s", itemType)
	}
	now := nowTimestamp()
	_, err := s.db.Exec(fmt.Sprintf("UPDATE workspaces SET %s = ?, last_studied = ? WHERE id = ?", col), seq, now, id)
	return err
}

// TouchWorkspace updates last_studied timestamp.
func (s *Store) TouchWorkspace(id int64) error {
	now := nowTimestamp()
	_, err := s.db.Exec("UPDATE workspaces SET last_studied = ? WHERE id = ?", now, id)
	return err
}

// DeleteWorkspace deletes a workspace.
func (s *Store) DeleteWorkspace(id int64) error {
	result, err := s.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workspace %d not found", id)
	}
	return nil
}

// countByTable returns the row count for one workspace in one table. Used by
// GetWorkspace and GetWorkspaceByName to enrich a single workspace. The batch
// path (many workspaces) uses countByWorkspace instead.
func (s *Store) countByTable(table string, workspaceID int64) int {
	var count int
	s.db.Get(&count, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE workspace_id = ?", table), workspaceID)
	return count
}

// countByWorkspace runs `SELECT workspace_id, COUNT(*) FROM <table> GROUP BY
// workspace_id` and returns a map. Used by GetWorkspaces to enrich N workspaces
// in 3 queries instead of 3N (the previous N+1).
func (s *Store) countByWorkspace(table string) (map[int64]int, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT workspace_id, COUNT(*) FROM %s GROUP BY workspace_id", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[int64]int)
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan %s count: %w", table, err)
		}
		counts[id] = n
	}
	return counts, rows.Err()
}

// WorkspaceCount returns total number of workspaces.
func (s *Store) WorkspaceCount() (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM workspaces")
	return count, err
}

// Stats holds aggregate counts across all workspaces.
type Stats struct {
	Workspaces int
	Lessons    int
	Records    int
	Refs       int
	Quizzes    int
}

// Totals sums counts across the given workspaces.
func Totals(ws []Workspace) Stats {
	var t Stats
	for _, w := range ws {
		t.Lessons += w.LessonCount
		t.Records += w.RecordCount
		t.Refs += w.RefCount
		t.Quizzes += w.QuizCount
	}
	t.Workspaces = len(ws)
	return t
}

// ContinueItem is the "continue where you left off" recommendation for the
// dashboard. URL is the navigational link; Label is the display text.
type ContinueItem struct {
	URL   string
	Label string
}

// ContinueItem derives the "continue where you left off" recommendation: the
// first workspace with a last-viewed lesson or reference. Returns nil if no
// workspace has any activity. The URL/label are built here so the handler is
// a thin caller.
//
// last_ref_seq stores a reference's row ID, not a sequence number — refs are
// slug-based (migration 00006 dropped sequence_number). The > 0 guard skips
// stale zero values from the pre-fix SetLastViewed("ref", 0) call.
func (s *Store) ContinueItem() (*ContinueItem, error) {
	ws, _ := s.GetWorkspaces()
	for _, w := range ws {
		if w.LastLessonSeq != nil && *w.LastLessonSeq > 0 {
			wsStore, err := s.Workspace(w.Name)
			if err != nil {
				continue
			}
			lessons, _ := wsStore.GetLessons()
			for _, l := range lessons {
				if l.SequenceNumber == *w.LastLessonSeq {
					return &ContinueItem{
						URL:   urls.Lesson(w.Name, l.SequenceNumber),
						Label: fmt.Sprintf("%s — Lesson: %s", w.Name, l.Title),
					}, nil
				}
			}
		} else if w.LastRefSeq != nil && *w.LastRefSeq > 0 {
			wsStore, err := s.Workspace(w.Name)
			if err != nil {
				continue
			}
			refs, _ := wsStore.GetRefs()
			for _, ref := range refs {
				if ref.ID == int64(*w.LastRefSeq) {
					return &ContinueItem{
						URL:   urls.Ref(w.Name, ref.Slug),
						Label: fmt.Sprintf("%s — Reference: %s", w.Name, ref.Title),
					}, nil
				}
			}
		}
	}
	return nil, nil
}
