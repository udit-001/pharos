package db

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/udit-001/pharos/internal/extract"
)

const sourceDocColumns = `id, workspace_id, title, slug, filename, path, source_ext, format, method, sha256, pages, estimated_tokens, COALESCE(chapters, '[]'), COALESCE(text, ''), created_at, updated_at`

const sourceDocColumnsQualified = `sources.id, sources.workspace_id, sources.title, sources.slug, sources.filename, sources.path, sources.source_ext, sources.format, sources.method, sources.sha256, sources.pages, sources.estimated_tokens, COALESCE(sources.chapters, '[]'), COALESCE(sources.text, ''), sources.created_at, sources.updated_at`

// ErrSourceDocNotFound is returned when a source document lookup misses.
var ErrSourceDocNotFound = errors.New("source document not found")

func scanSourceDoc(row interface{ Scan(...any) error }) (SourceDoc, error) {
	var d SourceDoc
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.Title, &d.Slug, &d.Filename, &d.Path,
		&d.SourceExt, &d.Format, &d.Method, &d.SHA256, &d.Pages, &d.EstimatedTokens,
		&d.Chapters, &d.Text, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func scanSourceDocs(rows RowScanner) ([]SourceDoc, error) {
	return scanRows(rows, "source document", scanSourceDoc)
}

// safeSourcePath resolves filename to an absolute path inside the workspace's
// sources/ directory, rejecting traversal. Mirrors safeAssetPath.
func (w *WorkspaceStore) safeSourcePath(filename string) (string, error) {
	return w.Layout().SafeJoin("sources", filename)
}

// CreateSourceDoc ingests one local file into this workspace. It is the one
// deep method that owns the whole invariant: extract + sanitize, copy the raw
// file into sources/, hash it, record metadata + the chapter map, and persist
// the sanitized text (which the FTS trigger indexes). The raw file is NEVER
// discarded. Commit happens only after the file copy succeeds, so a failed
// copy leaves no orphan row. Returns the underlying extraction error
// (unreadable file, unsupported format, XXE rejection) wrapped.
func (w *WorkspaceStore) CreateSourceDoc(absFile, title string) (SourceDoc, error) {
	if strings.TrimSpace(absFile) == "" {
		return SourceDoc{}, fmt.Errorf("source file path is required")
	}
	res, err := extract.FromFile(absFile)
	if err != nil {
		return SourceDoc{}, fmt.Errorf("extract source: %w", err)
	}
	if strings.TrimSpace(title) == "" {
		title = baseOf(absFile)
	}

	raw, err := os.ReadFile(absFile)
	if err != nil {
		return SourceDoc{}, fmt.Errorf("read source: %w", err)
	}
	sum := sha256Sum(raw)
	ext := filepath.Ext(absFile)
	slug := Slugify(title)
	filename := slug + ext
	target, err := w.safeSourcePath(filename)
	if err != nil {
		return SourceDoc{}, err
	}
	// Idempotent re-ingest: if the same stored filename was already ingested,
	// return the existing record when the content matches (a re-extract is a
	// no-op); different content under the same title is a conflict.
	if existing, scanErr := scanSourceDoc(w.db().QueryRow(
		"SELECT "+sourceDocColumns+" FROM source_documents WHERE workspace_id = ? AND filename = ?", w.ws.ID, filename,
	)); scanErr == nil {
		if existing.SHA256 == sum {
			return existing, nil
		}
		return SourceDoc{}, fmt.Errorf("source %q already ingested with different content (delete it first)", filename)
	}

	chaptersJSON, _ := json.Marshal(res.Chapters)
	now := nowTimestamp()

	tx, err := w.db().Begin()
	if err != nil {
		return SourceDoc{}, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`INSERT INTO source_documents
		   (workspace_id, title, slug, filename, path, source_ext, format, method,
		    sha256, pages, estimated_tokens, chapters, text, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.ws.ID, title, slug, filename, w.Layout().SourceRelPath(filename), ext,
		string(res.Format), res.Method, sum, res.Pages, res.EstimatedTokens,
		chaptersJSON, res.Text, now, now,
	)
	if err != nil {
		return SourceDoc{}, fmt.Errorf("insert source document: %w", err)
	}
	id, _ := result.LastInsertId()

	if err := writeToFile(target, string(raw)); err != nil {
		return SourceDoc{}, fmt.Errorf("write source file: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return SourceDoc{}, err
	}
	return w.GetSourceDocByID(id)
}

// GetSourceDocs returns all source documents in this workspace, by title.
// The full extracted text is omitted from the slice to keep listing cheap;
// use GetSourceText/GetSourceChapter to pull body content on demand.
func (w *WorkspaceStore) GetSourceDocs() ([]SourceDoc, error) {
	rows, err := w.db().Query("SELECT "+sourceDocColumns+" FROM source_documents WHERE workspace_id = ? ORDER BY title", w.ws.ID)
	if err != nil {
		return nil, fmt.Errorf("query source documents: %w", err)
	}
	defer rows.Close()
	docs, err := scanSourceDocs(rows)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		return []SourceDoc{}, nil
	}
	return docs, nil
}

// GetSourceDocByID returns a single source document by its workspace-scoped ID.
func (w *WorkspaceStore) GetSourceDocByID(id int64) (SourceDoc, error) {
	row := w.db().QueryRow("SELECT "+sourceDocColumns+" FROM source_documents WHERE id = ? AND workspace_id = ?", id, w.ws.ID)
	doc, err := scanSourceDoc(row)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return SourceDoc{}, ErrSourceDocNotFound
		}
		return SourceDoc{}, err
	}
	return doc, nil
}

// GetSourceText returns the full sanitized text of a source document — the
// agent's escape hatch for reading a whole source. Not user searchable.
func (w *WorkspaceStore) GetSourceText(id int64) (string, error) {
	var text string
	err := w.db().QueryRow(
		"SELECT text FROM source_documents WHERE id = ? AND workspace_id = ?", id, w.ws.ID,
	).Scan(&text)
	if err != nil {
		return "", ErrSourceDocNotFound
	}
	return text, nil
}

// GetSourceChapter returns one bounded chapter slice of a source document.
// Chapter numbers are 1-based; when no chapters were detected the whole text
// is the single chapter 1. Returns ErrSourceDocNotFound when the doc is
// unknown and an error when n is out of range.
func (w *WorkspaceStore) GetSourceChapter(id int64, n int) (SourceChapter, error) {
	doc, err := w.GetSourceDocByID(id)
	if err != nil {
		return SourceChapter{}, err
	}
	chapters, err := parseSourceChapters(doc.Chapters)
	if err != nil {
		return SourceChapter{}, err
	}
	if len(chapters) == 0 {
		chapters = []extract.Chapter{{Offset: 0}}
	}
	if n < 1 || n > len(chapters) {
		return SourceChapter{}, fmt.Errorf("chapter %d out of range (1..%d)", n, len(chapters))
	}
	runes := []rune(doc.Text)
	start := chapters[n-1].Offset
	end := len(runes)
	if n < len(chapters) {
		end = chapters[n].Offset
	}
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	return SourceChapter{Title: chapters[n-1].Title, Segment: string(runes[start:end])}, nil
}

// GetSourceChapters returns the ordered chapter boundaries (title + rune
// offset) of a source document, so the agent can discover chapter count/titles
// and pull a specific chapter with --chapter N.
func (w *WorkspaceStore) GetSourceChapters(id int64) ([]extract.Chapter, error) {
	doc, err := w.GetSourceDocByID(id)
	if err != nil {
		return nil, err
	}
	ch, err := parseSourceChapters(doc.Chapters)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		ch = []extract.Chapter{}
	}
	return ch, nil
}

// DeleteSourceDoc removes a source document: its DB row (FTS trigger cleans
// the index) and its raw file. Deleting an unknown doc is not an error.
func (w *WorkspaceStore) DeleteSourceDoc(id int64) error {
	doc, err := w.GetSourceDocByID(id)
	if err != nil {
		if errors.Is(err, ErrSourceDocNotFound) {
			return nil
		}
		return err
	}
	tx, err := w.db().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM source_documents WHERE id = ? AND workspace_id = ?", id, w.ws.ID); err != nil {
		return fmt.Errorf("delete source document: %w", err)
	}
	if target, err := w.safeSourcePath(doc.Filename); err == nil {
		_ = os.Remove(target) // best-effort file cleanup
	}
	return tx.Commit()
}

// QuerySources is the AGENT-ONLY source retrieval surface (Option A). It runs
// the existing FTS5 machinery over sources_fts — NOT user search — and
// returns provenanced passages {source, chapter, excerpt} so the teach skill
// can ground lesson claims on a cited excerpt. This is deliberately NOT wired
// into WorkspaceStore.Search / Store.Search / searchResultURL.
func (w *WorkspaceStore) QuerySources(query string) ([]SourceHit, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []SourceHit{}, nil
	}
	rows, err := w.db().Query(
		`SELECT source_documents.id, source_documents.title, source_documents.chapters, source_documents.text,
		       snippet(sources_fts, 1, '⟦', '⟧', '⋯', 45) AS snip
		   FROM sources_fts
		   JOIN source_documents ON source_documents.id = sources_fts.rowid
		  WHERE sources_fts MATCH ? AND source_documents.workspace_id = ?
		  ORDER BY bm25(sources_fts, '10.0, 1.0'), source_documents.id`,
		q, w.ws.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()

	var hits []SourceHit
	for rows.Next() {
		var id int64
		var title, chaptersJSON, text, snip string
		if err := rows.Scan(&id, &title, &chaptersJSON, &text, &snip); err != nil {
			return nil, fmt.Errorf("scan source hit: %w", err)
		}
		// Attribute the chapter by locating the matched SURFACE word in the
		// text: it survives porter stemming even when the typed term is not a
		// literal substring (query "running" matching "runs"). The FTS snippet
		// reveals that surface word; fall back to the literal term otherwise.
		off := -1
		if surf := matchedSurface(snip); surf != "" {
			off = findRuneOffset(text, surf)
		}
		if off < 0 {
			off = findTermRuneOffset(text, query)
		}
		// The FTS snippet is a stem-aware window around the real match — never
		// a misleading document-start slice.
		excerpt := cleanSnippet(snip)
		if excerpt == "" {
			excerpt = excerptAround([]rune(text), off)
		}
		hit := SourceHit{
			SourceID: id,
			Title:    title,
			Chapter:  chapterTitleAt(chaptersJSON, off),
			Excerpt:  excerpt,
		}
		hits = append(hits, hit)
	}
	if hits == nil {
		return []SourceHit{}, nil
	}
	return hits, rows.Err()
}

// snippetOpen / snippetClose delimit matches inside the FTS snippet() output.
const (
	snippetOpen  = "⟦"
	snippetClose = "⟧"
)

// snippetRe extracts the first ⟦...⟧ match (the matched surface word) from an
// FTS5 snippet() string.
var snippetRe = regexp.MustCompile(snippetOpen + `([^` + snippetClose + `]+)` + snippetClose)

// matchedSurface returns the first highlighted surface word from an FTS5
// snippet(), or "" if none is highlighted.
func matchedSurface(snip string) string {
	m := snippetRe.FindStringSubmatch(snip)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// cleanSnippet removes the match delimiters from an FTS5 snippet().
func cleanSnippet(snip string) string {
	return strings.ReplaceAll(strings.ReplaceAll(snip, snippetOpen, ""), snippetClose, "")
}

// parseSourceChapters parses the stored chapter-map JSON into typed offsets.
func parseSourceChapters(raw string) ([]extract.Chapter, error) {
	var chapters []extract.Chapter
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return chapters, nil
	}
	if err := json.Unmarshal([]byte(raw), &chapters); err != nil {
		return nil, fmt.Errorf("parse chapters: %w", err)
	}
	return chapters, nil
}

// chapterTitleAt returns the title of the chapter whose Offset bounds runeOff.
func chapterTitleAt(chaptersJSON string, runeOff int) string {
	chapters, err := parseSourceChapters(chaptersJSON)
	if err != nil || len(chapters) == 0 {
		return ""
	}
	title := ""
	for _, c := range chapters {
		if c.Offset <= runeOff {
			title = c.Title
		} else {
			break
		}
	}
	return title
}

// findRuneOffset returns the rune offset of the first case-insensitive
// occurrence of surface in text, or -1.
func findRuneOffset(text, surface string) int {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(surface))
	if idx < 0 {
		return -1
	}
	return len([]rune(text[:idx]))
}

// findTermRuneOffset returns the rune offset of the first case-insensitive
// occurrence of any query term in text, or -1 if none is found. A fallback
// for when no highlighted surface word is available.
func findTermRuneOffset(text, query string) int {
	for _, tok := range strings.Fields(query) {
		if off := findRuneOffset(text, tok); off >= 0 {
			return off
		}
	}
	return -1
}

// excerptAround builds a bounded ~280-rune excerpt around runeOff, breaking at
// whitespace. Falls back to a leading snippet when off is -1.
func excerptAround(runes []rune, off int) string {
	if off < 0 {
		off = 0
	}
	start := off - 120
	if start < 0 {
		start = 0
	}
	end := off + 160
	if end > len(runes) {
		end = len(runes)
	}
	s := strings.TrimSpace(string(runes[start:end]))
	if s == "" {
		return ""
	}
	if start > 0 {
		s = "…" + s
	}
	if end < len(runes) {
		s = s + "…"
	}
	return s
}

// sha256Sum returns the hex-encoded SHA-256 of data.
func sha256Sum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// baseOf returns the base filename (without directory) of a path.
func baseOf(path string) string {
	if b := filepath.Base(path); b != "" && b != "." && b != string(filepath.Separator) {
		return b
	}
	return path
}
