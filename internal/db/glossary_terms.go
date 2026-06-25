package db

const glossaryTermColumns = `id, workspace_id, term, definition, category, avoid, created_at, updated_at`

func scanGlossaryTerm(row interface{ Scan(...any) error }) (GlossaryTerm, error) {
	var t GlossaryTerm
	err := row.Scan(&t.ID, &t.WorkspaceID, &t.Term, &t.Definition, &t.Category, &t.Avoid, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func scanGlossaryTerms(rows RowScanner) ([]GlossaryTerm, error) {
	return scanRows(rows, "glossary_term", scanGlossaryTerm)
}
