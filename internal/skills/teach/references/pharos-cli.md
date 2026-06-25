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
pharos workspace stats                         # Show learning statistics

pharos init                                    # Create the database
```

Before running `pharos workspace create`, check `pharos workspace list` for an existing
workspace on the same topic. If one exists, use `pharos workspace use` to switch to it — don't duplicate.

## Search

```bash
pharos search "<query>"                    # Cross-entity search across all workspaces
pharos search "<query>" --workspace <name> # Scope to one workspace
pharos search "<query>" --json             # Machine-readable output
pharos search index                        # Rebuild search index for current workspace
pharos search index --all                  # Rebuild for all workspaces
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

## Singletons (one per workspace, flags not subcommands)

```bash
pharos mission    [--edit | --body-file <path>]
pharos resources  [--edit | --body-file <path>]
pharos glossary                                    # Display glossary terms
pharos glossary add "<term>" "<definition>"        # Add or update a term
pharos glossary delete "<term>"                    # Remove a term
pharos notes      [--edit | --body-file <path>] [--append]
```

- `--edit`: open in `$EDITOR` (mission, resources, notes only)
- `--body-file <path>`: write content from a file (mission, resources, notes only)
- `--append` (notes only): append instead of overwrite
- No flags: read and print

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

## Assets

```bash
pharos asset list                     # List assets with absolute paths
pharos asset create <filename> --body-file <path>  # Create or overwrite an asset
```

Assets are raw files (CSS, JS, images) with no database tracking.
Use `pharos asset create` for all mutations — don't write directly.

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

The web dashboard is read-only — all mutations go through the CLI.
