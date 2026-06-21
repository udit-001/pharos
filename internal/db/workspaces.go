package db

import (
	"fmt"
	"time"
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
	// Enrich with counts — 3 grouped queries instead of 3N per-workspace
	// queries (the previous N+1).
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
	for i, w := range ws {
		ws[i].LessonCount = lessonCounts[w.ID]
		ws[i].RecordCount = recordCounts[w.ID]
		ws[i].RefCount = refCounts[w.ID]
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
	w.LessonCount = s.lessonCount(w.ID)
	w.RecordCount = s.recordCount(w.ID)
	w.RefCount = s.refCount(w.ID)
	return w, nil
}

// GetWorkspaceByName returns a workspace by its name.
func (s *Store) GetWorkspaceByName(name string) (Workspace, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM workspaces WHERE name = ?", wsColumns), name)
	w, err := scanWorkspace(row)
	if err != nil {
		return w, err
	}
	w.LessonCount = s.lessonCount(w.ID)
	w.RecordCount = s.recordCount(w.ID)
	w.RefCount = s.refCount(w.ID)
	return w, nil
}

// AddWorkspace creates a new workspace.
func (s *Store) AddWorkspace(w Workspace) (Workspace, error) {
	now := time.Now().UTC().Format(time.RFC3339)
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
	_, err := s.db.Exec(fmt.Sprintf("UPDATE workspaces SET %s = ?, last_studied = datetime('now') WHERE id = ?", col), seq, id)
	return err
}

// TouchWorkspace updates last_studied timestamp.
func (s *Store) TouchWorkspace(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
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

func (s *Store) lessonCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM lessons WHERE workspace_id = ?", workspaceID)
	return count
}

func (s *Store) recordCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM learning_records WHERE workspace_id = ?", workspaceID)
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
