package db

import (
	"fmt"
	"time"
)

const lessonColumns = `id, workspace_id, title, sequence_number, filename, path, summary, created_at`

func scanLesson(row interface{ Scan(...any) error }) (Lesson, error) {
	var l Lesson
	err := row.Scan(&l.ID, &l.WorkspaceID, &l.Title, &l.SequenceNumber, &l.Filename, &l.Path, &l.Summary, &l.CreatedAt)
	return l, err
}

func scanLessons(rows RowScanner) ([]Lesson, error) {
	return scanRows(rows, "lesson", scanLesson)
}

// GetLessons returns all lessons for a workspace, ordered by sequence number.
func (s *Store) GetLessons(workspaceID int64) ([]Lesson, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? ORDER BY sequence_number ASC", lessonColumns),
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// GetLesson returns a single lesson by ID.
func (s *Store) GetLesson(id int64) (Lesson, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM lessons WHERE id = ?", lessonColumns), id)
	return scanLesson(row)
}

// AddLesson creates a new lesson record.
func (s *Store) AddLesson(l Lesson) (Lesson, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Determine next sequence number
	var maxSeq int
	s.db.Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM lessons WHERE workspace_id = ?", l.WorkspaceID)
	l.SequenceNumber = maxSeq + 1

	result, err := s.db.Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		l.WorkspaceID, l.Title, l.SequenceNumber, l.Filename, l.Path, l.Summary, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("add lesson: %w", err)
	}
	id, _ := result.LastInsertId()
	l.ID = id
	l.CreatedAt = now
	return l, nil
}

// UpdateLesson updates a lesson's title and/or summary.
func (s *Store) UpdateLesson(id int64, title, summary string) error {
	_, err := s.db.Exec("UPDATE lessons SET title = ?, summary = ? WHERE id = ?", title, summary, id)
	return err
}

// DeleteLesson deletes a lesson.
func (s *Store) DeleteLesson(id int64) error {
	result, err := s.db.Exec("DELETE FROM lessons WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("lesson %d not found", id)
	}
	return nil
}

// SearchLessons performs full-text search across lessons.
func (s *Store) SearchLessons(query string, workspaceID int64) ([]Lesson, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE id IN (SELECT rowid FROM lessons_fts WHERE lessons_fts MATCH ?) AND workspace_id = ? ORDER BY sequence_number ASC", lessonColumns),
		query, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}
