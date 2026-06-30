package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// seedWorkspace creates a workspace and returns its WorkspaceStore for
// seeding items. This replaces direct store.AddLesson/AddLearningRecord/
// AddReference calls that used the now-removed flat Store item-methods.
func seedWorkspace(t *testing.T, store *Store, name string) *WorkspaceStore {
	t.Helper()
	_, err := store.AddWorkspace(Workspace{Name: name, Path: "/tmp/" + name})
	if err != nil {
		t.Fatalf("seed workspace %s: %v", name, err)
	}
	wsStore, err := store.Workspace(name)
	if err != nil {
		t.Fatalf("get workspace %s: %v", name, err)
	}
	return wsStore
}

// TestWorkspaceStoreScoping proves the WorkspaceStore seam: workspace
// resolution happens once at construction, scoped methods need no ID.
func TestWorkspaceStoreScoping(t *testing.T) {
	store := newTestStore(t)

	// Seed two workspaces with lessons
	alpha := seedWorkspace(t, store, "alpha")
	if _, err := alpha.AddLesson(Lesson{Title: "alpha-1", Filename: "a1.html"}); err != nil {
		t.Fatal(err)
	}
	beta := seedWorkspace(t, store, "beta")
	if _, err := beta.AddLesson(Lesson{Title: "beta-1", Filename: "b1.html"}); err != nil {
		t.Fatal(err)
	}

	// Scope to alpha — no ID passed from here on
	ws, err := store.Workspace("alpha")
	if err != nil {
		t.Fatalf("scope alpha: %v", err)
	}
	if got := ws.Workspace().Name; got != "alpha" {
		t.Errorf("scoped workspace name = %q, want alpha", got)
	}

	lessons, err := ws.GetLessons()
	if err != nil {
		t.Fatalf("scoped GetLessons: %v", err)
	}
	if len(lessons) != 1 || lessons[0].Title != "alpha-1" {
		t.Errorf("scoped lessons = %+v, want [alpha-1]", lessons)
	}

	// AddLesson via the scoped store — WorkspaceID set automatically
	created, err := ws.AddLesson(Lesson{Title: "alpha-2", Filename: "a2.html"})
	if err != nil {
		t.Fatalf("scoped AddLesson: %v", err)
	}
	if created.WorkspaceID != alpha.Workspace().ID {
		t.Errorf("AddLesson WorkspaceID = %d, want %d (should be auto-set)", created.WorkspaceID, alpha.Workspace().ID)
	}
	if created.SequenceNumber != 2 {
		t.Errorf("AddLesson SequenceNumber = %d, want 2", created.SequenceNumber)
	}

	// SetLastViewed + Touch — scoped, no ID
	if err := ws.SetLastViewed("lesson", 2); err != nil {
		t.Fatalf("scoped SetLastViewed: %v", err)
	}
	if err := ws.Touch(); err != nil {
		t.Fatalf("scoped Touch: %v", err)
	}

	// Cross-check: beta still has only 1 lesson (alpha-2 didn't leak)
	betaLessons, _ := beta.GetLessons()
	if len(betaLessons) != 1 {
		t.Errorf("beta lessons = %d, want 1 (scoping leak?)", len(betaLessons))
	}
}

// TestWorkspaceUnknownName proves the seam errors cleanly on bad input.
func TestWorkspaceUnknownName(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Workspace("does-not-exist"); err == nil {
		t.Error("expected error for unknown workspace, got nil")
	}
}

// TestGetLessonBySeq proves GetLessonBySeq uses a WHERE clause (not
// fetch-all-and-loop): it finds the right lesson by seq, errors on not-found,
// and respects workspace scoping.
func TestGetLessonBySeq(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	beta := seedWorkspace(t, store, "beta")
	beta.AddLesson(Lesson{Title: "b1", Filename: "b1.html"})

	// Found
	got, err := alpha.GetLessonBySeq(2)
	if err != nil {
		t.Fatalf("GetLessonBySeq(2): %v", err)
	}
	if got.Title != "a2" {
		t.Errorf("got %q, want a2", got.Title)
	}

	// Not found in workspace
	if _, err := alpha.GetLessonBySeq(99); err == nil {
		t.Error("expected error for seq 99, got nil")
	}

	// Scoping: beta's seq 1 is "b1", not "a1"
	gotBeta, err := beta.GetLessonBySeq(1)
	if err != nil {
		t.Fatalf("beta GetLessonBySeq(1): %v", err)
	}
	if gotBeta.Title != "b1" {
		t.Errorf("beta seq 1 = %q, want b1", gotBeta.Title)
	}
}

// TestGetRecordBySeq proves GetRecordBySeq uses a WHERE clause.
func TestGetRecordBySeq(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md"})
	alpha.AddRecord(LearningRecord{Title: "r2", Filename: "r2.md"})

	got, err := alpha.GetRecordBySeq(2)
	if err != nil {
		t.Fatalf("GetRecordBySeq(2): %v", err)
	}
	if got.Title != "r2" {
		t.Errorf("got %q, want r2", got.Title)
	}

	if _, err := alpha.GetRecordBySeq(99); err == nil {
		t.Error("expected error for seq 99, got nil")
	}
}

// TestGetRefBySlug proves GetRefBySlug uses a WHERE clause.
func TestGetRefBySlug(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddRef(Reference{Title: "Joins", Slug: "joins", Filename: "joins.html"})
	alpha.AddRef(Reference{Title: "Indexes", Slug: "indexes", Filename: "indexes.html"})

	got, err := alpha.GetRefBySlug("indexes")
	if err != nil {
		t.Fatalf("GetRefBySlug(indexes): %v", err)
	}
	if got.Title != "Indexes" {
		t.Errorf("got %q, want Indexes", got.Title)
	}

	if _, err := alpha.GetRefBySlug("nonexistent"); err == nil {
		t.Error("expected error for nonexistent slug, got nil")
	}
}

func TestAddQuestionAndQuiz(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	// AddQuestion derives the slug and sets WorkspaceID/timestamps.
	q, err := alpha.AddQuestion(Question{
		Title:  "Strongest ASD Risk Gene",
		Mode:   "choice",
		Config: `{"options":["CHD8","FMR1"],"key":0}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if q.Slug != "strongest-asd-risk-gene" {
		t.Errorf("slug = %q, want strongest-asd-risk-gene", q.Slug)
	}
	if q.WorkspaceID != alpha.Workspace().ID {
		t.Errorf("workspaceID not set from scope")
	}

	// GetQuestions returns the seeded question.
	questions, err := alpha.GetQuestions()
	if err != nil {
		t.Fatalf("GetQuestions: %v", err)
	}
	if len(questions) != 1 || questions[0].Slug != q.Slug {
		t.Errorf("questions = %+v, want 1 with slug %q", questions, q.Slug)
	}

	// GetQuestionBySlug round-trips.
	got, err := alpha.GetQuestionBySlug(q.Slug)
	if err != nil {
		t.Fatalf("GetQuestionBySlug: %v", err)
	}
	if got.Mode != "choice" {
		t.Errorf("mode = %q, want choice", got.Mode)
	}
	if _, err := alpha.GetQuestionBySlug("nonexistent"); err == nil {
		t.Error("expected error for nonexistent question slug")
	}

	// ParseConfig returns the typed config selected by mode.
	cfg, err := got.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	cc, ok := cfg.(ChoiceConfig)
	if !ok {
		t.Fatalf("config type = %T, want ChoiceConfig", cfg)
	}
	if cc.Key != 0 || len(cc.Options) != 2 {
		t.Errorf("choice config = %+v, want key=0 options=2", cc)
	}

	// AddQuiz stores the slug array and derives the slug.
	quiz, err := alpha.AddQuiz(Quiz{
		Title:       "Genetics Foundations",
		Description: "Core factors",
		Items:       `["strongest-asd-risk-gene"]`,
	})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}
	if quiz.Slug != "genetics-foundations" {
		t.Errorf("quiz slug = %q, want genetics-foundations", quiz.Slug)
	}

	// GetQuizzes + GetQuizBySlug round-trip and ParseItems works.
	quizzes, err := alpha.GetQuizzes()
	if err != nil {
		t.Fatalf("GetQuizzes: %v", err)
	}
	if len(quizzes) != 1 || quizzes[0].Slug != quiz.Slug {
		t.Errorf("quizzes = %+v, want 1 with slug %q", quizzes, quiz.Slug)
	}
	gotQuiz, err := alpha.GetQuizBySlug(quiz.Slug)
	if err != nil {
		t.Fatalf("GetQuizBySlug: %v", err)
	}
	items, err := gotQuiz.ParseItems()
	if err != nil {
		t.Fatalf("ParseItems: %v", err)
	}
	if len(items) != 1 || items[0] != "strongest-asd-risk-gene" {
		t.Errorf("items = %+v, want [strongest-asd-risk-gene]", items)
	}
}

func TestQuizAttemptLifecycle(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	// Seed a choice question + quiz.
	q, err := alpha.AddQuestion(Question{
		Title:  "Capital of France",
		Mode:   "choice",
		Config: `{"options":["London","Paris","Berlin"],"key":1}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	quiz, err := alpha.AddQuiz(Quiz{
		Title: "Geography",
		Items: fmt.Sprintf(`["%s"]`, q.Slug),
	})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}

	// Create attempt — starts in_progress.
	qa, err := alpha.CreateQuizAttempt(quiz.ID)
	if err != nil {
		t.Fatalf("CreateQuizAttempt: %v", err)
	}
	if qa.Status != "in_progress" {
		t.Errorf("initial status = %q, want in_progress", qa.Status)
	}

	// Submit correct answer (index 1 = Paris).
	att, err := alpha.SubmitAttempt(qa.ID, q.ID, "1", 2500, nil)
	if err != nil {
		t.Fatalf("SubmitAttempt: %v", err)
	}
	if att.Correct == nil || !*att.Correct {
		t.Error("expected correct=true for Paris")
	}

	// Score is 1/1.
	correct, total := alpha.ScoreAttempt(qa.ID)
	if correct != 1 || total != 1 {
		t.Errorf("ScoreAttempt = %d/%d, want 1/1", correct, total)
	}

	// Complete the attempt.
	if err := alpha.CompleteQuizAttempt(qa.ID); err != nil {
		t.Fatalf("CompleteQuizAttempt: %v", err)
	}
	got, _ := alpha.GetQuizAttempt(qa.ID)
	if got.Status != "completed" {
		t.Errorf("status after complete = %q, want completed", got.Status)
	}

	// State machine: cannot submit to a completed attempt.
	_, err = alpha.SubmitAttempt(qa.ID, q.ID, "0", 100, nil)
	if err == nil {
		t.Error("expected error submitting to completed attempt")
	}

	// State machine: cannot complete again.
	if err := alpha.CompleteQuizAttempt(qa.ID); err == nil {
		t.Error("expected error completing already-completed attempt")
	}
}

func TestGetQuizScores(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	// Question + quiz that will be attempted.
	q, err := alpha.AddQuestion(Question{
		Title:  "Capital of France",
		Mode:   "choice",
		Config: `{"options":["London","Paris","Berlin"],"key":1}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	attempted, err := alpha.AddQuiz(Quiz{
		Title: "Geography",
		Items: fmt.Sprintf(`["%s"]`, q.Slug),
	})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}
	// Second quiz sharing the question, never attempted.
	fresh, err := alpha.AddQuiz(Quiz{Title: "Fresh", Items: fmt.Sprintf(`["%s"]`, q.Slug)})
	if err != nil {
		t.Fatalf("AddQuiz fresh: %v", err)
	}

	// Complete an attempt on "Geography" with the correct answer.
	qa, err := alpha.CreateQuizAttempt(attempted.ID)
	if err != nil {
		t.Fatalf("CreateQuizAttempt: %v", err)
	}
	if _, err := alpha.SubmitAttempt(qa.ID, q.ID, "1", 100, nil); err != nil {
		t.Fatalf("SubmitAttempt: %v", err)
	}
	if err := alpha.CompleteQuizAttempt(qa.ID); err != nil {
		t.Fatalf("CompleteQuizAttempt: %v", err)
	}

	scores, err := alpha.GetQuizScores()
	if err != nil {
		t.Fatalf("GetQuizScores: %v", err)
	}
	bySlug := map[string]QuizScore{}
	for _, s := range scores {
		bySlug[s.Slug] = s
	}

	got := bySlug[attempted.Slug]
	if !got.Attempted || got.BestScore != 1 || got.BestTotal != 1 {
		t.Errorf("attempted quiz score = %d/%d (attempted=%v), want 1/1 (attempted=true)", got.BestScore, got.BestTotal, got.Attempted)
	}

	freshScore := bySlug[fresh.Slug]
	if freshScore.Attempted || freshScore.BestScore != 0 || freshScore.BestTotal != 1 {
		t.Errorf("fresh quiz score = %d/%d (attempted=%v), want 0/1 (attempted=false)", freshScore.BestScore, freshScore.BestTotal, freshScore.Attempted)
	}
}

func TestGetQuizAttemptHistory(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	q, err := alpha.AddQuestion(Question{
		Title:  "Capital of France",
		Mode:   "choice",
		Config: `{"options":["London","Paris","Berlin"],"key":1}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	quiz, err := alpha.AddQuiz(Quiz{
		Title: "Geography",
		Items: fmt.Sprintf(`["%s"]`, q.Slug),
	})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}

	// First attempt: wrong (London, index 0).
	qa1, err := alpha.CreateQuizAttempt(quiz.ID)
	if err != nil {
		t.Fatalf("CreateQuizAttempt 1: %v", err)
	}
	if _, err := alpha.SubmitAttempt(qa1.ID, q.ID, "0", 100, nil); err != nil {
		t.Fatalf("SubmitAttempt 1: %v", err)
	}
	if err := alpha.CompleteQuizAttempt(qa1.ID); err != nil {
		t.Fatalf("CompleteQuizAttempt 1: %v", err)
	}

	// Second attempt: correct (Paris, index 1).
	qa2, err := alpha.CreateQuizAttempt(quiz.ID)
	if err != nil {
		t.Fatalf("CreateQuizAttempt 2: %v", err)
	}
	if _, err := alpha.SubmitAttempt(qa2.ID, q.ID, "1", 100, nil); err != nil {
		t.Fatalf("SubmitAttempt 2: %v", err)
	}
	if err := alpha.CompleteQuizAttempt(qa2.ID); err != nil {
		t.Fatalf("CompleteQuizAttempt 2: %v", err)
	}

	history, err := alpha.GetQuizAttemptHistory(quiz.ID)
	if err != nil {
		t.Fatalf("GetQuizAttemptHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2", len(history))
	}
	// Chronological by completed_at: first wrong (0/1), then correct (1/1).
	if history[0].Correct != 0 || history[0].Total != 1 {
		t.Errorf("first attempt = %d/%d, want 0/1", history[0].Correct, history[0].Total)
	}
	if history[1].Correct != 1 || history[1].Total != 1 {
		t.Errorf("second attempt = %d/%d, want 1/1", history[1].Correct, history[1].Total)
	}
	if history[0].CompletedAt > history[1].CompletedAt {
		t.Error("history is not chronological by completed_at")
	}

	// Reconciles with GetQuizScores: best = max of the series = 1/1.
	scores, _ := alpha.GetQuizScores()
	var best QuizScore
	for _, s := range scores {
		if s.Slug == quiz.Slug {
			best = s
		}
	}
	if !best.Attempted || best.BestScore != 1 || best.BestTotal != 1 {
		t.Errorf("best = %d/%d (attempted=%v), want 1/1 — trend must reconcile with best", best.BestScore, best.BestTotal, best.Attempted)
	}
}

func TestQuizLessonLink(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	lesson, err := alpha.AddLesson(Lesson{Title: "JOINs", Filename: "0001-joins.html"})
	if err != nil {
		t.Fatalf("AddLesson: %v", err)
	}
	q, err := alpha.AddQuestion(Question{
		Title:  "What is a JOIN?",
		Mode:   "choice",
		Config: `{"options":["a","b"],"key":0}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	seq := lesson.SequenceNumber
	linked, err := alpha.AddQuiz(Quiz{
		Title: "JOINs quiz",
		Items: fmt.Sprintf(`["%s"]`, q.Slug),
	})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}
	if linked.LessonSeq != nil {
		t.Errorf("created quiz LessonSeq = %v, want nil (not linked yet)", linked.LessonSeq)
	}

	// Link via the seam.
	lc := alpha.LessonContent()
	if err := lc.SetQuizLesson(linked.Slug, seq); err != nil {
		t.Fatalf("SetQuizLesson: %v", err)
	}

	// Forward: read back the link.
	got, err := alpha.GetQuizBySlug(linked.Slug)
	if err != nil {
		t.Fatalf("GetQuizBySlug: %v", err)
	}
	if got.LessonSeq == nil || *got.LessonSeq != seq {
		t.Errorf("read-back LessonSeq = %v, want %d", got.LessonSeq, seq)
	}

	// Reverse: QuizzesForLesson finds it.
	rev, err := lc.QuizzesForLesson(seq)
	if err != nil {
		t.Fatalf("QuizzesForLesson: %v", err)
	}
	if len(rev) != 1 || rev[0].Slug != linked.Slug {
		t.Errorf("reverse lookup = %d quizzes, want 1 with slug %q", len(rev), linked.Slug)
	}

	// An unlinked quiz has nil LessonSeq and is excluded from the reverse lookup.
	unlinked, _ := alpha.AddQuiz(Quiz{Title: "General", Items: fmt.Sprintf(`["%s"]`, q.Slug)})
	if unlinked.LessonSeq != nil {
		t.Errorf("unlinked quiz LessonSeq = %v, want nil", unlinked.LessonSeq)
	}
	rev2, _ := lc.QuizzesForLesson(seq)
	if len(rev2) != 1 {
		t.Errorf("reverse lookup after adding unlinked = %d, want 1", len(rev2))
	}

	// ClearQuizLesson removes the link. Idempotent — clear twice is fine.
	if err := lc.ClearQuizLesson(linked.Slug); err != nil {
		t.Fatalf("ClearQuizLesson: %v", err)
	}
	cleared, _ := alpha.GetQuizBySlug(linked.Slug)
	if cleared.LessonSeq != nil {
		t.Errorf("after clear, LessonSeq = %v, want nil", cleared.LessonSeq)
	}
	if err := lc.ClearQuizLesson(linked.Slug); err != nil {
		t.Fatalf("ClearQuizLesson (idempotent): %v", err)
	}

	// SetQuizLesson re-links. Overwrites existing link.
	if err := lc.SetQuizLesson(linked.Slug, seq); err != nil {
		t.Fatalf("SetQuizLesson: %v", err)
	}
	reset, _ := alpha.GetQuizBySlug(linked.Slug)
	if reset.LessonSeq == nil || *reset.LessonSeq != seq {
		t.Errorf("after re-set, LessonSeq = %v, want %d", reset.LessonSeq, seq)
	}
	// Setting the same lesson again works (idempotent overwrite).
	if err := lc.SetQuizLesson(linked.Slug, seq); err != nil {
		t.Fatalf("SetQuizLesson (overwrite): %v", err)
	}
}

func TestQuizLessonLinkErrors(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")

	lesson, err := alpha.AddLesson(Lesson{Title: "JOINs", Filename: "0001-joins.html"})
	if err != nil {
		t.Fatalf("AddLesson: %v", err)
	}
	q, err := alpha.AddQuestion(Question{
		Title:  "What is a JOIN?",
		Mode:   "choice",
		Config: `{"options":["a","b"],"key":0}`,
	})
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	quiz, err := alpha.AddQuiz(Quiz{Title: "Quiz", Items: fmt.Sprintf(`["%s"]`, q.Slug)})
	if err != nil {
		t.Fatalf("AddQuiz: %v", err)
	}
	lc := alpha.LessonContent()

	// SetQuizLesson with stale quiz slug.
	if err := lc.SetQuizLesson("nonexistent", 1); !errors.Is(err, ErrQuizNotFound) {
		t.Errorf("SetQuizLesson with bad slug: got %v, want ErrQuizNotFound", err)
	}

	// SetQuizLesson with nonexistent lesson sequence.
	if err := lc.SetQuizLesson(quiz.Slug, 999); !errors.Is(err, ErrLessonNotFound) {
		t.Errorf("SetQuizLesson with bad lesson: got %v, want ErrLessonNotFound", err)
	}

	// SetQuizLesson with valid inputs succeeds.
	seq := lesson.SequenceNumber
	if err := lc.SetQuizLesson(quiz.Slug, seq); err != nil {
		t.Fatalf("SetQuizLesson valid: %v", err)
	}

	// ClearQuizLesson with stale quiz slug.
	if err := lc.ClearQuizLesson("nonexistent"); !errors.Is(err, ErrQuizNotFound) {
		t.Errorf("ClearQuizLesson with bad slug: got %v, want ErrQuizNotFound", err)
	}

	// ClearQuizLesson when already unlinked is idempotent (no error).
	if err := lc.ClearQuizLesson(quiz.Slug); err != nil {
		t.Fatalf("ClearQuizLesson (unlinked): %v", err)
	}

	// QuizzesForLesson with no matches returns empty, not error.
	empty, err := lc.QuizzesForLesson(999)
	if err != nil {
		t.Fatalf("QuizzesForLesson unknown lesson: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("QuizzesForLesson unknown lesson: got %d quizzes, want 0", len(empty))
	}
}

func TestQuizAttemptAbandon(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	q, _ := alpha.AddQuestion(Question{
		Title:  "q1",
		Mode:   "choice",
		Config: `{"options":["a","b"],"key":0}`,
	})
	quiz, _ := alpha.AddQuiz(Quiz{Title: "Quiz", Items: fmt.Sprintf(`["%s"]`, q.Slug)})

	qa, _ := alpha.CreateQuizAttempt(quiz.ID)

	// Abandon an in_progress attempt.
	if err := alpha.AbandonQuizAttempt(qa.ID); err != nil {
		t.Fatalf("AbandonQuizAttempt: %v", err)
	}
	got, _ := alpha.GetQuizAttempt(qa.ID)
	if got.Status != "abandoned" {
		t.Errorf("status = %q, want abandoned", got.Status)
	}

	// State machine: cannot submit to abandoned attempt.
	_, err := alpha.SubmitAttempt(qa.ID, q.ID, "0", 100, nil)
	if err == nil {
		t.Error("expected error submitting to abandoned attempt")
	}

	// State machine: cannot abandon again.
	if err := alpha.AbandonQuizAttempt(qa.ID); err == nil {
		t.Error("expected error abandoning already-abandoned attempt")
	}
}

func TestChoiceConfigGrade(t *testing.T) {
	cc := ChoiceConfig{Options: []string{"a", "b", "c"}, Key: 1}
	cases := []struct {
		response string
		want     bool
	}{
		{"0", false},
		{"1", true},
		{"2", false},
	}
	for _, c := range cases {
		got, err := cc.Grade(c.response)
		if err != nil {
			t.Errorf("Grade(%q) error: %v", c.response, err)
		}
		if got != c.want {
			t.Errorf("Grade(%q) = %v, want %v", c.response, got, c.want)
		}
	}
	// Non-numeric response errors.
	if _, err := cc.Grade("abc"); err == nil {
		t.Error("expected error for non-numeric response")
	}
}

// TestGetSidebarData proves GetSidebarData returns the lightweight sidebar
// projections (not full model structs) with correct fields and workspace
// scoping. One call replaces three separate Get* calls.
func TestGetSidebarData(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md", Status: "active", Summary: "sum"})
	alpha.AddRef(Reference{Title: "Ref1", Slug: "ref1", Filename: "ref1.html"})
	alpha.AddQuiz(Quiz{Title: "Quiz1", Slug: "quiz1", Items: `["q1"]`})

	// Empty workspace
	beta := seedWorkspace(t, store, "beta")

	// Alpha: 2 lessons, 1 record, 1 ref
	sd, err := alpha.GetSidebarData()
	if err != nil {
		t.Fatalf("GetSidebarData: %v", err)
	}
	if sd.Workspace.Name != "alpha" {
		t.Errorf("workspace name = %q, want alpha", sd.Workspace.Name)
	}
	if len(sd.Lessons) != 2 {
		t.Fatalf("lessons = %d, want 2", len(sd.Lessons))
	}
	if sd.Lessons[0].Seq != 1 || sd.Lessons[0].Title != "a1" {
		t.Errorf("lesson[0] = %+v, want {1, a1}", sd.Lessons[0])
	}
	if len(sd.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(sd.Records))
	}
	if sd.Records[0].Status != "active" || sd.Records[0].Summary != "sum" {
		t.Errorf("record[0] = %+v, want status=active summary=sum", sd.Records[0])
	}
	if len(sd.Refs) != 1 {
		t.Fatalf("refs = %d, want 1", len(sd.Refs))
	}
	if sd.Refs[0].Slug != "ref1" || sd.Refs[0].Title != "Ref1" {
		t.Errorf("ref[0] = %+v, want {ref1, Ref1}", sd.Refs[0])
	}
	if len(sd.Quizzes) != 1 {
		t.Fatalf("quizzes = %d, want 1", len(sd.Quizzes))
	}
	if sd.Quizzes[0].Slug != "quiz1" || sd.Quizzes[0].Title != "Quiz1" {
		t.Errorf("quiz[0] = %+v, want {quiz1, Quiz1}", sd.Quizzes[0])
	}

	// Beta: empty workspace — no items leaked from alpha
	sdBeta, err := beta.GetSidebarData()
	if err != nil {
		t.Fatalf("beta GetSidebarData: %v", err)
	}
	if len(sdBeta.Lessons) != 0 || len(sdBeta.Records) != 0 || len(sdBeta.Refs) != 0 || len(sdBeta.Quizzes) != 0 {
		t.Errorf("beta should be empty, got L%d R%d Ref%d Q%d",
			len(sdBeta.Lessons), len(sdBeta.Records), len(sdBeta.Refs), len(sdBeta.Quizzes))
	}
}

// TestIndexLessons proves the backfill reads lesson HTML files from disk,
// extracts plain text, updates the DB, and makes lessons FTS-searchable.
// A second call is a no-op (all body_text fields are already populated).
func TestIndexLessons(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	// Write lesson HTML files to disk (simulating lessons made before
	// the body_text migration).
	lessonsDir := filepath.Join(dir, "lessons")
	if err := os.MkdirAll(lessonsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	html1 := `<html><body><h1>Introduction</h1><p>Welcome to SQL basics.</p></body></html>`
	if err := os.WriteFile(filepath.Join(lessonsDir, "0001-intro.html"), []byte(html1), 0o644); err != nil {
		t.Fatal(err)
	}
	html2 := `<html><body><h1>JOINs</h1><p>Combine tables with INNER JOIN.</p></body></html>`
	if err := os.WriteFile(filepath.Join(lessonsDir, "0002-joins.html"), []byte(html2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add lessons via AddLesson (does NOT set body_text).
	if _, err := ws.AddLesson(Lesson{Title: "Intro", Filename: "0001-intro.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddLesson(Lesson{Title: "JOINs", Filename: "0002-joins.html"}); err != nil {
		t.Fatal(err)
	}

	// Confirm body_text is empty on disk.
	l1, _ := ws.GetLessonBySeq(1)
	if l1.BodyText != "" {
		t.Fatalf("expected empty body_text before backfill, got %q", l1.BodyText)
	}

	// Backfill.
	n, err := ws.IndexLessons()
	if err != nil {
		t.Fatalf("IndexLessons: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 backfilled, got %d", n)
	}

	// Verify body_text is populated.
	l1, _ = ws.GetLessonBySeq(1)
	if !strings.Contains(l1.BodyText, "Introduction") || !strings.Contains(l1.BodyText, "SQL basics") {
		t.Errorf("body_text = %q, want it to contain both 'Introduction' and 'SQL basics'", l1.BodyText)
	}
	l2, _ := ws.GetLessonBySeq(2)
	if !strings.Contains(l2.BodyText, "JOINs") || !strings.Contains(l2.BodyText, "INNER JOIN") {
		t.Errorf("body_text = %q, want it to contain both 'JOINs' and 'INNER JOIN'", l2.BodyText)
	}

	// Verify FTS search returns results.
	results, err := ws.SearchLessons("SQL basics")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("FTS search for 'SQL basics' returned no results after backfill")
	}

	// Second call is a no-op.
	n, err = ws.IndexLessons()
	if err != nil {
		t.Fatalf("second IndexLessons: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 on second call, got %d", n)
	}
}

// TestIndexLessonsMissingFile proves the backfill does not crash when a
// lesson's HTML file is missing; it skips that lesson, reports an error, and
// continues with remaining lessons.
func TestIndexLessonsMissingFile(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	lessonsDir := filepath.Join(dir, "lessons")
	if err := os.MkdirAll(lessonsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	html := `<html><body><p>Only this one exists on disk.</p></body></html>`
	if err := os.WriteFile(filepath.Join(lessonsDir, "0001-exists.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddLesson(Lesson{Title: "Exists", Filename: "0001-exists.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddLesson(Lesson{Title: "Missing", Filename: "0002-missing.html"}); err != nil {
		t.Fatal(err)
	}

	// Backfill — one file exists, one does not.
	n, err := ws.IndexLessons()
	if n != 1 {
		t.Fatalf("expected 1 backfilled, got %d", n)
	}
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	// The existing file was backfilled.
	l1, _ := ws.GetLessonBySeq(1)
	if !strings.Contains(l1.BodyText, "Only this one") {
		t.Errorf("body_text = %q, want 'Only this one'", l1.BodyText)
	}

	// The missing file still has empty body_text.
	l2, _ := ws.GetLessonBySeq(2)
	if l2.BodyText != "" {
		t.Errorf("expected empty body_text for missing file, got %q", l2.BodyText)
	}
}

// TestGetWorkspacesCountsCorrect proves LEARN-17's batch enrichment: counts
// are correct across multiple workspaces with varying lesson/record/ref
// counts, via the grouped COUNT(*) queries (not the old per-workspace N+1).
func TestGetWorkspacesCountsCorrect(t *testing.T) {
	store := newTestStore(t)

	// alpha: 2 lessons, 1 record, 1 ref
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md"})
	alpha.AddRef(Reference{Title: "ref1", Filename: "ref1.html"})

	// beta: 0 lessons, 2 records, 0 refs
	beta := seedWorkspace(t, store, "beta")
	beta.AddRecord(LearningRecord{Title: "b1", Filename: "b1.md"})
	beta.AddRecord(LearningRecord{Title: "b2", Filename: "b2.md"})

	ws, err := store.GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces: %v", err)
	}
	byName := map[string]Workspace{}
	for _, w := range ws {
		byName[w.Name] = w
	}

	cases := []struct {
		name                  string
		wantL, wantR, wantRef int
	}{
		{"alpha", 2, 1, 1},
		{"beta", 0, 2, 0},
	}
	for _, c := range cases {
		w := byName[c.name]
		if w.LessonCount != c.wantL || w.RecordCount != c.wantR || w.RefCount != c.wantRef {
			t.Errorf("%s: counts = L%d R%d Ref%d, want L%d R%d Ref%d",
				c.name, w.LessonCount, w.RecordCount, w.RefCount, c.wantL, c.wantR, c.wantRef)
		}
	}
}

// TestWorkspaceStoreSearch proves WorkspaceStore.Search returns results from
// all three entity types within a single workspace. FTS indexing of body_text
// is verified via the body_text backfill test; this test focuses on the
// cross-entity aggregation.
func TestWorkspaceStoreSearch(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "test")

	// Add a lesson with a searchable summary.
	ws.AddLesson(Lesson{Title: "SQL Joins", Filename: "j.html", Summary: "Combining tables with JOIN"})
	// Add a record.
	ws.AddRecord(LearningRecord{Title: "Understood JOINs", Filename: "r.md", Summary: "INNER vs LEFT JOIN"})
	// Add a reference.
	ws.AddRef(Reference{Title: "JOIN Cheatsheet", Slug: "joins", Filename: "jc.html", Summary: "Quick JOIN syntax"})

	results, err := ws.Search("JOIN")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	types := map[string]bool{}
	for _, r := range results {
		types[r.Type] = true
		if r.WorkspaceName != "test" {
			t.Errorf("result workspace = %q, want test", r.WorkspaceName)
		}
	}
	if !types["lesson"] || !types["record"] || !types["ref"] {
		t.Errorf("missing entity types, got %v", types)
	}
}

// TestFTSTriggersIndexAtInsert proves the FTS5 AFTER INSERT triggers maintain
// the search index at insert time — no RebuildFTS() call is needed on Open or
// before Search. Regression guard for the removal of the redundant
// store.RebuildFTS() call that used to run on every Open() (LEARN-102): if a
// trigger is dropped or its body_text column drifts, a newly-created item is
// no longer searchable by a term that appears only in its body, and this
// test fails.
func TestFTSTriggersIndexAtInsert(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "trig")

	// Distinctive body term that appears in NO title or summary, only in body.
	const needle = "pterodactyl"

	// Create one of each entity type via the create path (the path agents use),
	// each with the needle only in the body — never in title or summary.
	if _, err := ws.CreateLesson("Lesson Title", "<p>body "+needle+"</p>"); err != nil {
		t.Fatalf("CreateLesson: %v", err)
	}
	if _, err := ws.CreateRecord("Record Title", "body "+needle, "summary without the needle"); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if _, err := ws.CreateRef("Ref Title", "<p>body "+needle+"</p>"); err != nil {
		t.Fatalf("CreateRef: %v", err)
	}

	// Search immediately — no rebuild call anywhere. If the _ai triggers work,
	// the needle (present only in body_text) is indexed at insert and found.
	results, err := ws.Search(needle)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 trigger-indexed results for %q, got %d "+
			"(FTS AFTER INSERT trigger not maintaining index?)",
			needle, len(results))
	}
}

// TestFTSUpdateTriggerResyncsOnIndexedColumnChange proves the scoped _au
// trigger (AFTER UPDATE OF title, summary, body_text) still fires when an
// indexed column changes. ReviseLesson SETs title/summary/body_text, so the
// trigger resyncs FTS: the new title is searchable and the old is gone.
// Regression guard for LEARN-106 — if the scoping accidentally excluded an
// indexed column, search would return stale results.
func TestFTSUpdateTriggerResyncsOnIndexedColumnChange(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "rev")

	l, err := ws.CreateLesson("guitar", "<p>body content</p>")
	if err != nil {
		t.Fatalf("CreateLesson: %v", err)
	}

	newTitle := "banjo"
	if err := ws.ReviseLesson(int(l.SequenceNumber), "<p>body content</p>", &newTitle, nil); err != nil {
		t.Fatalf("ReviseLesson: %v", err)
	}

	if results, err := ws.Search("banjo"); err != nil {
		t.Fatalf("Search new title: %v", err)
	} else if len(results) != 1 {
		t.Errorf("expected new title searchable, got %d results", len(results))
	}
	if results, err := ws.Search("guitar"); err != nil {
		t.Fatalf("Search old title: %v", err)
	} else if len(results) != 0 {
		t.Errorf("expected old title gone from index, got %d results", len(results))
	}
}

// TestSearchResultSnippet proves that a lesson matching only on body_text
// (empty summary) gets a preview snippet in the search result.
func TestSearchResultSnippet(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "test")

	// Add a lesson with body_text but no summary. AddLesson doesn't set
	// body_text, so we update it directly — simulating a lesson that was
	// indexed but never had a summary.
	l, err := ws.AddLesson(Lesson{Title: "Deep Dive", Filename: "deep.html"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("UPDATE lessons SET body_text = 'This lesson covers advanced topics like JOINs and subqueries in depth.' WHERE id = ?", l.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Add a record with summary (should not have a snippet).
	ws.AddRecord(LearningRecord{Title: "Understood JOINs", Filename: "r.md", Summary: "INNER vs LEFT JOIN"})

	results, err := ws.Search("JOINs")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Type == "lesson" {
			if r.Summary != "" {
				t.Errorf("lesson summary should be empty, got %q", r.Summary)
			}
			if r.Snippet == "" {
				t.Error("lesson with body_text should have a snippet when summary is empty")
			}
			if !strings.Contains(r.Snippet, "JOINs") {
				t.Errorf("snippet should contain the matched term, got %q", r.Snippet)
			}
		}
		if r.Type == "record" {
			if r.Snippet != "" {
				t.Errorf("record should not have a snippet, got %q", r.Snippet)
			}
		}
	}
}

// TestStoreSearch proves Store.Search returns results across multiple
// workspaces, correctly aggregating all entity types.
func TestStoreSearch(t *testing.T) {
	store := newTestStore(t)

	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "Alpha Lesson", Filename: "a.html", Summary: "Alpha content with joins"})
	alpha.AddRecord(LearningRecord{Title: "Alpha Record", Filename: "ar.md", Summary: "Alpha joins record"})

	beta := seedWorkspace(t, store, "beta")
	beta.AddRef(Reference{Title: "Beta Ref", Slug: "beta-ref", Filename: "br.html", Summary: "Beta joins reference"})

	results, err := store.Search("joins")
	if err != nil {
		t.Fatalf("Store.Search: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results across 2 workspaces, got %d", len(results))
	}

	workspaces := map[string]int{}
	for _, r := range results {
		workspaces[r.WorkspaceName]++
	}
	if workspaces["alpha"] != 2 {
		t.Errorf("expected 2 results from alpha, got %d", workspaces["alpha"])
	}
	if workspaces["beta"] != 1 {
		t.Errorf("expected 1 result from beta, got %d", workspaces["beta"])
	}
}

// TestStoreSearchEmptyQuery proves Store.Search returns no results for an
// empty/no-match query rather than erroring.
func TestStoreSearchEmptyQuery(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "test")
	ws.AddLesson(Lesson{Title: "Something", Filename: "s.html", Summary: "unrelated"})

	results, err := store.Search("zzzznonexistent")
	if err != nil {
		t.Fatalf("Store.Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestIndexRefs proves IndexRefs reads ref HTML files from disk, extracts
// plain text, updates the DB, and makes refs FTS-searchable. A second call
// is a no-op.
func TestIndexRefs(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	refDir := filepath.Join(dir, "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	html1 := `<html><body><h1>SQL Cheatsheet</h1><p>Common SQL commands and syntax.</p></body></html>`
	if err := os.WriteFile(filepath.Join(refDir, "sql-cheatsheet.html"), []byte(html1), 0o644); err != nil {
		t.Fatal(err)
	}
	html2 := `<html><body><h1>Go Snippets</h1><p>Useful Go patterns.</p></body></html>`
	if err := os.WriteFile(filepath.Join(refDir, "go-snippets.html"), []byte(html2), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddRef(Reference{Title: "SQL Cheatsheet", Slug: "sql-cheatsheet", Filename: "sql-cheatsheet.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddRef(Reference{Title: "Go Snippets", Slug: "go-snippets", Filename: "go-snippets.html"}); err != nil {
		t.Fatal(err)
	}

	r1, _ := ws.GetRefBySlug("sql-cheatsheet")
	if r1.BodyText != "" {
		t.Fatalf("expected empty body_text before backfill, got %q", r1.BodyText)
	}

	n, err := ws.IndexRefs()
	if err != nil {
		t.Fatalf("IndexRefs: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 indexed, got %d", n)
	}

	r1, _ = ws.GetRefBySlug("sql-cheatsheet")
	if !strings.Contains(r1.BodyText, "SQL commands") {
		t.Errorf("body_text = %q, want 'SQL commands'", r1.BodyText)
	}
	r2, _ := ws.GetRefBySlug("go-snippets")
	if !strings.Contains(r2.BodyText, "Go patterns") {
		t.Errorf("body_text = %q, want 'Go patterns'", r2.BodyText)
	}

	results, err := ws.SearchRefs("SQL commands")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("FTS search returned no results after IndexRefs")
	}

	n, err = ws.IndexRefs()
	if err != nil {
		t.Fatalf("second IndexRefs: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 on second call, got %d", n)
	}
}

// TestIndexRefsMissingFile proves IndexRefs skips missing files and reports errors.
func TestIndexRefsMissingFile(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	refDir := filepath.Join(dir, "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	html := `<html><body><p>Exists on disk.</p></body></html>`
	if err := os.WriteFile(filepath.Join(refDir, "exists.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddRef(Reference{Title: "Exists", Slug: "exists", Filename: "exists.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddRef(Reference{Title: "Missing", Slug: "missing", Filename: "missing.html"}); err != nil {
		t.Fatal(err)
	}

	n, err := ws.IndexRefs()
	if n != 1 {
		t.Fatalf("expected 1 indexed, got %d", n)
	}
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	r1, _ := ws.GetRefBySlug("exists")
	if !strings.Contains(r1.BodyText, "Exists on disk") {
		t.Errorf("body_text = %q, want 'Exists on disk'", r1.BodyText)
	}
	r2, _ := ws.GetRefBySlug("missing")
	if r2.BodyText != "" {
		t.Errorf("expected empty body_text for missing file, got %q", r2.BodyText)
	}
}

// TestSearchResultRefSnippet proves a ref matching only on body_text gets a
// preview snippet.
func TestSearchResultRefSnippet(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "test")

	r, err := ws.AddRef(Reference{Title: "Deep Ref", Slug: "deep-ref", Filename: "deep.html"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("UPDATE references_t SET body_text = 'This reference covers advanced SQL JOINs and subqueries in depth.' WHERE id = ?", r.ID)
	if err != nil {
		t.Fatal(err)
	}

	results, err := ws.Search("JOINs")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "ref" {
		t.Errorf("expected type ref, got %q", results[0].Type)
	}
	if results[0].Summary != "" {
		t.Errorf("expected empty summary, got %q", results[0].Summary)
	}
	if results[0].Snippet == "" {
		t.Error("ref with body_text should have a snippet when summary is empty")
	}
	if !strings.Contains(results[0].Snippet, "JOINs") {
		t.Errorf("snippet should contain matched term, got %q", results[0].Snippet)
	}
}

// TestIndexRecords proves IndexRecords reads record markdown files from disk,
// extracts plain text, updates the DB, and makes records FTS-searchable.
// A second call is a no-op.
func TestIndexRecords(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	recDir := filepath.Join(dir, "learning-records")
	if err := os.MkdirAll(recDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md1 := `# Understanding JOINs

INNER JOIN returns only matching rows.
LEFT JOIN keeps all rows from the left table.`
	if err := os.WriteFile(filepath.Join(recDir, "0001-joins.md"), []byte(md1), 0o644); err != nil {
		t.Fatal(err)
	}
	md2 := `## Key Takeaway

Always prefer explicit JOIN syntax over implicit commas.`
	if err := os.WriteFile(filepath.Join(recDir, "0002-syntax.md"), []byte(md2), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddRecord(LearningRecord{Title: "JOINs", Filename: "0001-joins.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddRecord(LearningRecord{Title: "Syntax", Filename: "0002-syntax.md"}); err != nil {
		t.Fatal(err)
	}

	r1, _ := ws.GetRecordBySeq(1)
	if r1.BodyText != "" {
		t.Fatalf("expected empty body_text before index, got %q", r1.BodyText)
	}

	n, err := ws.IndexRecords()
	if err != nil {
		t.Fatalf("IndexRecords: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 indexed, got %d", n)
	}

	r1, _ = ws.GetRecordBySeq(1)
	if !strings.Contains(r1.BodyText, "INNER JOIN") {
		t.Errorf("body_text = %q, want 'INNER JOIN'", r1.BodyText)
	}
	r2, _ := ws.GetRecordBySeq(2)
	if !strings.Contains(r2.BodyText, "explicit JOIN") {
		t.Errorf("body_text = %q, want 'explicit JOIN'", r2.BodyText)
	}

	results, err := ws.SearchRecords("INNER JOIN")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("FTS search returned no results after IndexRecords")
	}

	n, err = ws.IndexRecords()
	if err != nil {
		t.Fatalf("second IndexRecords: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 on second call, got %d", n)
	}
}

// TestIndexRecordsMissingFile proves IndexRecords skips missing files and reports errors.
func TestIndexRecordsMissingFile(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	recDir := filepath.Join(dir, "learning-records")
	if err := os.MkdirAll(recDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := `# Exists

This record exists on disk.`
	if err := os.WriteFile(filepath.Join(recDir, "0001-exists.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddRecord(LearningRecord{Title: "Exists", Filename: "0001-exists.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddRecord(LearningRecord{Title: "Missing", Filename: "0002-missing.md"}); err != nil {
		t.Fatal(err)
	}

	n, err := ws.IndexRecords()
	if n != 1 {
		t.Fatalf("expected 1 indexed, got %d", n)
	}
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	r1, _ := ws.GetRecordBySeq(1)
	if !strings.Contains(r1.BodyText, "Exists") || !strings.Contains(r1.BodyText, "exists on disk") {
		t.Errorf("body_text = %q, want 'Exists on disk'", r1.BodyText)
	}
	r2, _ := ws.GetRecordBySeq(2)
	if r2.BodyText != "" {
		t.Errorf("expected empty body_text for missing file, got %q", r2.BodyText)
	}
}

// TestSearchResultRecordSnippet proves a record matching only on body_text gets
// a preview snippet.
func TestSearchResultRecordSnippet(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "test")

	r, err := ws.AddRecord(LearningRecord{Title: "Deep Record", Filename: "deep.md"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("UPDATE learning_records SET body_text = 'This record covers advanced SQL JOINs and subqueries in depth.' WHERE id = ?", r.ID)
	if err != nil {
		t.Fatal(err)
	}

	results, err := ws.Search("JOINs")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "record" {
		t.Errorf("expected type record, got %q", results[0].Type)
	}
	if results[0].Summary != "" {
		t.Errorf("expected empty summary, got %q", results[0].Summary)
	}
	if results[0].Snippet == "" {
		t.Error("record with body_text should have a snippet when summary is empty")
	}
	if !strings.Contains(results[0].Snippet, "JOINs") {
		t.Errorf("snippet should contain matched term, got %q", results[0].Snippet)
	}
}

func TestSearchPrefixAndPorter(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	_, err := store.AddWorkspace(Workspace{Name: "test", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := store.Workspace("test")
	if err != nil {
		t.Fatal(err)
	}

	lessonsDir := filepath.Join(dir, "lessons")
	if err := os.MkdirAll(lessonsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	html1 := `<html><body><h1>Brain Disorders</h1><p>The neurotic patient exhibited neurological deficits.</p></body></html>`
	if err := os.WriteFile(filepath.Join(lessonsDir, "0001-neuro.html"), []byte(html1), 0o644); err != nil {
		t.Fatal(err)
	}
	html2 := `<html><body><h1>Database Joins</h1><p>We are joining tables. The joined results were surprising.</p></body></html>`
	if err := os.WriteFile(filepath.Join(lessonsDir, "0002-joins.html"), []byte(html2), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.AddLesson(Lesson{Title: "Brain Disorders", Filename: "0001-neuro.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.AddLesson(Lesson{Title: "Database Joins", Filename: "0002-joins.html"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.IndexLessons(); err != nil {
		t.Fatalf("IndexLessons: %v", err)
	}

	neuroResults, err := ws.SearchLessons("neuro")
	if err != nil {
		t.Fatalf("search neuro: %v", err)
	}
	if len(neuroResults) == 0 {
		t.Error("prefix search for 'neuro' returned no results; expected to match neurotic/neurological")
	}

	joinResults, err := ws.SearchLessons("joins")
	if err != nil {
		t.Fatalf("search joins: %v", err)
	}
	if len(joinResults) == 0 {
		t.Error("porter search for 'joins' returned no results; expected to match joining/joined")
	}

	emptyResults, err := ws.SearchLessons("")
	if err != nil {
		t.Fatalf("search empty: %v", err)
	}
	if len(emptyResults) != 0 {
		t.Errorf("empty query returned %d results, expected 0", len(emptyResults))
	}

	noResults, err := ws.SearchLessons("xyzzy")
	if err != nil {
		t.Fatalf("search xyzzy: %v", err)
	}
	if len(noResults) != 0 {
		t.Errorf("search for 'xyzzy' returned %d results, expected 0", len(noResults))
	}
}
