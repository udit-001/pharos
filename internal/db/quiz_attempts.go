package db

const quizAttemptColumns = `id, workspace_id, quiz_id, status, started_at, COALESCE(completed_at, '')`

func scanQuizAttempt(row interface{ Scan(...any) error }) (QuizAttempt, error) {
	var a QuizAttempt
	err := row.Scan(&a.ID, &a.WorkspaceID, &a.QuizID, &a.Status, &a.StartedAt, &a.CompletedAt)
	return a, err
}

const attemptColumns = `id, quiz_attempt_id, question_id, correct, response, COALESCE(latency_ms, ''), created_at`

func scanAttempt(row interface{ Scan(...any) error }) (Attempt, error) {
	var a Attempt
	var correct *int64
	var latency *int64
	err := row.Scan(&a.ID, &a.QuizAttemptID, &a.QuestionID, &correct, &a.Response, &latency, &a.CreatedAt)
	if correct != nil {
		c := *correct == 1
		a.Correct = &c
	}
	if latency != nil {
		l := int(*latency)
		a.LatencyMs = &l
	}
	return a, err
}

func scanAttempts(rows RowScanner) ([]Attempt, error) {
	return scanRows(rows, "attempt", scanAttempt)
}
