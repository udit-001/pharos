package db

const recordColumns = `id, workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, COALESCE(body_text, ''), created_at, updated_at`

const recordColumnsQualified = `learning_records.id, learning_records.workspace_id, learning_records.title, learning_records.sequence_number, learning_records.filename, learning_records.path, learning_records.status, learning_records.superseded_by, learning_records.summary, COALESCE(learning_records.body_text, ''), learning_records.created_at, learning_records.updated_at`

func scanRecord(row interface{ Scan(...any) error }) (LearningRecord, error) {
	var r LearningRecord
	var supersededBy *int64
	err := row.Scan(&r.ID, &r.WorkspaceID, &r.Title, &r.SequenceNumber, &r.Filename, &r.Path, &r.Status, &supersededBy, &r.Summary, &r.BodyText, &r.CreatedAt, &r.UpdatedAt)
	if supersededBy != nil {
		r.SupersededBy = *supersededBy
	}
	return r, err
}

func scanRecords(rows RowScanner) ([]LearningRecord, error) {
	return scanRows(rows, "record", scanRecord)
}

// RecordCount returns the number of learning records in a workspace. Used by
// GetWorkspaces for count enrichment.
func (s *Store) recordCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM learning_records WHERE workspace_id = ?", workspaceID)
	return count
}
