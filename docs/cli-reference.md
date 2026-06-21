# CLI Reference

## Workspaces

```bash
pharos init "<name>"                               # Create a new workspace
pharos init "<name>" --cwd                         # Create in current directory
pharos init "<name>" --dir <path>                  # Create at custom path
pharos init "<name>" --topic "<friendly title>"    # Override the display title
pharos init "<name>" --force                       # Recreate DB if exists

pharos workspace list                              # List all workspaces
pharos workspace open "<name>"                      # Show workspace details
pharos workspace open "<name>" --open               # Open in file manager
pharos workspace stats                              # Show learning statistics
```

## Lessons

```bash
pharos lesson create "<title>" -w "<workspace>" --body-file <path>    # Create lesson with content
pharos lesson create "<title>" -w "<workspace>" --body-file <path> --open   # Create & open
pharos lesson list -w "<workspace>"                                      # List lessons
pharos lesson list -w "<workspace>" --search "<q>"                       # Search lessons
```

## Learning Records

```bash
pharos record add "<title>" -w "<workspace>" --body-file <path>         # Add a record with content
pharos record add "<title>" -w "<workspace>" --body-file <path> --summary "..."   # With summary
pharos record list -w "<workspace>"                                       # List records
pharos record list -w "<workspace>" --search "<q>"                        # Search records
```

## References

```bash
pharos reference create "<title>" -w "<workspace>" --body-file <path>  # Create a reference document
pharos reference list -w "<workspace>"                                   # List references
pharos reference list -w "<workspace>" --search "<q>"                    # Search references
```

## Mission & Resources

```bash
pharos mission -w "<workspace>"                    # Show mission
pharos mission -w "<workspace>" --edit             # Edit mission in $EDITOR
pharos resources -w "<workspace>"                  # Show resources
pharos resources -w "<workspace>" --edit           # Edit resources
pharos glossary -w "<workspace>"                   # Show glossary
pharos glossary -w "<workspace>" --edit            # Edit glossary
```

## Web UI

```bash
pharos start                        # Start read-only web dashboard (default :8080)
pharos start --port 9090            # Custom port
pharos start --no-open              # Don't auto-open browser
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
| Reference  | `0001-dash-case-name.html`      | `0001-notation-cheat-sheet.html`|

## Workspace Layout

```
<name>/
├── MISSION.md            # Why you're learning
├── RESOURCES.md          # Curated sources and communities
├── GLOSSARY.md           # Canonical terminology (built over time)
├── NOTES.md              # Preferences and working notes
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
