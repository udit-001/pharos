package db

import (
	"errors"
	"fmt"
)

// ErrQuizNotFound is returned when a quiz slug doesn't resolve.
var ErrQuizNotFound = errors.New("quiz not found")

// ErrLessonNotFound is returned when a lesson sequence doesn't resolve.
var ErrLessonNotFound = errors.New("lesson not found")

// LessonContentStore encapsulates the relationships between lessons and
// their practice content (quizzes, and in future inline questions from
// LEARN-57). Methods validate their inputs and enforce invariants at the
// store seam.
type LessonContentStore struct {
	ws *WorkspaceStore
}

// LessonContent returns the LessonContentStore for this workspace.
func (w *WorkspaceStore) LessonContent() *LessonContentStore {
	return &LessonContentStore{ws: w}
}

// SetQuizLesson links a quiz to a lesson by slug, making the quiz practice
// that lesson's content. If the quiz is already linked to a different lesson,
// the link is overwritten. Validates both exist before writing. Unlike the old
// seq-based link, slug links survive lesson reorder.
func (l *LessonContentStore) SetQuizLesson(quizSlug string, lessonSlug string) error {
	quiz, err := l.ws.GetQuizBySlug(quizSlug)
	if err != nil {
		return fmt.Errorf("set quiz lesson: %w", ErrQuizNotFound)
	}
	if _, err := l.ws.GetLessonBySlug(lessonSlug); err != nil {
		return fmt.Errorf("set quiz lesson: lesson %q: %w", lessonSlug, ErrLessonNotFound)
	}
	now := nowTimestamp()
	_, err = l.ws.db().Exec(
		"UPDATE quizzes SET lesson_slug = ?, updated_at = ? WHERE id = ?",
		lessonSlug, now, quiz.ID,
	)
	if err != nil {
		return fmt.Errorf("set quiz lesson: %w", err)
	}
	return nil
}

// ClearQuizLesson removes the lesson link from a quiz. It is idempotent —
// no error if the quiz is already unlinked. Validates the quiz slug exists.
func (l *LessonContentStore) ClearQuizLesson(quizSlug string) error {
	quiz, err := l.ws.GetQuizBySlug(quizSlug)
	if err != nil {
		return fmt.Errorf("clear quiz lesson: %w", ErrQuizNotFound)
	}
	now := nowTimestamp()
	_, err = l.ws.db().Exec(
		"UPDATE quizzes SET lesson_slug = NULL, updated_at = ? WHERE id = ?",
		now, quiz.ID,
	)
	if err != nil {
		return fmt.Errorf("clear quiz lesson: %w", err)
	}
	return nil
}

// QuizzesForLesson returns the quizzes linked to a lesson by slug, ordered
// by title. Returns an empty slice if the lesson has no linked quizzes
// (no error — the caller may not know whether quizzes exist).
func (l *LessonContentStore) QuizzesForLesson(lessonSlug string) ([]Quiz, error) {
	rows, err := l.ws.db().Query(
		fmt.Sprintf("SELECT %s FROM quizzes WHERE workspace_id = ? AND lesson_slug = ? ORDER BY title ASC", quizColumns),
		l.ws.ws.ID, lessonSlug,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanQuizzes(rows)
}
