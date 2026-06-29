package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/udit-001/pharos/internal/extract"
)

// WorkspaceStore is a scoped view over the database bound to a single workspace.
// Created via Store.Workspace(name) — the workspace is resolved once at the
// seam, so callers never thread workspaceID through every call.
//
// WorkspaceStore owns the SQL for all workspace-scoped item operations
// (lessons, records, references). The flat Store no longer exposes item
// methods — callers must go through this seam.
type WorkspaceStore struct {
	store *Store
	ws    Workspace
}

// Workspace returns the resolved workspace.
func (w *WorkspaceStore) Workspace() Workspace { return w.ws }

// Layout returns the on-disk layout for this workspace.
func (w *WorkspaceStore) Layout() Layout { return NewLayout(w.ws.Path) }

// db returns the underlying *sqlx.DB for direct query access.
func (w *WorkspaceStore) db() *sqlx.DB { return w.store.db }

// ── Lessons ──

// GetLessons returns all lessons in this workspace, ordered by sequence number.
func (w *WorkspaceStore) GetLessons() ([]Lesson, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? ORDER BY sequence_number ASC", lessonColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// GetLessonBySeq returns a single lesson by its sequence number, or an error if not found.
func (w *WorkspaceStore) GetLessonBySeq(seq int) (*Lesson, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? AND sequence_number = ?", lessonColumns),
		w.ws.ID, seq,
	)
	lesson, err := scanLesson(row)
	if err != nil {
		return nil, fmt.Errorf("lesson %d not found: %w", seq, err)
	}
	return &lesson, nil
}

// SearchLessons performs full-text search within this workspace.
func (w *WorkspaceStore) SearchLessons(query string) ([]Lesson, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Lesson{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons_fts JOIN lessons ON lessons.id = lessons_fts.rowid WHERE lessons_fts MATCH ? AND lessons.workspace_id = ? ORDER BY bm25(lessons_fts, %s), lessons.sequence_number ASC", lessonColumnsQualified, bm25Weights),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// Search returns results from all entity types in this workspace.
func (w *WorkspaceStore) Search(query string) ([]SearchResult, error) {
	var results []SearchResult

	lessons, err := w.SearchLessons(query)
	if err != nil {
		return nil, fmt.Errorf("search lessons: %w", err)
	}
	for _, l := range lessons {
		sr := SearchResult{
			Type: "lesson", Title: l.Title, Summary: l.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			SequenceNumber: l.SequenceNumber,
		}
		if sr.Summary == "" && l.BodyText != "" {
			sr.Snippet = truncateSnippet(stripLeadingTitle(l.BodyText, l.Title), 200)
		}
		results = append(results, sr)
	}

	recs, err := w.SearchRecords(query)
	if err != nil {
		return nil, fmt.Errorf("search records: %w", err)
	}
	for _, rec := range recs {
		sr := SearchResult{
			Type: "record", Title: rec.Title, Summary: rec.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			SequenceNumber: rec.SequenceNumber,
		}
		if sr.Summary == "" && rec.BodyText != "" {
			sr.Snippet = truncateSnippet(rec.BodyText, 200)
		}
		results = append(results, sr)
	}

	refs, err := w.SearchRefs(query)
	if err != nil {
		return nil, fmt.Errorf("search refs: %w", err)
	}
	for _, ref := range refs {
		slug := ref.Slug
		if slug == "" {
			slug = Slugify(ref.Title)
		}
		sr := SearchResult{
			Type: "ref", Title: ref.Title, Summary: ref.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			Slug: slug,
		}
		if sr.Summary == "" && ref.BodyText != "" {
			sr.Snippet = truncateSnippet(stripLeadingTitle(ref.BodyText, ref.Title), 200)
		}
		results = append(results, sr)
	}

	if results == nil {
		return []SearchResult{}, nil
	}
	return results, nil
}

// AddLesson creates a new lesson in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
//
// AddLesson is a low-level insert used by tests and internal wiring. Callers
// that need body_text indexing should use CreateLesson instead.
func (w *WorkspaceStore) AddLesson(l Lesson) (Lesson, error) {
	l.WorkspaceID = w.ws.ID
	now := nowTimestamp()

	// Determine next sequence number
	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM lessons WHERE workspace_id = ?", l.WorkspaceID)
	l.SequenceNumber = maxSeq + 1

	result, err := w.db().Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		l.WorkspaceID, l.Title, l.SequenceNumber, l.Filename, l.Path, l.Summary, now, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("add lesson: %w", err)
	}
	id, _ := result.LastInsertId()
	l.ID = id
	l.CreatedAt = now
	l.UpdatedAt = now
	return l, nil
}

// ── Learning records ──

// GetRecords returns all learning records in this workspace.
func (w *WorkspaceStore) GetRecords() ([]LearningRecord, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? ORDER BY sequence_number ASC", recordColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// GetRecordBySeq returns a single learning record by its sequence number, or an error if not found.
func (w *WorkspaceStore) GetRecordBySeq(seq int) (*LearningRecord, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? AND sequence_number = ?", recordColumns),
		w.ws.ID, seq,
	)
	record, err := scanRecord(row)
	if err != nil {
		return nil, fmt.Errorf("record %d not found: %w", seq, err)
	}
	return &record, nil
}

// SearchRecords performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRecords(query string) ([]LearningRecord, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []LearningRecord{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM records_fts JOIN learning_records ON learning_records.id = records_fts.rowid WHERE records_fts MATCH ? AND learning_records.workspace_id = ? ORDER BY bm25(records_fts, %s), learning_records.sequence_number ASC", recordColumnsQualified, bm25Weights),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// AddRecord creates a new learning record in this workspace. WorkspaceID is
// set automatically from the scoped workspace.
func (w *WorkspaceStore) AddRecord(r LearningRecord) (LearningRecord, error) {
	r.WorkspaceID = w.ws.ID
	now := nowTimestamp()

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM learning_records WHERE workspace_id = ?", r.WorkspaceID)
	r.SequenceNumber = maxSeq + 1

	if r.Status == "" {
		r.Status = "active"
	}

	var supersededBy interface{}
	if r.SupersededBy > 0 {
		supersededBy = r.SupersededBy
	}

	result, err := w.db().Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.SequenceNumber, r.Filename, r.Path, r.Status, supersededBy, r.Summary, now, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("add learning record: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	r.UpdatedAt = now
	return r, nil
}

// ── Questions ──

// GetQuestions returns all questions in this workspace, ordered by title.
func (w *WorkspaceStore) GetQuestions() ([]Question, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM questions WHERE workspace_id = ? ORDER BY title ASC", questionColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanQuestions(rows)
}

// GetQuestionBySlug returns a single question by its slug, or an error if not found.
func (w *WorkspaceStore) GetQuestionBySlug(slug string) (*Question, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM questions WHERE workspace_id = ? AND slug = ?", questionColumns),
		w.ws.ID, slug,
	)
	question, err := scanQuestion(row)
	if err != nil {
		return nil, fmt.Errorf("question %q not found: %w", slug, err)
	}
	return &question, nil
}

// AddQuestion creates a new question in this workspace. WorkspaceID is set
// automatically and the slug is derived from the title if empty. The caller
// is responsible for validating the config (see Question.ParseConfig).
func (w *WorkspaceStore) AddQuestion(q Question) (Question, error) {
	q.WorkspaceID = w.ws.ID
	now := nowTimestamp()

	if q.Slug == "" {
		q.Slug = Slugify(q.Title)
	}

	result, err := w.db().Exec(
		`INSERT INTO questions (workspace_id, title, slug, mode, config, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		q.WorkspaceID, q.Title, q.Slug, q.Mode, q.Config, now, now,
	)
	if err != nil {
		return Question{}, fmt.Errorf("add question: %w", err)
	}
	id, _ := result.LastInsertId()
	q.ID = id
	q.CreatedAt = now
	q.UpdatedAt = now
	return q, nil
}

// ── Quizzes ──

// GetQuizzes returns all quizzes in this workspace, ordered by title.
func (w *WorkspaceStore) GetQuizzes() ([]Quiz, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM quizzes WHERE workspace_id = ? ORDER BY title ASC", quizColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanQuizzes(rows)
}

// GetQuizBySlug returns a single quiz by its slug, or an error if not found.
func (w *WorkspaceStore) GetQuizBySlug(slug string) (*Quiz, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM quizzes WHERE workspace_id = ? AND slug = ?", quizColumns),
		w.ws.ID, slug,
	)
	quiz, err := scanQuiz(row)
	if err != nil {
		return nil, fmt.Errorf("quiz %q not found: %w", slug, err)
	}
	return &quiz, nil
}

// GetQuizByID returns a single quiz by its primary key.
func (w *WorkspaceStore) GetQuizByID(id int64) (*Quiz, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM quizzes WHERE workspace_id = ? AND id = ?", quizColumns),
		w.ws.ID, id,
	)
	quiz, err := scanQuiz(row)
	if err != nil {
		return nil, fmt.Errorf("quiz %d not found: %w", id, err)
	}
	return &quiz, nil
}

// AddQuiz creates a new quiz in this workspace. WorkspaceID is set
// automatically and the slug is derived from the title if empty.
func (w *WorkspaceStore) AddQuiz(q Quiz) (Quiz, error) {
	q.WorkspaceID = w.ws.ID
	now := nowTimestamp()

	if q.Slug == "" {
		q.Slug = Slugify(q.Title)
	}

	result, err := w.db().Exec(
		`INSERT INTO quizzes (workspace_id, title, slug, description, items, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		q.WorkspaceID, q.Title, q.Slug, q.Description, q.Items, now, now,
	)
	if err != nil {
		return Quiz{}, fmt.Errorf("add quiz: %w", err)
	}
	id, _ := result.LastInsertId()
	q.ID = id
	q.CreatedAt = now
	q.UpdatedAt = now
	return q, nil
}

// ── Quiz Attempts ──

// GetWeakQuestions returns questions sorted by accuracy ascending, using
// only completed quiz attempts. Questions with zero attempts (null accuracy)
// sort first. The limit caps the result count.
func (w *WorkspaceStore) GetWeakQuestions(limit int) ([]WeakQuestionResult, error) {
	// Fetch all questions for this workspace.
	questions, err := w.GetQuestions()
	if err != nil {
		return nil, err
	}

	// Fetch accuracy per question from completed attempts only.
	type acc struct {
		QuestionID int64
		Correct    int
		Total      int
	}
	var accs []acc
	rows, err := w.db().Query(
		`SELECT a.question_id,
		   SUM(CASE WHEN a.correct = 1 THEN 1 ELSE 0 END) AS correct,
		   COUNT(*) AS total
		 FROM attempts a
		 JOIN quiz_attempts qa ON a.quiz_attempt_id = qa.id
		 WHERE qa.workspace_id = ? AND qa.status = 'completed'
		 GROUP BY a.question_id`,
		w.ws.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("query question accuracy: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a acc
		if err := rows.Scan(&a.QuestionID, &a.Correct, &a.Total); err != nil {
			return nil, fmt.Errorf("scan accuracy: %w", err)
		}
		accs = append(accs, a)
	}

	accMap := map[int64]acc{}
	for _, a := range accs {
		accMap[a.QuestionID] = a
	}

	// Build results.
	out := make([]WeakQuestionResult, len(questions))
	for i, q := range questions {
		out[i] = WeakQuestionResult{Question: q}
		if a, ok := accMap[q.ID]; ok {
			out[i].Correct = a.Correct
			out[i].Total = a.Total
			out[i].HasData = true
		}
	}

	// Sort: null accuracy (HasData=false) first, then by accuracy ascending.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if !out[j-1].HasData && out[j].HasData {
				break
			}
			if out[j-1].HasData && out[j].HasData {
				prevAcc := float64(out[j-1].Correct) / float64(out[j-1].Total)
				curAcc := float64(out[j].Correct) / float64(out[j].Total)
				if prevAcc <= curAcc {
					break
				}
			}
			out[j-1], out[j] = out[j], out[j-1]
		}
	}

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ErrQuizHasInProgress signals that a quiz has in-progress attempts and
// cannot be revised until they complete or are abandoned.
var ErrQuizHasInProgress = errors.New("quiz has in-progress attempts")

// UpdateQuizItems updates a quiz's items JSON. Rejects if any in-progress
// quiz attempts exist for this quiz.
func (w *WorkspaceStore) UpdateQuizItems(slug string, items string) error {
	quiz, err := w.GetQuizBySlug(slug)
	if err != nil {
		return err
	}

	// Block if in-progress attempts exist.
	attempts, err := w.GetQuizAttempts(quiz.ID)
	if err != nil {
		return fmt.Errorf("check in-progress attempts: %w", err)
	}
	for _, a := range attempts {
		if a.Status == "in_progress" {
			return fmt.Errorf("cannot revise quiz %q: %w", slug, ErrQuizHasInProgress)
		}
	}

	now := nowTimestamp()
	_, err = w.db().Exec(
		"UPDATE quizzes SET items = ?, updated_at = ? WHERE id = ?",
		items, now, quiz.ID,
	)
	if err != nil {
		return fmt.Errorf("update quiz items: %w", err)
	}
	return nil
}

// ── Quiz Attempt lifecycle ──

// CreateQuizAttempt starts a new attempt for the given quiz. The attempt is
// created with status in_progress.
func (w *WorkspaceStore) CreateQuizAttempt(quizID int64) (QuizAttempt, error) {
	now := nowTimestamp()
	result, err := w.db().Exec(
		`INSERT INTO quiz_attempts (workspace_id, quiz_id, status, started_at)
		 VALUES (?, ?, 'in_progress', ?)`,
		w.ws.ID, quizID, now,
	)
	if err != nil {
		return QuizAttempt{}, fmt.Errorf("create quiz attempt: %w", err)
	}
	id, _ := result.LastInsertId()
	return QuizAttempt{
		ID: id, WorkspaceID: w.ws.ID, QuizID: quizID,
		Status: "in_progress", StartedAt: now,
	}, nil
}

// GetQuizAttempt fetches a single quiz attempt by ID.
func (w *WorkspaceStore) GetQuizAttempt(id int64) (*QuizAttempt, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM quiz_attempts WHERE id = ?", quizAttemptColumns),
		id,
	)
	a, err := scanQuizAttempt(row)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt %d not found: %w", id, err)
	}
	return &a, nil
}

// GetQuizAttempts returns all attempts for a quiz, newest first.
func (w *WorkspaceStore) GetQuizAttempts(quizID int64) ([]QuizAttempt, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM quiz_attempts WHERE quiz_id = ? ORDER BY started_at DESC", quizAttemptColumns),
		quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuizAttempt
	for rows.Next() {
		a, err := scanQuizAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan quiz attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAttempts returns all answers for a quiz attempt, ordered by creation.
func (w *WorkspaceStore) GetAttempts(quizAttemptID int64) ([]Attempt, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM attempts WHERE quiz_attempt_id = ? ORDER BY created_at ASC", attemptColumns),
		quizAttemptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAttempts(rows)
}

// SubmitAttempt records a single answer and returns the resulting Attempt
// row. Returns an error if the parent quiz attempt is not in_progress.
//
// For choice mode: response is the selected option index; the server grades
// it via Config.Grade() and clientCorrect is ignored.
// For recall mode: the client self-grades and passes clientCorrect; the
// server stores it as-is (never overrides the learner's self-assessment).
func (w *WorkspaceStore) SubmitAttempt(quizAttemptID, questionID int64, response string, latencyMs int, clientCorrect *bool) (Attempt, error) {
	// State machine guard: attempt must be in_progress.
	qa, err := w.GetQuizAttempt(quizAttemptID)
	if err != nil {
		return Attempt{}, err
	}
	if qa.Status != "in_progress" {
		return Attempt{}, fmt.Errorf("cannot submit to %s attempt", qa.Status)
	}

	// Resolve the question + config to determine grading mode.
	question, err := w.GetQuestionByID(questionID)
	if err != nil {
		return Attempt{}, err
	}
	cfg, err := question.ParseConfig()
	if err != nil {
		return Attempt{}, err
	}

	var correct bool
	switch cfg.Mode() {
	case "choice":
		// Server grades choice questions.
		correct, err = cfg.Grade(response)
		if err != nil {
			return Attempt{}, err
		}
	case "recall":
		// Client self-grades recall questions; server stores as-is.
		if clientCorrect == nil {
			return Attempt{}, fmt.Errorf("recall questions require a client_correct value")
		}
		correct = *clientCorrect
	default:
		return Attempt{}, fmt.Errorf("unknown question mode %q", cfg.Mode())
	}

	now := nowTimestamp()
	// Replace any existing answer for this question in this quiz attempt
	// so re-answering doesn't accumulate duplicate rows.
	w.db().Exec(`DELETE FROM attempts WHERE quiz_attempt_id = ? AND question_id = ?`, quizAttemptID, questionID)
	result, err := w.db().Exec(
		`INSERT INTO attempts (quiz_attempt_id, question_id, correct, response, latency_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		quizAttemptID, questionID, correct, response, latencyMs, now,
	)
	if err != nil {
		return Attempt{}, fmt.Errorf("insert attempt: %w", err)
	}
	id, _ := result.LastInsertId()
	return Attempt{
		ID: id, QuizAttemptID: quizAttemptID, QuestionID: questionID,
		Correct: &correct, Response: response, LatencyMs: &latencyMs,
		CreatedAt: now,
	}, nil
}

// CompleteQuizAttempt marks an attempt as completed. Enforces that the
// current status is in_progress.
func (w *WorkspaceStore) CompleteQuizAttempt(id int64) error {
	return w.transitionQuizAttempt(id, "in_progress", "completed")
}

// AbandonQuizAttempt marks an attempt as abandoned. Enforces that the
// current status is in_progress.
func (w *WorkspaceStore) AbandonQuizAttempt(id int64) error {
	return w.transitionQuizAttempt(id, "in_progress", "abandoned")
}

func (w *WorkspaceStore) transitionQuizAttempt(id int64, fromStatus, toStatus string) error {
	now := nowTimestamp()
	var completedAt interface{}
	if toStatus == "completed" {
		completedAt = now
	}
	res, err := w.db().Exec(
		`UPDATE quiz_attempts SET status = ?, completed_at = ? WHERE id = ? AND status = ?`,
		toStatus, completedAt, id, fromStatus,
	)
	if err != nil {
		return fmt.Errorf("transition quiz attempt to %s: %w", toStatus, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("quiz attempt %d not in %s state", id, fromStatus)
	}
	return nil
}

// ScoreAttempt returns (correct, total) for a quiz attempt via SQL aggregation.
func (w *WorkspaceStore) ScoreAttempt(quizAttemptID int64) (correct, total int) {
	// Deduplicate by question_id — take the latest answer per question.
	row := w.db().QueryRow(
		`SELECT COUNT(CASE WHEN correct = 1 THEN 1 END), COUNT(DISTINCT question_id)
		 FROM attempts a
		 WHERE quiz_attempt_id = ?
		   AND id = (SELECT MAX(id) FROM attempts WHERE quiz_attempt_id = a.quiz_attempt_id AND question_id = a.question_id)`,
		quizAttemptID,
	)
	_ = row.Scan(&correct, &total)
	return correct, total
}

// GetQuestionByID fetches a single question by its primary key.
func (w *WorkspaceStore) GetQuestionByID(id int64) (*Question, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM questions WHERE workspace_id = ? AND id = ?", questionColumns),
		w.ws.ID, id,
	)
	q, err := scanQuestion(row)
	if err != nil {
		return nil, fmt.Errorf("question %d not found: %w", id, err)
	}
	return &q, nil
}

// ── References ──

// GetRefs returns all references in this workspace.
func (w *WorkspaceStore) GetRefs() ([]Reference, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? ORDER BY title ASC", refColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// GetRefBySlug returns a single reference by its slug, or an error if not found.
func (w *WorkspaceStore) GetRefBySlug(slug string) (*Reference, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? AND slug = ?", refColumns),
		w.ws.ID, slug,
	)
	ref, err := scanRef(row)
	if err != nil {
		return nil, fmt.Errorf("reference %q not found: %w", slug, err)
	}
	return &ref, nil
}

// SearchRefs performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRefs(query string) ([]Reference, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Reference{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM refs_fts JOIN references_t ON references_t.id = refs_fts.rowid WHERE refs_fts MATCH ? AND references_t.workspace_id = ? ORDER BY bm25(refs_fts, %s), references_t.title ASC", refColumnsQualified, bm25Weights),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// AddRef creates a new reference in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
func (w *WorkspaceStore) AddRef(r Reference) (Reference, error) {
	r.WorkspaceID = w.ws.ID
	now := nowTimestamp()

	if r.Slug == "" {
		r.Slug = Slugify(r.Title)
	}

	result, err := w.db().Exec(
		`INSERT INTO references_t (workspace_id, title, slug, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.Slug, r.Filename, r.Path, r.Summary, now, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("add reference: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	r.UpdatedAt = now
	return r, nil
}

// ── Workspace-scoped mutations ──

// SetLastViewed records which item was last viewed in this workspace.
func (w *WorkspaceStore) SetLastViewed(itemType string, seq int) error {
	return w.store.SetLastViewed(w.ws.ID, itemType, seq)
}

// Touch updates the last_studied timestamp for this workspace.
func (w *WorkspaceStore) Touch() error {
	return w.store.TouchWorkspace(w.ws.ID)
}

// UpdateMission updates the mission_why field for this workspace.
func (w *WorkspaceStore) UpdateMission(missionWhy string) error {
	return w.store.UpdateWorkspaceMission(w.ws.ID, missionWhy)
}

// UpdateTopic updates the topic field for this workspace.
func (w *WorkspaceStore) UpdateTopic(topic string) error {
	return w.store.UpdateWorkspaceTopic(w.ws.ID, topic)
}

// ── Deep create/revise/supersede methods ──

// CreateLesson creates a new lesson: sequencing, slugify, filename, write file,
// DB row — all in one method. The CLI shrinks to parse-and-call.
func (w *WorkspaceStore) CreateLesson(title, bodyHTML string) (Lesson, error) {
	now := nowTimestamp()

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM lessons WHERE workspace_id = ?", w.ws.ID)
	seqNum := maxSeq + 1

	slug := Slugify(title)
	filename := fmt.Sprintf("%04d-%s.html", seqNum, slug)

	if err := writeToFile(w.Layout().LessonPath(filename), bodyHTML); err != nil {
		return Lesson{}, fmt.Errorf("write lesson file: %w", err)
	}

	bodyText := extract.FromHTML(bodyHTML)

	result, err := w.db().Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().LessonRelPath(filename), "", bodyText, now, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("insert lesson: %w", err)
	}
	id, _ := result.LastInsertId()

	return Lesson{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().LessonRelPath(filename),
		BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ReviseLesson overwrites a lesson's content in place. Sequence and filename
// are unchanged.
func (w *WorkspaceStore) ReviseLesson(seq int, bodyHTML string, title *string, summary *string) error {
	lessons, err := w.GetLessons()
	if err != nil {
		return fmt.Errorf("find lesson: %w", err)
	}
	var current *Lesson
	for i := range lessons {
		if lessons[i].SequenceNumber == seq {
			current = &lessons[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("lesson %d not found", seq)
	}

	if err := writeToFile(w.Layout().LessonPath(current.Filename), bodyHTML); err != nil {
		return fmt.Errorf("write lesson file: %w", err)
	}

	now := nowTimestamp()
	t := current.Title
	if title != nil {
		t = *title
	}
	s := current.Summary
	if summary != nil {
		s = *summary
	}
	bodyText := extract.FromHTML(bodyHTML)
	_, err = w.db().Exec("UPDATE lessons SET title = ?, summary = ?, body_text = ?, updated_at = ? WHERE id = ?", t, s, bodyText, now, current.ID)
	return err
}

// CreateRecord creates a new learning record: sequencing, slugify, filename,
// write file, DB row — all in one method.
func (w *WorkspaceStore) CreateRecord(title, bodyMD, summary string) (LearningRecord, error) {
	now := nowTimestamp()

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM learning_records WHERE workspace_id = ?", w.ws.ID)
	seqNum := maxSeq + 1

	slug := Slugify(title)
	filename := fmt.Sprintf("%04d-%s.md", seqNum, slug)

	if err := writeToFile(w.Layout().RecordPath(filename), bodyMD); err != nil {
		return LearningRecord{}, fmt.Errorf("write record file: %w", err)
	}

	bodyText := extract.FromMarkdown(bodyMD)

	var supersededBy interface{}
	result, err := w.db().Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().RecordRelPath(filename), supersededBy, summary, bodyText, now, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("insert record: %w", err)
	}
	id, _ := result.LastInsertId()

	return LearningRecord{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().RecordRelPath(filename),
		Status: "active", Summary: summary, BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// SupersedeRecord atomically creates a new record and marks the old one as
// superseded. Returns the new record.
func (w *WorkspaceStore) SupersedeRecord(seq int, title, bodyMD, summary string) (LearningRecord, LearningRecord, error) {
	records, err := w.GetRecords()
	if err != nil {
		return LearningRecord{}, LearningRecord{}, fmt.Errorf("find old record: %w", err)
	}
	var old *LearningRecord
	for i := range records {
		if records[i].SequenceNumber == seq {
			old = &records[i]
			break
		}
	}
	if old == nil {
		return LearningRecord{}, LearningRecord{}, fmt.Errorf("record %d not found", seq)
	}

	created, err := w.CreateRecord(title, bodyMD, summary)
	if err != nil {
		return LearningRecord{}, LearningRecord{}, err
	}

	now := nowTimestamp()
	_, err = w.db().Exec("UPDATE learning_records SET status = 'superseded', superseded_by = ?, updated_at = ? WHERE id = ?", created.ID, now, old.ID)
	if err != nil {
		return created, LearningRecord{}, fmt.Errorf("supersede old record: %w", err)
	}

	old.Status = "superseded"
	old.SupersededBy = created.ID
	old.UpdatedAt = now

	return created, *old, nil
}

// ErrRefSlugExists signals that a reference with the given slug already
// exists in the workspace. Callers (e.g. the CLI) wrap it into a user-facing
// message that hints at the revise command.
var ErrRefSlugExists = errors.New("reference slug already exists")

// CreateRef creates a new reference: slug-based filename, write file, DB row.
func (w *WorkspaceStore) CreateRef(title, bodyHTML string) (Reference, error) {
	now := nowTimestamp()

	slug := Slugify(title)
	filename := slug + ".html"

	// Check for duplicate slug
	existing, _ := w.GetRefs()
	for _, r := range existing {
		if r.Slug == slug {
			return Reference{}, fmt.Errorf("reference with slug %q already exists: %w", slug, ErrRefSlugExists)
		}
	}

	if err := writeToFile(w.Layout().RefPath(filename), bodyHTML); err != nil {
		return Reference{}, fmt.Errorf("write reference file: %w", err)
	}

	bodyText := extract.FromHTML(bodyHTML)

	result, err := w.db().Exec(
		`INSERT INTO references_t (workspace_id, title, slug, filename, path, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, slug, filename, w.Layout().RefRelPath(filename), "", bodyText, now, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("insert reference: %w", err)
	}
	id, _ := result.LastInsertId()

	return Reference{
		ID: id, WorkspaceID: w.ws.ID, Title: title, Slug: slug,
		Filename: filename, Path: w.Layout().RefRelPath(filename),
		BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ReviseRef overwrites a reference's content in place. Slug is unchanged.
func (w *WorkspaceStore) ReviseRef(slug, bodyHTML string, title *string, summary *string) error {
	refs, err := w.GetRefs()
	if err != nil {
		return fmt.Errorf("find reference: %w", err)
	}
	var current *Reference
	for i := range refs {
		if refs[i].Slug == slug {
			current = &refs[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("reference %q not found", slug)
	}

	if err := writeToFile(w.Layout().RefPath(current.Filename), bodyHTML); err != nil {
		return fmt.Errorf("write reference file: %w", err)
	}

	now := nowTimestamp()
	t := current.Title
	if title != nil {
		t = *title
	}
	s := current.Summary
	if summary != nil {
		s = *summary
	}
	bodyText := extract.FromHTML(bodyHTML)
	_, err = w.db().Exec("UPDATE references_t SET title = ?, summary = ?, body_text = ?, updated_at = ? WHERE id = ?", t, s, bodyText, now, current.ID)
	return err
}

// ── Glossary Terms ──

// GetGlossaryTerms returns all glossary terms in this workspace, ordered by
// category (uncategorised last) then term alphabetically.
func (w *WorkspaceStore) GetGlossaryTerms() ([]GlossaryTerm, error) {
	rows, err := w.db().Query(
		fmt.Sprintf(`SELECT %s FROM glossary_terms WHERE workspace_id = ?
			ORDER BY CASE WHEN category IS NULL OR category = '' THEN 1 ELSE 0 END,
			category COLLATE NOCASE ASC, term COLLATE NOCASE ASC`, glossaryTermColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGlossaryTerms(rows)
}

// AddGlossaryTerm inserts or updates a single glossary term for the workspace.
// If a term with the same name (case-insensitive) already exists, it is updated.
// Empty category/avoid strings are treated as "leave unchanged" on update —
// to clear a category or avoid, delete the term and re-add it.
func (w *WorkspaceStore) AddGlossaryTerm(term, definition, category, avoid string) error {
	term = strings.TrimSpace(term)
	definition = strings.TrimSpace(definition)
	category = strings.TrimSpace(category)
	avoid = strings.TrimSpace(avoid)
	if term == "" {
		return fmt.Errorf("term must not be empty")
	}
	if definition == "" {
		return fmt.Errorf("definition must not be empty")
	}
	now := nowTimestamp()
	_, err := w.db().Exec(
		`INSERT INTO glossary_terms (workspace_id, term, definition, category, avoid, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id, term) DO UPDATE SET
			definition = excluded.definition,
			category = COALESCE(NULLIF(excluded.category, ''), glossary_terms.category),
			avoid = COALESCE(NULLIF(excluded.avoid, ''), glossary_terms.avoid),
			updated_at = excluded.updated_at`,
		w.ws.ID, term, definition, category, avoid, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert term %q: %w", term, err)
	}
	return nil
}

// DeleteGlossaryTerm removes a glossary term by name (case-insensitive).
// Returns nil if the term doesn't exist — delete is idempotent.
func (w *WorkspaceStore) DeleteGlossaryTerm(term string) error {
	term = strings.TrimSpace(term)
	if term == "" {
		return fmt.Errorf("term must not be empty")
	}
	_, err := w.db().Exec(
		"DELETE FROM glossary_terms WHERE workspace_id = ? AND term = ? COLLATE NOCASE",
		w.ws.ID, term,
	)
	if err != nil {
		return fmt.Errorf("delete term %q: %w", term, err)
	}
	return nil
}

// CreateAsset writes a file to the workspace's assets directory.
func (w *WorkspaceStore) CreateAsset(filename, content string) error {
	return writeToFile(w.Layout().AssetPath(filename), content)
}

// ListAssets returns the filenames in the workspace's assets directory.
func (w *WorkspaceStore) ListAssets() ([]string, error) {
	entries, err := readDirNames(filepath.Join(w.ws.Path, "assets"))
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ── Construction ──

// Workspace returns a WorkspaceStore scoped to the named workspace. The
// workspace is resolved (name→ID) once here; subsequent calls need not
// pass the ID. Returns an error if the workspace does not exist.
func (s *Store) Workspace(name string) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("workspace %q not found: %w", name, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}

// WorkspaceByID returns a WorkspaceStore scoped to the workspace with the
// given ID. Used when the ID is already known.
func (s *Store) WorkspaceByID(id int64) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("workspace %d not found: %w", id, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}

// QuizAttemptWorkspace resolves the workspace that owns a quiz attempt and
// returns a scoped WorkspaceStore. Used by API handlers that receive only
// an attempt ID (not a workspace name from the URL).
func (s *Store) QuizAttemptWorkspace(attemptID int64) (*WorkspaceStore, error) {
	var wsID int64
	if err := s.db.Get(&wsID, "SELECT workspace_id FROM quiz_attempts WHERE id = ?", attemptID); err != nil {
		return nil, fmt.Errorf("quiz attempt %d not found: %w", attemptID, err)
	}
	return s.WorkspaceByID(wsID)
}

// QuizDashboardData is the cross-workspace quiz summary for the dashboard.
type QuizDashboardData struct {
	RecentCompleted *CompletedQuizSummary
	InProgress      []InProgressSummary
}

type CompletedQuizSummary struct {
	WorkspaceName string
	QuizSlug      string
	QuizTitle     string
	AttemptID     int64
	Score         int
	Total         int
}

type InProgressSummary struct {
	WorkspaceName string
	QuizSlug      string
	QuizTitle     string
	AttemptID     int64
}

// GetQuizDashboardData returns the latest completed quiz across all
// workspaces and any in-progress attempts. This is a cross-workspace
// query on Store (not WorkspaceStore).
func (s *Store) GetQuizDashboardData() (QuizDashboardData, error) {
	var data QuizDashboardData

	// Latest completed quiz attempt across all workspaces.
	var rc struct {
		AttemptID    int64
		WorkspaceID  int64
		QuizID       int64
		CompletedAt  string
	}
	row := s.db.QueryRow(
		`SELECT qa.id, qa.workspace_id, qa.quiz_id, qa.completed_at
		 FROM quiz_attempts qa
		 WHERE qa.status = 'completed'
		 ORDER BY qa.completed_at DESC
		 LIMIT 1`,
	)
	if err := row.Scan(&rc.AttemptID, &rc.WorkspaceID, &rc.QuizID, &rc.CompletedAt); err == nil {
		wsStore, err := s.WorkspaceByID(rc.WorkspaceID)
		if err == nil {
			ws := wsStore.Workspace()
			quiz, err := wsStore.GetQuizByID(rc.QuizID)
			if err == nil {
				correct, total := wsStore.ScoreAttempt(rc.AttemptID)
			data.RecentCompleted = &CompletedQuizSummary{
				WorkspaceName: ws.Name,
				QuizSlug:      quiz.Slug,
				QuizTitle:     quiz.Title,
				AttemptID:     rc.AttemptID,
				Score:         correct,
				Total:         total,
			}
			}
		}
	}

	// In-progress attempts across all workspaces.
	rows, err := s.db.Query(
		`SELECT qa.id, qa.workspace_id, qa.quiz_id
		 FROM quiz_attempts qa
		 WHERE qa.status = 'in_progress'
		 ORDER BY qa.started_at DESC`,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ip struct {
				AttemptID   int64
				WorkspaceID int64
				QuizID      int64
			}
			if err := rows.Scan(&ip.AttemptID, &ip.WorkspaceID, &ip.QuizID); err != nil {
				continue
			}
			wsStore, err := s.WorkspaceByID(ip.WorkspaceID)
			if err != nil {
				continue
			}
			ws := wsStore.Workspace()
			quiz, err := wsStore.GetQuizByID(ip.QuizID)
			if err != nil {
				continue
			}
			data.InProgress = append(data.InProgress, InProgressSummary{
				WorkspaceName: ws.Name,
				QuizSlug:      quiz.Slug,
				QuizTitle:     quiz.Title,
				AttemptID:     ip.AttemptID,
			})
		}
	}

	return data, nil
}

// truncateSnippet returns a short preview of text, breaking at a word boundary.
// Used to show body content previews in search results when Summary is empty.
// stripLeadingTitle removes the first line of bodyText if it matches the
// entity's title, avoiding redundant text in search snippets.
func stripLeadingTitle(bodyText, title string) string {
	bodyText = strings.TrimSpace(bodyText)
	if title == "" {
		return bodyText
	}
	if strings.HasPrefix(bodyText, title) {
		rest := strings.TrimSpace(bodyText[len(title):])
		return rest
	}
	return bodyText
}

func truncateSnippet(s string, maxLen int) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	cut := strings.LastIndex(trimmed[:maxLen], " ")
	if cut < 1 {
		cut = maxLen
	}
	return strings.TrimSpace(trimmed[:cut]) + "..."
}

// IndexLessons reads all lessons with empty body_text, extracts plain text
// from their HTML files on disk, and updates the DB so the FTS index captures
// lesson body content. Returns the number of lessons updated and any errors.
// If one file fails, processing continues with the remaining lessons.
func (w *WorkspaceStore) IndexLessons() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? AND body_text = ''", lessonColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	lessons, err := scanLessons(rows)
	if err != nil {
		return 0, fmt.Errorf("scan lessons: %w", err)
	}

	layout := w.Layout()
	return indexItems(lessons,
		func(l Lesson) error {
			data, err := os.ReadFile(layout.LessonPath(l.Filename))
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			bodyText := extract.FromHTML(string(data))
			if _, err := w.db().Exec("UPDATE lessons SET body_text = ? WHERE id = ?", bodyText, l.ID); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			return nil
		},
		func(l Lesson) string { return fmt.Sprintf("%d (%s)", l.SequenceNumber, l.Filename) },
		"lesson",
	)
}

// IndexRefs reads all references with empty body_text, extracts plain text
// from their HTML files on disk, and updates the DB so the FTS index captures
// reference body content. Returns the number of references updated and any
// errors. If one file fails, processing continues with the remaining refs.
func (w *WorkspaceStore) IndexRefs() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? AND body_text = ''", refColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query references: %w", err)
	}
	defer rows.Close()

	refs, err := scanRefs(rows)
	if err != nil {
		return 0, fmt.Errorf("scan references: %w", err)
	}

	layout := w.Layout()
	return indexItems(refs,
		func(r Reference) error {
			data, err := os.ReadFile(layout.RefPath(r.Filename))
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			bodyText := extract.FromHTML(string(data))
			if _, err := w.db().Exec("UPDATE references_t SET body_text = ? WHERE id = ?", bodyText, r.ID); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			return nil
		},
		func(r Reference) string { return fmt.Sprintf("%s (%s)", r.Slug, r.Filename) },
		"ref",
	)
}

// IndexRecords reads all learning records with empty body_text, extracts plain
// text from their markdown files on disk, and updates the DB so the FTS index
// captures record body content. Returns the number of records updated and any
// errors. If one file fails, processing continues with the remaining records.
func (w *WorkspaceStore) IndexRecords() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? AND body_text = ''", recordColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query records: %w", err)
	}
	defer rows.Close()

	recs, err := scanRecords(rows)
	if err != nil {
		return 0, fmt.Errorf("scan records: %w", err)
	}

	layout := w.Layout()
	return indexItems(recs,
		func(r LearningRecord) error {
			data, err := os.ReadFile(layout.RecordPath(r.Filename))
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			bodyText := extract.FromMarkdown(string(data))
			if _, err := w.db().Exec("UPDATE learning_records SET body_text = ? WHERE id = ?", bodyText, r.ID); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			return nil
		},
		func(r LearningRecord) string { return fmt.Sprintf("%d (%s)", r.SequenceNumber, r.Filename) },
		"record",
	)
}
