package db

const questionColumns = `id, workspace_id, title, slug, mode, config, created_at, updated_at`

const questionColumnsQualified = `questions.id, questions.workspace_id, questions.title, questions.slug, questions.mode, questions.config, questions.created_at, questions.updated_at`

func scanQuestion(row interface{ Scan(...any) error }) (Question, error) {
	var q Question
	err := row.Scan(&q.ID, &q.WorkspaceID, &q.Title, &q.Slug, &q.Mode, &q.Config, &q.CreatedAt, &q.UpdatedAt)
	return q, err
}

func scanQuestions(rows RowScanner) ([]Question, error) {
	return scanRows(rows, "question", scanQuestion)
}
