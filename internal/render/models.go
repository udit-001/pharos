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
	Name                            string
	Topic                           string
	LessonCount, RecordCount, RefCount int
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

// Sidebar is the workspace tree shown in the left rail. When Workspace is
// nil, an empty-state prompt is shown instead.
type Sidebar struct {
	Workspace *Workspace
	Lessons   []LessonEntry
	Records   []RecordEntry
	Refs      []RefEntry
}

// FrameContent reports whether the page body should fill the viewport as an
// iframe frame (lesson/ref pages) rather than scroll. Derived from ActiveType.
func (f Frame) FrameContent() bool {
	return f.ActiveType == "lesson" || f.ActiveType == "ref"
}

// ── Page-specific view models ──

// Stats is the dashboard totals row.
type Stats struct {
	Workspaces, Lessons, Records, Refs int
}

// ContinueItem is the "continue where you left off" card. Nil = hidden.
type ContinueItem struct {
	URL, Label string
}

// WorkspaceCard is one tile in the dashboard workspace grid.
type WorkspaceCard struct {
	Name                                 string
	Topic                                string // friendly display title; empty falls back to Name
	LessonCount, RecordCount, RefCount   int
	LastStudied                          string
}

// DashboardData drives the dashboard page.
type DashboardData struct {
	Stats      Stats
	Continue   *ContinueItem
	Workspaces []WorkspaceCard
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
	Title   string
	Status  string // "active" | "superseded"
	BodyHTML string
}

// RefData drives a reference detail page (iframe).
type RefData struct {
	Title  string
	RawURL string
}

// DocumentData drives workspace document pages (Mission, Resources, Glossary, Notes).
type DocumentData struct {
	Kind     string // "mission", "resources", "glossary", "notes"
	Title    string
	BodyHTML string
	Empty    bool
}

// SearchResult is one row on the search page.
type SearchResult struct {
	Type        string // "lesson" | "record" | "ref"
	Title       string
	URL         string
	Workspace   string
	Summary     string
	Snippet     string // body content preview when summary is empty
}

// SearchData drives the search page.
type SearchData struct {
	Query   string
	Results []SearchResult
}
