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
pharos workspace rename "<new name>"     # Update display name (directory slug unchanged)
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
pharos asset list -w "<workspace>"                  # List workspace assets (seeded, vendored, user)
pharos asset create <filename> -w "<workspace>" --body-file <path>  # Create or overwrite asset file
pharos asset add <name> -w "<workspace>"            # Install a vendored/seeded asset (skips if present)
pharos asset redeploy <name> -w "<workspace>"       # Force-sync asset to current binary version
pharos asset delete <filename> -w "<workspace>"     # Remove an asset file
```

## Questions

Questions are DB-only entities used to build quizzes. Two modes:
- **choice** — `--body-file` is JSON `{"options": [...], "key": N}` (0-based correct index)
- **recall** — `--body-file` is the reveal text shown after self-grading

```bash
pharos question create "<title>" -w "<ws>" --mode choice --body-file <path>   # Create a choice question
pharos question create "<title>" -w "<ws>" --mode recall --body-file <path>   # Create a recall question
pharos question create "<title>" -w "<ws>" --mode choice --body-file <path> --stimulus-file <html>  # With stimulus

pharos question list -w "<workspace>"               # List questions
pharos question list -w "<workspace>" --weak        # Sort by accuracy ascending (struggles first)
pharos question list -w "<workspace>" --limit N     # Max results
pharos question read <slug> -w "<workspace>"        # Read question content and metadata
pharos question revise <slug> -w "<workspace>" --title "<new>"       # Update title
pharos question revise <slug> -w "<workspace>" --body-file <path>    # Update config
pharos question revise <slug> -w "<workspace>" --mode recall --body-file <path>  # Change mode
pharos question revise <slug> -w "<workspace>" --stimulus-file <path>  # Add/replace stimulus
pharos question revise <slug> -w "<workspace>" --clear-stimulus       # Remove stimulus
pharos question delete <slug> -w "<workspace>"      # Delete (blocks if a quiz references it)
```

## Quizzes

Quizzes are DB-only ordered lists of question slugs. Taken in the web dashboard.

```bash
pharos quiz create "<title>" -w "<ws>" --items "slug1,slug2"             # Create a quiz
pharos quiz create "<title>" -w "<ws>" --items "slug1" --description "..."  # With description
pharos quiz create "<title>" -w "<ws>" --items "slug1" --lesson <seq>    # Link to a lesson

pharos quiz list -w "<workspace>"                   # List quizzes with best scores
pharos quiz list -w "<workspace>" --weak            # Sort by lowest accuracy first
pharos quiz list -w "<workspace>" --limit N         # Max results
pharos quiz read <slug> -w "<workspace>"            # Read quiz metadata and items
pharos quiz show <slug> -w "<workspace>"            # Open quiz in web dashboard
pharos quiz revise <slug> -w "<ws>" --items "slug1,slug2"  # Update items (blocks if in-progress attempts)
pharos quiz revise <slug> -w "<ws>" --lesson <seq>         # Link/unlink lesson (0 to unlink)
pharos quiz attempts <slug> -w "<workspace>"        # Show attempt history and trend
pharos quiz delete <slug> -w "<workspace>"          # Delete (blocks if in-progress attempts)
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
pharos config set auto_submit_choice on           # Auto-submit choice questions on selection
```

## Skills

```bash
pharos skills install                 # Interactively install pharos skill into AI agent
pharos skills install --agent opencode  # Install for a specific agent
pharos skills install --project       # Install at project level (not global)
pharos skills check                   # Check installed skills and their status
pharos skills uninstall               # Remove installed skills (interactive)
pharos skills uninstall --orphans     # Remove only orphaned installs at old locations
pharos skills uninstall --all         # Remove all discovered installs
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
```

## File Naming

The CLI generates filenames automatically from titles:

| Type       | Pattern                        | Example                         |
|------------|--------------------------------|---------------------------------|
| Lesson     | `0001-dash-case-name.html`      | `0001-sql-joins.html`           |
| Record     | `0001-dash-case-title.md`       | `0001-understood-inner-join.md` |
| Reference  | Slug-based (from title)         | `notation-cheat-sheet.html`     |
| Question   | Slug-based (from title)         | `what-is-a-join` (DB-only; stimulus HTML in `questions/`) |

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
├── questions/            # Stimulus HTML files for questions (optional)
└── assets/               # Reusable components (style.css, etc.)
```

## Data

SQLite database at `~/.pharos/pharos.db` (configurable via `config set data_dir`).

FTS5 full-text search (Porter tokenizer) on lessons, records, and references.
All mutations happen through the CLI — the web UI is read-only.
