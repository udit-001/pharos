package db

import (
	"fmt"
	"sort"
)

const quizColumns = `id, workspace_id, title, slug, description, items, lesson_seq, created_at, updated_at`

const quizColumnsQualified = `quizzes.id, quizzes.workspace_id, quizzes.title, quizzes.slug, quizzes.description, quizzes.items, quizzes.lesson_seq, quizzes.created_at, quizzes.updated_at`

func scanQuiz(row interface{ Scan(...any) error }) (Quiz, error) {
	var q Quiz
	var lessonSeq *int64
	err := row.Scan(&q.ID, &q.WorkspaceID, &q.Title, &q.Slug, &q.Description, &q.Items, &lessonSeq, &q.CreatedAt, &q.UpdatedAt)
	if lessonSeq != nil {
		s := int(*lessonSeq)
		q.LessonSeq = &s
	}
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

// currentQuestionIDs resolves a quiz's item slugs to their IDs. Scores count
// only questions still in the quiz, so removed items don't inflate them.
// Shared by GetQuizScores (best) and GetQuizAttemptHistory (trend).
func (w *WorkspaceStore) currentQuestionIDs(items []string) map[int64]bool {
	currentQ := make(map[int64]bool, len(items))
	for _, slug := range items {
		if q, err := w.GetQuestionBySlug(slug); err == nil {
			currentQ[q.ID] = true
		}
	}
	return currentQ
}

// scoreAttemptCurrent scores one completed attempt against the quiz's current
// questions: correct = unique correct answers (latest per question wins) for
// questions still in the quiz. This is the single per-attempt scoring site —
// GetQuizScores takes the max of these for BestScore, GetQuizAttemptHistory
// returns them as a series. ScoreAttempt (historical, all answers) is a
// different concern used by the review page.
func (w *WorkspaceStore) scoreAttemptCurrent(quizAttemptID int64, currentQ map[int64]bool) int {
	answers, _ := w.GetAttempts(quizAttemptID)
	latestByQ := make(map[int64]Attempt)
	for _, ans := range answers {
		if currentQ[ans.QuestionID] {
			latestByQ[ans.QuestionID] = ans
		}
	}
	correct := 0
	for _, ans := range latestByQ {
		if ans.Correct != nil && *ans.Correct {
			correct++
		}
	}
	return correct
}

// GetQuizScores returns every quiz in the workspace with its best completed-
// attempt score. BestScore is the highest correct count across completed
// attempts, counting only questions still in the quiz's item list (removed
// items don't inflate it); BestTotal is the current item count. Attempted is
// false when no completed attempt exists.
//
// This is the single source for quiz scoring: the dashboard library page and
// the `quiz list` / `quiz list --weak` CLI commands all consume it, so the
// per-quiz iterate-attempts-count-unique-correct-take-best logic lives here
// once rather than at each caller.
func (w *WorkspaceStore) GetQuizScores() ([]QuizScore, error) {
	quizzes, err := w.GetQuizzes()
	if err != nil {
		return nil, fmt.Errorf("get quizzes: %w", err)
	}
	out := make([]QuizScore, len(quizzes))
	for i, q := range quizzes {
		items, _ := q.ParseItems()
		total := len(items)
		score := QuizScore{Quiz: q, BestTotal: total}

		attempts, err := w.GetQuizAttempts(q.ID)
		if err == nil {
			currentQ := w.currentQuestionIDs(items)
			for _, a := range attempts {
				if a.Status != "completed" {
					continue
				}
				score.Attempted = true
				if correct := w.scoreAttemptCurrent(a.ID, currentQ); correct > score.BestScore {
					score.BestScore = correct
					score.BestTotal = total
				}
			}
		}
		out[i] = score
	}
	return out, nil
}

// GetQuizAttemptHistory returns a quiz's completed attempts in chronological
// order, each scored against the quiz's current questions — so the series
// reconciles with GetQuizScores' BestScore (which is the max of these). Use it
// to see whether accuracy is improving across retakes. In-progress and
// abandoned attempts are excluded (no score).
func (w *WorkspaceStore) GetQuizAttemptHistory(quizID int64) ([]QuizAttemptScore, error) {
	quiz, err := w.GetQuizByID(quizID)
	if err != nil {
		return nil, fmt.Errorf("get quiz: %w", err)
	}
	items, _ := quiz.ParseItems()
	total := len(items)
	currentQ := w.currentQuestionIDs(items)

	attempts, err := w.GetQuizAttempts(quizID)
	if err != nil {
		return nil, fmt.Errorf("get quiz attempts: %w", err)
	}
	var out []QuizAttemptScore
	for _, a := range attempts {
		if a.Status != "completed" {
			continue
		}
		out = append(out, QuizAttemptScore{
			QuizAttempt: a,
			Correct:     w.scoreAttemptCurrent(a.ID, currentQ),
			Total:       total,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CompletedAt < out[j].CompletedAt
	})
	return out, nil
}
