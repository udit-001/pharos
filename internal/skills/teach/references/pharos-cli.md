# Pharos CLI — Reference

Command reference for the `pharos` CLI. Loaded when the teach skill needs the
mechanism; the teach skill decides _when_ and _why_ to run each.

## Workspace management

```bash
pharos workspace create "<name>"              # Create workspace (auto-sets as current)
pharos workspace create "<name>" --dir <path> # Create at a custom path
pharos workspace create "<name>" --topic "<friendly title>"  # Override the display title

pharos workspace use "<name>"                  # Set current workspace
pharos workspace current                       # Show current workspace
pharos workspace list                          # List all workspaces (current marked with *)
pharos workspace delete "<name>"               # Delete workspace + directory (prompts)
pharos workspace delete "<name>" --force       # Delete without prompt
pharos workspace rename "<new name>"            # Rename workspace (updates display name; slug stays)
pharos workspace rename "<new name>" -w "<name>" # Rename a specific workspace by -w
pharos workspace stats                         # Show learning statistics (lessons, records, quizzes)

pharos init                                    # Create the database
```

Before running `pharos workspace create`, check `pharos workspace list` for an existing
workspace on the same topic. If one exists, use `pharos workspace use` to switch to it — don't duplicate.

## Search

```bash
pharos search "<query>"                    # Cross-entity search across all workspaces
pharos search "<query>" --workspace <name> # Scope to one workspace
pharos search "<query>" --json             # Machine-readable output
pharos search --rebuild-index              # Rebuild search index for current workspace
pharos search --rebuild-index --all        # Rebuild for all workspaces
```

Searches lessons, learning records, and references by full-text match across title, summary, and body content. Results show type badges (Lesson/Record/Reference), workspace name, and a snippet when no summary is set. Use this before creating any entity to avoid duplicates.

## Current workspace

Most commands accept `-w "<name>"` to specify a workspace. If omitted, the
**current workspace** is used. Set it with:

```bash
pharos workspace use "<name>"
```

`pharos workspace create` auto-sets the new workspace as current. If only one
workspace exists, it is used automatically.

## Mission, Resources, Notes

```bash
pharos mission read                            # Print the file
pharos mission read --json                     # Print as JSON
pharos mission edit                            # Open in $EDITOR
pharos mission edit --body-file <path>         # Write content from a file

pharos resources read                          # Print the file
pharos resources read --json                   # Print as JSON
pharos resources edit                          # Open in $EDITOR
pharos resources edit --body-file <path>       # Write content from a file

pharos notes read                              # Print the file
pharos notes read --json                       # Print as JSON
pharos notes edit                              # Open in $EDITOR
pharos notes edit --body-file <path>           # Write content from a file
pharos notes edit --append --body-file <path>  # Append to file
```

## Glossary

```bash
pharos glossary list                              # Display glossary terms
pharos glossary list --json                       # Display glossary terms as JSON
pharos glossary create "<term>" "<definition>"     # Add or update a term
pharos glossary delete "<term>"                    # Remove a term
```

## Lessons

```bash
pharos lesson create "<title>" --body-file <path>  # Create a new lesson
pharos lesson list                                  # List lessons
pharos lesson list --search "<q>"                   # Search lessons
pharos lesson revise <seq> --body-file <path>       # Revise an existing lesson
pharos lesson show <seq>                            # Show in dashboard
pharos lesson read <seq>                            # Read content + metadata
pharos lesson read <seq> --meta-only                # Read metadata only
```

Before creating, search for an existing lesson on the same topic. If one
exists, **revise** it with `pharos lesson revise` rather than create a duplicate.

After creating or revising a lesson, **present** it:

```bash
pharos lesson show <seq>    # Starts dashboard if needed, opens in browser
```

## Learning records

```bash
pharos record create "<title>" --body-file <path>   # Create a learning record
pharos record create "<title>" --body-file <path> --summary "..."
pharos record list                                   # List records
pharos record list --search "<q>"                    # Search records
pharos record supersede <seq> --title "<new>" --body-file <path>  # Supersede with new understanding
pharos record show <seq>                             # Show in dashboard
pharos record read <seq>                             # Read content + metadata
pharos record read <seq> --meta-only                 # Read metadata only
```

Records follow the ADR convention: don't edit them, **supersede** them.
`pharos record supersede` atomically creates a new record AND marks the old one
as superseded.

## References

```bash
pharos reference create "<title>" --body-file <path>  # Create a reference (slug-based filename)
pharos reference list                                  # List references
pharos reference list --search "<q>"                   # Search references
pharos reference revise <slug> --body-file <path>      # Revise an existing reference
pharos reference show <slug>                           # Show in dashboard
pharos reference read <slug>                           # Read content + metadata
pharos reference read <slug> --meta-only               # Read metadata only
```

References are addressed by **slug** (descriptive name), not sequence number.
The slug is derived from the title (e.g. "SQL Join Cheat Sheet" → `sql-join-cheat-sheet`).
Two workspaces can each have a reference with the same slug.

## Questions

```bash
pharos question create "<title>" --mode choice|recall --body-file <path>  # Create a question (DB-only)
pharos question list                                                       # List questions
pharos question list --weak                                               # Sort by accuracy ascending (completed attempts only)
pharos question list --weak --limit 5 --json                               # Top 5 weakest, machine-readable
pharos question read <slug>                                                # Print metadata + config (correct option marked for choice)
```

Questions are DB-only (no file on disk) — the item bank a workspace's quizzes draw from.
A question's `--mode` selects its config shape and how `--body-file` is interpreted:

- **choice** — `--body-file` is JSON: `{"options": ["A","B","C","D"], "key": 2}` where `key` is the 0-based index of the correct answer.
- **recall** — `--body-file` is the reveal text shown after the learner self-grades.

The slug is derived from the title. Before creating, `pharos question list` to reuse existing questions across quizzes rather than duplicating.

`--weak` surfaces what the learner struggles with: questions never answered in a completed attempt sort first, then by accuracy ascending, with a `Last` column showing when each was last attempted so you can tell stale weakness from fresh. This is the workspace's storage-strength signal — use it to decide what to practice or teach next.

## Quizzes

```bash
pharos quiz create "<title>" --items "slug1,slug2" [--description "..."] [--lesson <seq>]  # Create a quiz from question slugs (DB-only)
pharos quiz list                                                          # List quizzes (with best score + lesson link)
pharos quiz list --weak                                                   # Sort by weakness: never-attempted first, then by best-score ratio ascending
pharos quiz list --weak --limit 5 --json                                  # Top 5 weakest, machine-readable
pharos quiz revise <slug> [--items "slug1,slug2"] [--lesson <seq>]        # Update items and/or lesson link (0 to unlink)
pharos quiz read <slug>                                                   # Print metadata + question slugs + lesson link
pharos quiz attempts <slug>                                               # Completed-attempt history + trend (is accuracy improving?)
pharos quiz show <slug>                                                    # Show in dashboard
```

Quizzes are DB-only ordered lists of question slugs, grouped under a title. The learner takes them in the dashboard (library → attempt → review); the CLI only authors them. `--items` is a comma-separated slug list in presentation order. `quiz list` shows the best score from completed attempts per quiz.

A quiz optionally links to the **lesson** whose skill it practices, via `--lesson <seq>` on `create` or `revise` (pass `0` to `revise` to unlink). The link is a soft reference by lesson sequence number (not a FK) — it is surfaced in `quiz read`, `quiz list`, `quiz attempts`, and `quiz show`, and in reverse via `lesson read` and `lesson list` (which show linked quiz slugs). See the [Skills](../SKILL.md#skills) section for why the link exists.

`quiz list --weak` is the skill-area weakness signal: never-attempted quizzes sort first (most urgent to assess), then by best-score ratio ascending — weakest skill area first. Use it alongside `question list --weak` (per-question) to decide what to practice or teach next.

`quiz attempts <slug>` shows the retake history with a trend summary (recent-half vs earlier-half average accuracy). It answers "is accuracy improving?" — the trajectory, where `--weak` is a snapshot. Per-attempt scores reconcile with `quiz list`'s best-score column (best = max of this series).

`quiz revise --items` blocks while the quiz has in-progress attempts — wait for them to complete or be abandoned first. `--lesson` does not block (it's metadata that doesn't affect a running attempt).

After creating or revising a quiz, **present** it:

```bash
pharos quiz show <slug>    # Opens the quiz intro page; the learner starts when ready
```

## Assets

```bash
pharos asset list                      # seeded / vendored / user, with add/redeploy hints
pharos asset create <filename> --body-file <path>  # author or overwrite a user component
pharos asset add <name>                # install a vendored or seeded asset (skip if present)
pharos asset redeploy <name>           # force-sync to the binary (overwrites)
pharos asset delete <filename>         # remove a file (no prompt)
```

Assets are raw files (CSS, JS, images, fonts) with no database tracking.
`redeploy` overwrites user edits to a file.

## Dashboard

```bash
pharos start              # Start the read-only web UI (default :9090)
pharos start --port 9090  # Custom port
pharos start --no-open    # Don't auto-open the browser
```

If already running, `pharos start` prints the URL and returns.
The dashboard opens on the current workspace if one is set.

## Skills

```bash
pharos skills install --agent <name>   # Install skills (non-interactive)
pharos skills check                    # Check if installed skills are current
```

Supported agents: `opencode`, `claude-code`, `codex`, `pi.dev`.
`--agent` implies non-interactive — no prompts, always overwrites.

## Links inside lesson HTML (iframe escape)

A lesson renders inside an **iframe** at `/api/lesson-html/<workspace>/<file>`,
so two link types resolve differently:

**Asset references** (stylesheets, scripts, images) resolve against the
iframe's URL — use a **root-relative** path:

```html
<link rel="stylesheet" href="assets/style.css">
```

Never use `../assets/style.css` — the `../` climbs above the iframe's
document root and returns a 404.

**Contextual links** (clicking to another dashboard page) must escape
the iframe to update the dashboard. Use an absolute route with
`target="_top"`. These are the exact dashboard URL patterns:

| Page | Route | Example |
|------|-------|---------|
| Workspace | `/workspace/{name}` | `/workspace/sql` |
| Mission | `/workspace/{name}/mission` | `/workspace/sql/mission` |
| Glossary | `/workspace/{name}/glossary` | `/workspace/sql/glossary` |
| Resources | `/workspace/{name}/resources` | `/workspace/sql/resources` |
| Notes | `/workspace/{name}/notes` | `/workspace/sql/notes` |
| Lesson | `/workspace/{name}/lesson/{seq}` | `/workspace/sql/lesson/1` |
| Learning Record | `/workspace/{name}/record/{seq}` | `/workspace/sql/record/2` |
| Reference | `/workspace/{name}/ref/{slug}` | `/workspace/sql/ref/join-syntax` |

```html
<a href="/workspace/sql/glossary" target="_top">Review the glossary</a>
<a href="/workspace/sql/lesson/3" target="_top">Next lesson</a>
<a href="/workspace/sql/ref/join-syntax" target="_top">SQL join cheat sheet</a>
```

Key rules:
- `{name}` is the workspace name (URL-encoded — spaces become `%20`)
- `{seq}` is the sequence number (1, 2, 3…), not the filename
- `{slug}` is the hyphenated reference slug (e.g. `sql-syntax`), not a number
- Never use relative links like `lessons/0002.html` — they load inside the iframe and lose the dashboard chrome

## Global flags

```bash
--json   # Machine-readable JSON output (all data-returning commands)
--db     # Custom database path (default: ~/.pharos/pharos.db)
```

## File naming (generated by the CLI)

- Lessons:    `0001-dash-case-name.html` (4-digit zero-padded sequence)
- Records:    `0001-dash-case-title.md`
- References: `<slug>.html` (descriptive, e.g. `sql-syntax.html`)
- Assets:     `<filename>` (you choose the name)

Sequence numbers are assigned by the CLI on creation — don't hand-number.

## Workspace layout

```
<name>/
├── MISSION.md            # Why — reason the user is learning
├── RESOURCES.md          # Curated knowledge sources and communities
├── NOTES.md              # User preferences and working notes
├── lessons/              # Self-contained lesson HTML files
├── learning-records/     # ADR-style records of what was learned
├── reference/            # Cheat sheets and reference documents
└── assets/               # Reusable components (style.css, etc.)
```

Questions, quizzes, and quiz attempts are **DB-only** — they have no file on disk and are managed entirely via the CLI. The web dashboard is read-only — all mutations go through the CLI.
