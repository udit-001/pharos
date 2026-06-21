package db

import (
	"fmt"
	"time"
)

const recordColumns = `id, workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, created_at`

func scanRecord(row interface{ Scan(...any) error }) (LearningRecord, error) {
	var r LearningRecord
	var supersededBy *int64
	err := row.Scan(&r.ID, &r.WorkspaceID, &r.Title, &r.SequenceNumber, &r.Filename, &r.Path, &r.Status, &supersededBy, &r.Summary, &r.CreatedAt)
	if supersededBy != nil {
		r.SupersededBy = *supersededBy
	}
	return r, err
}

func scanRecords(rows RowScanner) ([]LearningRecord, error) {
	return scanRows(rows, "record", scanRecord)
}

// GetLearningRecords returns all learning records for a workspace.
func (s *Store) GetLearningRecords(workspaceID int64) ([]LearningRecord, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? ORDER BY sequence_number ASC", recordColumns),
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// GetLearningRecord returns a single record by ID.
func (s *Store) GetLearningRecord(id int64) (LearningRecord, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM learning_records WHERE id = ?", recordColumns), id)
	return scanRecord(row)
}

// AddLearningRecord creates a new learning record.
func (s *Store) AddLearningRecord(r LearningRecord) (LearningRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var maxSeq int
	s.db.Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM learning_records WHERE workspace_id = ?", r.WorkspaceID)
	r.SequenceNumber = maxSeq + 1

	if r.Status == "" {
		r.Status = "active"
	}

	var supersededBy interface{}
	if r.SupersededBy > 0 {
		supersededBy = r.SupersededBy
	}

	result, err := s.db.Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.SequenceNumber, r.Filename, r.Path, r.Status, supersededBy, r.Summary, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("add learning record: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	return r, nil
}

// SupersedeLearningRecord marks a record as superseded by another.
func (s *Store) SupersedeLearningRecord(id int64, supersededBy int64) error {
	_, err := s.db.Exec("UPDATE learning_records SET status = 'superseded', superseded_by = ? WHERE id = ?", supersededBy, id)
	return err
}

// DeleteLearningRecord deletes a learning record.
func (s *Store) DeleteLearningRecord(id int64) error {
	result, err := s.db.Exec("DELETE FROM learning_records WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("learning record %d not found", id)
	}
	return nil
}

// SearchLearningRecords performs full-text search across records.
func (s *Store) SearchLearningRecords(query string, workspaceID int64) ([]LearningRecord, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE id IN (SELECT rowid FROM records_fts WHERE records_fts MATCH ?) AND workspace_id = ? ORDER BY sequence_number ASC", recordColumns),
		query, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}
