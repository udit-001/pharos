package db

const lessonColumns = `id, workspace_id, title, sequence_number, filename, path, summary, COALESCE(body_text, ''), created_at, updated_at`

func scanLesson(row interface{ Scan(...any) error }) (Lesson, error) {
	var l Lesson
	err := row.Scan(&l.ID, &l.WorkspaceID, &l.Title, &l.SequenceNumber, &l.Filename, &l.Path, &l.Summary, &l.BodyText, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func scanLessons(rows RowScanner) ([]Lesson, error) {
	return scanRows(rows, "lesson", scanLesson)
}

// LessonCount returns the number of lessons in a workspace. Used by
// GetWorkspaces for count enrichment.
func (s *Store) lessonCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM lessons WHERE workspace_id = ?", workspaceID)
	return count
}
