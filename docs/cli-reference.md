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
pharos stop                              # Stop the running web server
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
pharos mission read -w "<workspace>"                                 # Read mission
pharos mission read -w "<workspace>" --json                          # Read mission as JSON
pharos mission edit -w "<workspace>"                                 # Edit mission in $EDITOR
pharos mission edit -w "<workspace>" --body-file <path>               # Write mission from file

pharos resources read -w "<workspace>"                                # Read resources
pharos resources read -w "<workspace>" --json                         # Read resources as JSON
pharos resources edit -w "<workspace>"                                # Edit resources in $EDITOR
pharos resources edit -w "<workspace>" --body-file <path>             # Write resources from file

pharos notes read -w "<workspace>"                                    # Read notes
pharos notes read -w "<workspace>" --json                             # Read notes as JSON
pharos notes edit -w "<workspace>"                                    # Edit notes in $EDITOR
pharos notes edit -w "<workspace>" --body-file <path>                 # Write notes from file
pharos notes edit -w "<workspace>" --append --body-file <path>        # Append to notes

pharos glossary list                                                  # Show glossary
pharos glossary list --json                                           # Show glossary as JSON
pharos glossary create "<term>" "<definition>" -w "<workspace>"       # Add or update a term
pharos glossary create "<term>" "<definition>" --category "<name>"    # Group under a heading
pharos glossary create "<term>" "<definition>" --avoid "<synonym>"    # Flag a synonym to avoid
pharos glossary delete "<term>" -w "<workspace>"                      # Remove a term (idempotent)
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

## Search

```bash
pharos search "<query>"                          # Search across all workspaces
pharos search "<query>" -w "<workspace>"         # Search within one workspace
pharos search --rebuild-index                     # Index the current workspace's content
pharos search --rebuild-index --all               # Rebuild index across all workspaces
```

## Configuration

```bash
pharos config read                                # Read current configuration
pharos config set data_dir ~/my-pharos            # Change the data directory
```

## Skills

```bash
pharos skills install                 # Interactively install pharos skill into AI agent
pharos skills install --agent opencode  # Install for a specific agent
pharos skills install --project       # Install at project level (not global)
pharos skills check                   # Check installed skills and their status
```

## Maintenance

```bash
pharos upgrade                        # Upgrade pharos via 'go install ...@latest'
pharos tailwind download              # Download the Tailwind CLI binary to .bin/tailwindcss
pharos build                          # Rebuild CSS + Go binary
pharos build --no-css                 # Go-only build (skip CSS rebuild)
pharos dev                            # Hot-reload dev server
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

SQLite database at `~/.pharos/pharos.db` (configurable via `--db` or `config set data_dir`).

6 tables: `workspaces`, `lessons`, `learning_records`, `references_t`, `settings`, `glossary_terms`.
FTS5 full-text search (Porter tokenizer) on lessons, records, and references.
12 migrations (see `pharos migrate status`).
All mutations happen through the CLI — the web UI is read-only.
