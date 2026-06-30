package db

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Workspace represents a learning workspace.
type Workspace struct {
	ID            int64  `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`   // directory name
	Topic         string `db:"topic" json:"topic"` // user-friendly topic
	Path          string `db:"path" json:"path"`   // absolute path to workspace dir
	MissionWhy    string `db:"mission_why" json:"missionWhy"`
	LastLessonSeq *int   `db:"last_lesson_seq" json:"lastLessonSeq,omitempty"`
	LastRecordSeq *int   `db:"last_record_seq" json:"lastRecordSeq,omitempty"`
	LastRefSeq    *int   `db:"last_ref_seq" json:"lastRefSeq,omitempty"`
	LessonCount   int    `db:"-" json:"lessonCount"` // computed
	RecordCount   int    `db:"-" json:"recordCount"` // computed
	RefCount      int    `db:"-" json:"refCount"`    // computed
	QuizCount     int    `db:"-" json:"quizCount"`   // computed
	CreatedAt     string `db:"created_at" json:"createdAt"`
	LastStudied   string `db:"last_studied" json:"lastStudied"`
}

// Lesson represents a single lesson file.
type Lesson struct {
	ID             int64  `db:"id" json:"id"`
	WorkspaceID    int64  `db:"workspace_id" json:"workspaceId"`
	Title          string `db:"title" json:"title"`
	SequenceNumber int    `db:"sequence_number" json:"sequenceNumber"`
	Filename       string `db:"filename" json:"filename"` // e.g. 0003-joins.html
	Path           string `db:"path" json:"path"`         // relative to workspace
	Summary        string `db:"summary" json:"summary"`
	BodyText       string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt      string `db:"created_at" json:"createdAt"`
	UpdatedAt      string `db:"updated_at" json:"updatedAt"`
}

// LearningRecord represents an ADR-style learning record.
type LearningRecord struct {
	ID             int64  `db:"id" json:"id"`
	WorkspaceID    int64  `db:"workspace_id" json:"workspaceId"`
	Title          string `db:"title" json:"title"`
	SequenceNumber int    `db:"sequence_number" json:"sequenceNumber"`
	Filename       string `db:"filename" json:"filename"` // e.g. 0003-joins.md
	Path           string `db:"path" json:"path"`         // relative to workspace
	Status         string `db:"status" json:"status"`     // active | superseded
	SupersededBy   int64  `db:"superseded_by" json:"supersededBy,omitempty"`
	Summary        string `db:"summary" json:"summary"`
	BodyText       string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt      string `db:"created_at" json:"createdAt"`
	UpdatedAt      string `db:"updated_at" json:"updatedAt"`
}

// Reference represents a reference document (cheat sheet).
type Reference struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Title       string `db:"title" json:"title"`
	Slug        string `db:"slug" json:"slug"`
	Filename    string `db:"filename" json:"filename"`
	Path        string `db:"path" json:"path"`
	Summary     string `db:"summary" json:"summary"`
	BodyText    string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// Question represents a single question in a workspace. Questions are
// DB-only (no file on disk). Config holds the mode-specific JSON; use
// ParseConfig for typed access.
type Question struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Title       string `db:"title" json:"title"`
	Slug        string `db:"slug" json:"slug"`
	Mode        string `db:"mode" json:"mode"`     // "choice" | "recall"
	Config      string `db:"config" json:"config"` // raw JSON; use ParseConfig for typed access
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// Quiz represents an ordered list of question slugs grouped under a title.
// Items is the raw JSON slug array; use ParseItems for typed access.
// LessonSeq optionally links the quiz to the lesson whose skill it practices
// (nil = unlinked). It is a soft reference by (workspace, sequence_number),
// not a FK — resolved at read time.
type Quiz struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Title       string `db:"title" json:"title"`
	Slug        string `db:"slug" json:"slug"`
	Description string `db:"description" json:"description"`
	Items       string `db:"items" json:"items"` // raw JSON array of question slugs
	LessonSeq   *int   `db:"lesson_seq" json:"lessonSeq,omitempty"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// QuestionConfig is the typed, mode-specific shape of a Question's config.
// Each concrete config validates its own invariants and grades responses.
type QuestionConfig interface {
	Mode() string
	Validate() error
	Grade(response string) (bool, error)
}

// QuizAttempt tracks a single run through a quiz. Status transitions:
// in_progress → completed | abandoned (enforced at the store layer).
type QuizAttempt struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	QuizID      int64  `db:"quiz_id" json:"quizId"`
	Status      string `db:"status" json:"status"` // in_progress | completed | abandoned
	StartedAt   string `db:"started_at" json:"startedAt"`
	CompletedAt string `db:"completed_at" json:"completedAt,omitempty"`
}

// Attempt is one answered question within a QuizAttempt. For Choice mode,
// Correct is set server-side by Config.Grade. For Recall (Slice 3), the
// client sends the self-grade and Correct is set from it.
type Attempt struct {
	ID            int64  `db:"id" json:"id"`
	QuizAttemptID int64  `db:"quiz_attempt_id" json:"quizAttemptId"`
	QuestionID    int64  `db:"question_id" json:"questionId"`
	Correct       *bool  `db:"correct" json:"correct,omitempty"` // nullable until answered
	Response      string `db:"response" json:"response"`
	LatencyMs     *int   `db:"latency_ms" json:"latencyMs,omitempty"`
	CreatedAt     string `db:"created_at" json:"createdAt"`
}

// ChoiceConfig is the config for a choice-mode question: a list of options
// and the 0-based index of the correct answer.
type ChoiceConfig struct {
	Options []string `json:"options"`
	Key     int      `json:"key"`
}

func (c ChoiceConfig) Mode() string { return "choice" }

// Validate checks that a choice config has at least two options and an
// in-range correct-answer index.
func (c ChoiceConfig) Validate() error {
	if len(c.Options) < 2 {
		return fmt.Errorf("need at least 2 options")
	}
	if c.Key < 0 || c.Key >= len(c.Options) {
		return fmt.Errorf("key out of range")
	}
	return nil
}

// Grade parses the response as the selected option index and compares it
// to the correct key. Returns true if they match.
func (c ChoiceConfig) Grade(response string) (bool, error) {
	idx, err := strconv.Atoi(response)
	if err != nil {
		return false, fmt.Errorf("choice response must be a number, got %q: %w", response, err)
	}
	return idx == c.Key, nil
}

// RecallConfig is the config for a recall-mode question: the text revealed
// after the learner self-grades.
type RecallConfig struct {
	RevealText string `json:"reveal_text"`
}

func (c RecallConfig) Mode() string { return "recall" }

// Validate checks that a recall config has non-empty reveal text.
func (c RecallConfig) Validate() error {
	if c.RevealText == "" {
		return fmt.Errorf("reveal_text must not be empty")
	}
	return nil
}

// Grade returns false — recall is self-graded on the client (Slice 3).
// The server does not grade recall responses.
func (c RecallConfig) Grade(response string) (bool, error) {
	return false, nil
}

// ParseConfig parses a Question's raw config JSON into the typed config
// selected by the question's mode.
func (q Question) ParseConfig() (QuestionConfig, error) {
	switch q.Mode {
	case "choice":
		var c ChoiceConfig
		if err := json.Unmarshal([]byte(q.Config), &c); err != nil {
			return nil, fmt.Errorf("parse choice config: %w", err)
		}
		return c, nil
	case "recall":
		var c RecallConfig
		if err := json.Unmarshal([]byte(q.Config), &c); err != nil {
			return nil, fmt.Errorf("parse recall config: %w", err)
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown question mode %q", q.Mode)
	}
}

// ParseItems parses a Quiz's raw items JSON into an ordered slug slice.
func (q Quiz) ParseItems() ([]string, error) {
	var items []string
	if err := json.Unmarshal([]byte(q.Items), &items); err != nil {
		return nil, fmt.Errorf("parse quiz items: %w", err)
	}
	return items, nil
}

// DisplayName returns the user-friendly topic if set, else the directory name.
// Used everywhere a human reads the name; URLs and keys must still use Name.
func (w Workspace) DisplayName() string {
	if w.Topic != "" {
		return w.Topic
	}
	return w.Name
}

// GlossaryTerm is a single term in a workspace's glossary.
type GlossaryTerm struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Term        string `db:"term" json:"term"`
	Definition  string `db:"definition" json:"definition"`
	Category    string `db:"category" json:"category"`
	Avoid       string `db:"avoid" json:"avoid"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// SearchResult is one result from a cross-entity, cross-workspace search.
// URLs are a presentation concern and are constructed by the caller from the
// fields here.
type SearchResult struct {
	Type           string `json:"type"` // "lesson" | "record" | "ref"
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Snippet        string `json:"snippet,omitempty"` // body content preview when summary is empty
	WorkspaceName  string `json:"workspaceName"`
	WorkspaceID    int64  `json:"-"`
	SequenceNumber int    `json:"sequenceNumber,omitempty"` // lessons and records
	Slug           string `json:"slug,omitempty"`           // refs only
}

// QuizScore is a Quiz with its best score across completed attempts.
// BestScore counts correct answers for questions still in the quiz's item
// list (so removed items don't inflate it); BestTotal is the current item
// count. Attempted is false when no completed attempt exists — the CLI shows
// "—" in that case.
type QuizScore struct {
	Quiz
	BestScore int  `json:"bestScore"`
	BestTotal int  `json:"bestTotal"`
	Attempted bool `json:"attempted"`
}

// QuizAttemptScore is one completed attempt scored against the quiz's current
// questions. Correct counts unique correct answers (latest per question wins)
// for questions still in the quiz; Total is the current item count. These
// scores use the same computation as QuizScore.BestScore, so the trend
// reconciles with `quiz list`'s best-score column.
type QuizAttemptScore struct {
	QuizAttempt
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

// WeakQuestionResult is a question with its accuracy from completed attempts.
// Accuracy is null (nil) when the question has never been answered in a
// completed attempt — these sort first (most urgent to assess).
// LastAttempted is the timestamp of the most recent answer ("" when never
// attempted) so the agent can tell stale weakness from fresh.
type WeakQuestionResult struct {
	Question
	Correct       int    `json:"correct"`
	Total         int    `json:"total"`
	HasData       bool   `json:"hasData"` // false = never answered in a completed attempt
	LastAttempted string `json:"lastAttempted,omitempty"`
}

// Settings holds user preferences.
type Settings struct {
	ID                  int64  `db:"id" json:"id"`
	DefaultView         string `db:"default_view" json:"defaultView"`
	ItemsPerPage        int    `db:"items_per_page" json:"itemsPerPage"`
	LessonsDir          string `db:"lessons_dir" json:"lessonsDir"`
	RecordsDir          string `db:"records_dir" json:"recordsDir"`
	ReferenceDir        string `db:"reference_dir" json:"referenceDir"`
	AssetsDir           string `db:"assets_dir" json:"assetsDir"`
	LastActiveWorkspace string `db:"last_active_workspace" json:"lastActiveWorkspace"`
}
