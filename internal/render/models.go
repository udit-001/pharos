package render

// Frame is the page chrome: what the shell needs to render the topbar,
// sidebar, and document wrapper. Handlers build a Frame plus page-specific
// content data, then call Page(frame, content).
type Frame struct {
	Title       string // document <title> and topbar heading
	ActiveWS    string // active workspace name (for breadcrumb + sidebar highlight)
	ActiveType  string // "", "lesson", "record", "ref" (sidebar highlight)
	ActiveSeq   int    // active sequence number (sidebar highlight for lesson/record)
	ActiveSlug  string // active slug (sidebar highlight for ref)
	SearchQuery string // current search query for search page
	Sidebar     Sidebar
}

// Workspace is the render-owned projection of a workspace. It carries only
// the fields render needs (name, topic, counts) — the server adapter copies
// from db.Workspace so render never imports db.
type Workspace struct {
	Name                                          string
	Topic                                         string
	LessonCount, RecordCount, RefCount, QuizCount int
}

// LessonEntry is the render-owned projection of a lesson for sidebar and
// workspace page lists.
type LessonEntry struct {
	Seq   int
	Title string
}

// RecordEntry is the render-owned projection of a learning record.
type RecordEntry struct {
	Seq     int
	Title   string
	Status  string
	Summary string
}

// RefEntry is the render-owned projection of a reference.
type RefEntry struct {
	Slug  string
	Title string
}

// QuizEntry is the render-owned projection of a quiz.
type QuizEntry struct {
	Slug        string
	Title       string
	Description string
	ItemCount   int
	// BestScore is the highest score across all completed attempts, or -1 if none.
	BestScore int
	// BestTotal is the total questions for the best attempt.
	BestTotal int
}

// Sidebar is the workspace tree shown in the left rail. When Workspace is
// nil, an empty-state prompt is shown instead.
type Sidebar struct {
	Workspace *Workspace
	Lessons   []LessonEntry
	Records   []RecordEntry
	Refs      []RefEntry
	Quizzes   []QuizEntry
}

// FrameContent reports whether the page body should fill the viewport as an
// iframe frame (lesson/ref pages) rather than scroll. Derived from ActiveType.
func (f Frame) FrameContent() bool {
	return f.ActiveType == "lesson" || f.ActiveType == "ref"
}

// ── Page-specific view models ──

// Stats is the dashboard totals row.
type Stats struct {
	Workspaces, Lessons, Records, Refs, Quizzes int
}

// ContinueItem is the "continue where you left off" card. Nil = hidden.
type ContinueItem struct {
	URL, Label string
}

// WorkspaceCard is one tile in the dashboard workspace grid.
type WorkspaceCard struct {
	Name                                          string
	Topic                                         string
	LessonCount, RecordCount, RefCount, QuizCount int
	LastStudied                                   string
}

// DashboardData drives the dashboard page.
type DashboardData struct {
	Stats      Stats
	Continue   *ContinueItem
	Workspaces []WorkspaceCard
	QuizWidget *QuizWidgetData
}

// QuizWidgetData drives the dashboard quiz widget.
type QuizWidgetData struct {
	RecentCompleted *QuizWidgetItem
	InProgress      []QuizWidgetItem
}

// QuizWidgetItem is one entry in the dashboard quiz widget.
type QuizWidgetItem struct {
	WorkspaceName string
	QuizTitle     string
	URL           string
	Score         int
	Total         int
}

// WorkspaceData drives a workspace landing page.
type WorkspaceData struct {
	Workspace Workspace
	Mission   string
	Lessons   []LessonEntry
	Records   []RecordEntry
	Refs      []RefEntry
}

// LessonData drives a lesson detail page (iframe).
type LessonData struct {
	Title  string
	RawURL string
	Seq    int
	Total  int
}

// RecordData drives a learning-record detail page (rendered markdown).
type RecordData struct {
	Title    string
	Status   string // "active" | "superseded"
	BodyHTML string
}

// RefData drives a reference detail page (iframe).
type RefData struct {
	Title  string
	RawURL string
}

// QuizLibraryData drives the quiz library page: a list of quizzes.
type QuizLibraryData struct {
	Workspace Workspace
	Quizzes   []QuizEntry
	// InProgress is a quiz with an active attempt, shown as a resume banner.
	InProgress *QuizResumeLink
}

// QuizResumeLink is the in-progress banner link.
type QuizResumeLink struct {
	AttemptID int64
	QuizSlug  string
	QuizTitle string
	Scored    int // answered so far
	Total     int // total questions in quiz
}

// QuizData drives a quiz detail page: title, description, item count, and a
// Start button. Attempt display is deferred to a later slice.
type QuizData struct {
	Workspace   Workspace
	Slug        string
	Title       string
	Description string
	ItemCount   int
	// InProgressAttempt is the ID of a resumable attempt, or 0 if none.
	InProgressAttempt int64
	// PastAttempts shows completed attempts with scores, newest first.
	PastAttempts []QuizAttemptSummary
	// ExtraAttemptCount is the number of completed attempts beyond the 3 shown.
	ExtraAttemptCount int
}

// QuizAttemptSummary is one row in the quiz detail attempt list.
type QuizAttemptSummary struct {
	ID          int64
	Score       int
	Total       int
	CompletedAt string
}

// AttemptQuestion is one question in the attempt page, embedded as JSON.
// Options is empty for recall mode. Reveal is empty for choice mode.
// The correct answer is NOT included.
type AttemptQuestion struct {
	ID      int64    `json:"id"`
	Title   string   `json:"title"`
	Mode    string   `json:"mode"`
	Options []string `json:"options,omitempty"`
	Reveal  string   `json:"reveal,omitempty"`
}

// AttemptData drives the quiz attempt page.
type AttemptData struct {
	Workspace Workspace
	QuizSlug  string
	QuizTitle string
	AttemptID int64
	Questions []AttemptQuestion
	// AnsweredIDs is the set of question IDs already answered (for resume).
	AnsweredIDs map[int64]bool
	// AnsweredResults maps answered question ID → correct/incorrect.
	AnsweredResults map[int64]bool
}

// ReviewItem is one question in the review page.
type ReviewItem struct {
	QuestionID    int64
	QuestionTitle string
	Mode          string
	Options       []string
	UserResponse  string
	CorrectIndex  int // for choice mode
	IsCorrect     bool
	RevealText    string // for recall mode
}

// QuizReviewData drives the quiz review page.
type QuizReviewData struct {
	Workspace Workspace
	QuizSlug  string
	QuizTitle string
	AttemptID int64
	Score     int
	Total     int
	Items     []ReviewItem
}

// GlossaryTermRow is one term in the glossary page.
type GlossaryTermRow struct {
	Term       string
	Definition string
	Category   string
	Avoid      string
}

// DocumentData drives workspace document pages (Mission, Resources, Glossary, Notes).
type DocumentData struct {
	Kind          string // "mission", "resources", "glossary", "notes"
	Title         string
	BodyHTML      string
	Empty         bool
	GlossaryTerms []GlossaryTermRow // set for kind=glossary
}

// SearchResult is one row on the search page.
type SearchResult struct {
	Type      string // "lesson" | "record" | "ref" | "quiz"
	Title     string
	URL       string
	Workspace string
	Summary   string
	Snippet   string // body content preview when summary is empty
}

// SearchData drives the search page.
type SearchData struct {
	Query   string
	Results []SearchResult
}
