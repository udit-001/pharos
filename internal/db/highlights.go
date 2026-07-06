package db

const highlightColumns = `id, workspace_id, doc_type, doc_id, color, note_text, anchor_data, created_at, updated_at`

const highlightColumnsQualified = `highlights.id, highlights.workspace_id, highlights.doc_type, highlights.doc_id, highlights.color, highlights.note_text, highlights.anchor_data, highlights.created_at, highlights.updated_at`

func scanHighlight(row interface{ Scan(...any) error }) (Highlight, error) {
	var h Highlight
	err := row.Scan(&h.ID, &h.WorkspaceID, &h.DocType, &h.DocID, &h.Color, &h.NoteText, &h.AnchorData, &h.CreatedAt, &h.UpdatedAt)
	return h, err
}

func scanHighlights(rows RowScanner) ([]Highlight, error) {
	return scanRows(rows, "highlight", scanHighlight)
}
