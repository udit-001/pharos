package db

const refColumns = `id, workspace_id, title, slug, filename, path, summary, COALESCE(body_text, ''), created_at, updated_at`

const refColumnsQualified = `references_t.id, references_t.workspace_id, references_t.title, references_t.slug, references_t.filename, references_t.path, references_t.summary, COALESCE(references_t.body_text, ''), references_t.created_at, references_t.updated_at`

func scanRef(row interface{ Scan(...any) error }) (Reference, error) {
	var r Reference
	err := row.Scan(&r.ID, &r.WorkspaceID, &r.Title, &r.Slug, &r.Filename, &r.Path, &r.Summary, &r.BodyText, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func scanRefs(rows RowScanner) ([]Reference, error) {
	return scanRows(rows, "ref", scanRef)
}
