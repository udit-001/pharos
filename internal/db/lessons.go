package db

const lessonColumns = `id, workspace_id, title, sequence_number, filename, path, summary, COALESCE(body_text, ''), created_at, updated_at`

const lessonColumnsQualified = `lessons.id, lessons.workspace_id, lessons.title, lessons.sequence_number, lessons.filename, lessons.path, lessons.summary, COALESCE(lessons.body_text, ''), lessons.created_at, lessons.updated_at`

func scanLesson(row interface{ Scan(...any) error }) (Lesson, error) {
	var l Lesson
	err := row.Scan(&l.ID, &l.WorkspaceID, &l.Title, &l.SequenceNumber, &l.Filename, &l.Path, &l.Summary, &l.BodyText, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func scanLessons(rows RowScanner) ([]Lesson, error) {
	return scanRows(rows, "lesson", scanLesson)
}
