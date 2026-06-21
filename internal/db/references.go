package db

import (
	"fmt"
	"time"
)

const refColumns = `id, workspace_id, title, sequence_number, filename, path, summary, created_at`

func scanRef(row interface{ Scan(...any) error }) (Reference, error) {
	var r Reference
	err := row.Scan(&r.ID, &r.WorkspaceID, &r.Title, &r.SequenceNumber, &r.Filename, &r.Path, &r.Summary, &r.CreatedAt)
	return r, err
}

func scanRefs(rows RowScanner) ([]Reference, error) {
	return scanRows(rows, "ref", scanRef)
}

// GetReferences returns all references for a workspace, ordered by sequence.
func (s *Store) GetReferences(workspaceID int64) ([]Reference, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? ORDER BY sequence_number ASC", refColumns),
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// GetReference returns a single reference by ID.
func (s *Store) GetReference(id int64) (Reference, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM references_t WHERE id = ?", refColumns), id)
	return scanRef(row)
}

// AddReference creates a new reference record.
func (s *Store) AddReference(r Reference) (Reference, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var maxSeq int
	s.db.Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM references_t WHERE workspace_id = ?", r.WorkspaceID)
	r.SequenceNumber = maxSeq + 1

	result, err := s.db.Exec(
		`INSERT INTO references_t (workspace_id, title, sequence_number, filename, path, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.SequenceNumber, r.Filename, r.Path, r.Summary, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("add reference: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	return r, nil
}

// DeleteReference deletes a reference.
func (s *Store) DeleteReference(id int64) error {
	result, err := s.db.Exec("DELETE FROM references_t WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reference %d not found", id)
	}
	return nil
}

// SearchReferences performs full-text search across references.
func (s *Store) SearchReferences(query string, workspaceID int64) ([]Reference, error) {
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE id IN (SELECT rowid FROM refs_fts WHERE refs_fts MATCH ?) AND workspace_id = ? ORDER BY sequence_number ASC", refColumns),
		query, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

func (s *Store) refCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM references_t WHERE workspace_id = ?", workspaceID)
	return count
}
