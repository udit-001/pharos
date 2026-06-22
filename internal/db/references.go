package db

const refColumns = `id, workspace_id, title, slug, filename, path, summary, created_at, updated_at`

func scanRef(row interface{ Scan(...any) error }) (Reference, error) {
	var r Reference
	err := row.Scan(&r.ID, &r.WorkspaceID, &r.Title, &r.Slug, &r.Filename, &r.Path, &r.Summary, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func scanRefs(rows RowScanner) ([]Reference, error) {
	return scanRows(rows, "ref", scanRef)
}

// RefCount returns the number of references in a workspace. Used by
// GetWorkspaces for count enrichment.
func (s *Store) refCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM references_t WHERE workspace_id = ?", workspaceID)
	return count
}
