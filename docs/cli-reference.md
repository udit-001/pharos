# CLI Reference

## Workspaces

```bash
learn init "<name>"                               # Create a new workspace
learn init "<name>" --cwd                         # Create in current directory
learn init "<name>" --dir <path>                  # Create at custom path
learn init "<name>" --topic "<friendly title>"    # Override the display title
learn init "<name>" --force                       # Recreate DB if exists

learn workspace list                              # List all workspaces
learn workspace open "<name>"                      # Show workspace details
learn workspace open "<name>" --open               # Open in file manager
learn workspace stats                              # Show learning statistics
```

## Lessons

```bash
learn lesson create "<title>" -w "<workspace>" --body-file <path>    # Create lesson with content
learn lesson create "<title>" -w "<workspace>" --body-file <path> --open   # Create & open
learn lesson list -w "<workspace>"                                      # List lessons
learn lesson list -w "<workspace>" --search "<q>"                       # Search lessons
```

## Learning Records

```bash
learn record add "<title>" -w "<workspace>" --body-file <path>         # Add a record with content
learn record add "<title>" -w "<workspace>" --body-file <path> --summary "..."   # With summary
learn record list -w "<workspace>"                                       # List records
learn record list -w "<workspace>" --search "<q>"                        # Search records
```

## References

```bash
learn reference create "<title>" -w "<workspace>" --body-file <path>  # Create a reference document
learn reference list -w "<workspace>"                                   # List references
learn reference list -w "<workspace>" --search "<q>"                    # Search references
```

## Mission & Resources

```bash
learn mission -w "<workspace>"                    # Show mission
learn mission -w "<workspace>" --edit             # Edit mission in $EDITOR
learn resources -w "<workspace>"                  # Show resources
learn resources -w "<workspace>" --edit           # Edit resources
learn glossary -w "<workspace>"                   # Show glossary
learn glossary -w "<workspace>" --edit            # Edit glossary
```

## Web UI

```bash
learn start                        # Start read-only web dashboard (default :8080)
learn start --port 9090            # Custom port
learn start --no-open              # Don't auto-open browser
```

## Global Flags

```bash
--json      # Machine-readable JSON output (most commands)
--db        # Custom database path (default: ~/.learn-tool/learn.db)
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

SQLite database at `~/.learn-tool/learn.db` (configurable via `--db`).

4 tables: `workspaces`, `lessons`, `learning_records`, `references_t`.
FTS5 full-text search on lessons, records, and references.
All mutations happen through the CLI — the web UI is read-only.
