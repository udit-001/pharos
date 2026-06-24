# CLI Reference

## Initialize

```bash
pharos init                              # Create database and run migrations
pharos init --force                      # Recreate database from scratch
```

## Dev Server (hot-reload)

```bash
pharos dev                               # Start dev server with live Go + CSS rebuild
pharos dev --port 9090                   # Custom port
pharos dev --no-open                     # Don't auto-open browser
```

## Web UI

```bash
pharos start                             # Start read-only web dashboard (default :9090)
pharos start --port 9090                 # Custom port (auto-increments if busy)
pharos start --no-open                   # Don't auto-open browser
pharos start --foreground / -f           # Run in foreground
pharos start --background / -b           # Run in background (default)
pharos start --dev-css                   # Serve CSS from disk (dev mode)
```

## Workspaces

```bash
pharos workspace create "<name>"         # Create a new workspace
pharos workspace create "<name>" --dir <path>    # Create at custom path
pharos workspace create "<name>" --topic "<title>"  # Override display title

pharos workspace list                    # List all workspaces
pharos workspace stats                   # Show learning statistics (with bar charts)
pharos workspace use "<name>"            # Set as current workspace
pharos workspace current                 # Show current workspace
pharos workspace delete "<name>"         # Delete workspace and directory
pharos workspace delete "<name>" --force # Skip confirmation prompt
```

## Lessons

```bash
pharos lesson create "<title>" -w "<workspace>" --body-file <path>   # Create lesson with content
pharos lesson list -w "<workspace>"                                   # List lessons
pharos lesson list -w "<workspace>" --search "<q>"                    # Search lessons
pharos lesson read <seq> -w "<workspace>"                             # Read lesson content
pharos lesson read <seq> -w "<workspace>" --meta-only                 # Show metadata only
pharos lesson show <seq> -w "<workspace>"                             # Open in web dashboard
pharos lesson revise <seq> -w "<workspace>" --body-file <path>        # Revise lesson content
pharos lesson revise <seq> -w "<workspace>" --title "<new>"           # Update lesson title
pharos lesson revise <seq> -w "<workspace>" --summary "<new>"         # Update lesson summary
```

## Learning Records

```bash
pharos record create "<title>" -w "<workspace>" --body-file <path>    # Create a learning record
pharos record create "<title>" -w "<workspace>" --body-file <path> \  # With summary
  --summary "..."
pharos record list -w "<workspace>"                                    # List records
pharos record list -w "<workspace>" --search "<q>"                     # Search records
pharos record read <seq> -w "<workspace>"                              # Read record content
pharos record read <seq> -w "<workspace>" --meta-only                  # Show metadata only
pharos record show <seq> -w "<workspace>"                              # Open in web dashboard
pharos record supersede <seq> -w "<workspace>" --title "<new>" \      # Supersede with new understanding
  --body-file <path>
```

## References

```bash
pharos reference create "<title>" -w "<workspace>" --body-file <path>  # Create a reference
pharos reference list -w "<workspace>"                                  # List references
pharos reference list -w "<workspace>" --search "<q>"                   # Search references
pharos reference read <slug> -w "<workspace>"                           # Read reference content
pharos reference read <slug> -w "<workspace>" --meta-only               # Show metadata only
pharos reference show <slug> -w "<workspace>"                           # Open in web dashboard
pharos reference revise <slug> -w "<workspace>" --body-file <path>     # Revise reference content
pharos reference revise <slug> -w "<workspace>" --title "<new>"        # Update reference title
pharos reference revise <slug> -w "<workspace>" --summary "<new>"      # Update reference summary
```

## Workspace Documents

```bash
pharos mission -w "<workspace>"                   # Show mission
pharos mission -w "<workspace>" --edit / -e       # Edit mission in $EDITOR
pharos mission -w "<workspace>" --body-file <path> # Write mission from file

pharos resources -w "<workspace>"                  # Show resources
pharos resources -w "<workspace>" --edit / -e      # Edit resources
pharos resources -w "<workspace>" --body-file <path>

pharos glossary -w "<workspace>"                   # Show glossary
pharos glossary -w "<workspace>" --edit / -e        # Edit glossary
pharos glossary -w "<workspace>" --body-file <path>

pharos notes -w "<workspace>"                      # Show notes (scratchpad)
pharos notes -w "<workspace>" --edit / -e           # Edit notes
pharos notes -w "<workspace>" --body-file <path>    # Write notes from file
pharos notes -w "<workspace>" --append              # Append to notes instead of overwriting
```

## Assets

```bash
pharos asset list -w "<workspace>"                  # List workspace assets
pharos asset create <filename> -w "<workspace>" --body-file <path>  # Create or overwrite asset file
```

## Migrations

```bash
pharos migrate up                     # Apply all pending migrations
pharos migrate down                   # Roll back most recent migration
pharos migrate up-to <version>        # Run migrations up to a specific version
pharos migrate down-to <version>      # Roll back to a specific version
pharos migrate status                 # Show migration status
```

## Skills

```bash
pharos skills install                 # Interactively install pharos skill into AI agent
pharos skills install --agent opencode  # Install for a specific agent
pharos skills install --project       # Install at project level (not global)
pharos skills check                   # Check installed skills and their status
```

## Global Flags

```bash
--json      # Machine-readable JSON output (most commands)
--db        # Custom database path (default: ~/.pharos/pharos.db)
```

## File Naming

The CLI generates filenames automatically from titles:

| Type       | Pattern                        | Example                         |
|------------|--------------------------------|---------------------------------|
| Lesson     | `0001-dash-case-name.html`      | `0001-sql-joins.html`           |
| Record     | `0001-dash-case-title.md`       | `0001-understood-inner-join.md` |
| Reference  | Slug-based (from title)         | `notation-cheat-sheet.html`     |

## Workspace Layout

```
<name>/
├── MISSION.md            # Why you're learning
├── RESOURCES.md          # Curated sources and communities
├── GLOSSARY.md           # Canonical terminology (built over time)
├── NOTES.md              # Preferences and working notes (scratchpad)
├── lessons/              # Self-contained lesson HTML files
├── learning-records/     # ADR-style records of what was learned
├── reference/            # Cheat sheets and reference documents
└── assets/               # Reusable components (style.css, etc.)
```

## Data

SQLite database at `~/.pharos/pharos.db` (configurable via `--db`).

4 tables: `workspaces`, `lessons`, `learning_records`, `references_t`.
FTS5 full-text search on lessons, records, and references.
All mutations happen through the CLI — the web UI is read-only.
