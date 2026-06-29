package db

const quizColumns = `id, workspace_id, title, slug, description, items, created_at, updated_at`

const quizColumnsQualified = `quizzes.id, quizzes.workspace_id, quizzes.title, quizzes.slug, quizzes.description, quizzes.items, quizzes.created_at, quizzes.updated_at`

func scanQuiz(row interface{ Scan(...any) error }) (Quiz, error) {
	var q Quiz
	err := row.Scan(&q.ID, &q.WorkspaceID, &q.Title, &q.Slug, &q.Description, &q.Items, &q.CreatedAt, &q.UpdatedAt)
	return q, err
}

func scanQuizzes(rows RowScanner) ([]Quiz, error) {
	return scanRows(rows, "quiz", scanQuiz)
}

// quizCount returns the number of quizzes in a workspace.
func (s *Store) quizCount(workspaceID int64) int {
	var count int
	s.db.Get(&count, "SELECT COUNT(*) FROM quizzes WHERE workspace_id = ?", workspaceID)
	return count
}
