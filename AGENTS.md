# Pharos

CLI + read-only web dashboard for AI-guided learning workspaces.

## Commands

- `make test` — `go test ./...` (real SQLite temp files, no mocks)
- `make build` — builds binary (runs `make css` first)

## Conventions

- HTML is rendered via `fmt.Sprintf` in `internal/render/` — no templates.
- Tailwind v4 standalone CLI: edit `web/input.css`, then `make css`. CSS is `//go:embed`'d and committed.
- Each Cobra command lives in its own file under `internal/cli/`.
- Version lives in `internal/version/Version` as a `var`, overridable via ldflags or auto-detected from `debug.BuildInfo`.
- Repo: `github.com/udit-001/pharos`

## Lessons learned

### File locations

- **Teach skill lives at `internal/skills/teach/`**, not `.opencode/skills/teach/`. Installed copies go to `.opencode/skills/` or `~/.config/opencode/skills/` — source of truth is `internal/`.
- **Learning records file is `learning_records.go`**, not `records.go`.

### Build workflow

- **Run `make css` after editing `web/input.css`**, then rebuild the binary. Tailwind v4 scans `.go` files for classes; stale `app.css` is the #1 cause of missing styles. CSS is embedded via `//go:embed web/app.css` — binary must be rebuilt.
- **Run `pharos stop && make build && pharos start`** after any rebuild. `pharos start` detects a running server via HTTP GET and skips starting — the old binary keeps serving.

### Edit discipline

- **Check for duplicate matches before editing.** When inserting a new section, the `oldString` may match text that was just inserted by a prior edit in the same turn. Verify the file state after each edit.
- **Read the file after editing** to confirm the result is correct, especially when inserting near existing similar content.

### Code patterns

- `goquery` parses HTML and extracts text. When extracting body text from lesson HTML, **strip `<head>`, `<script>`, `<style>`, and `<noscript>` tags first** — otherwise their content contaminates the extracted text.
- `extractText()` is the shared helper in `workspace_store.go` for HTML→plaintext. `extractTextFromMarkdown()` handles markdown→plaintext without a goldmark roundtrip.
- `IndexLessons()` / `IndexRefs()` / `IndexRecords()` skip items that already have non-empty `body_text` — they're idempotent. To re-index after an extractText fix, clear body_text first: `UPDATE lessons SET body_text = ''`.
- `pharos search index --all` rebuilds across all workspaces.
