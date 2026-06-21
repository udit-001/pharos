# Learn Tool

A CLI tool to create and manage learning lessons and workspaces, following the [teach skill](.agents/skills/teach/SKILL.md) format.

Inspired by [Waypoint](https://github.com/SwatiBio/waypoint) — same architecture: Go CLI, SQLite, AI skill integration, and web dashboard.

## Install

### From source

```bash
git clone <repo-url>
cd learn-tool
make build     # builds the binary (includes Tailwind CSS rebuild)
```

> **Note:** `make build` rebuilds the Tailwind CSS from source using the
> standalone CLI binary at `.local/bin/tailwindcss`. If it's missing,
> download it once:
>
> ```bash
> mkdir -p .local/bin
> curl -L -o .local/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
> chmod +x .local/bin/tailwindcss
> ```
>
> (Use the `tailwindcss-macos-arm64` / `-windows.exe` binaries on other
> platforms.) No Node.js is required.

Or install directly:

```bash
go install ./cmd/learn
```

## Quick Start

```bash
learn init "sql-for-research"
cd ~/.learn-tool/workspaces/sql-for-research
learn lesson create "SELECT Basics" --workspace "sql-for-research"
learn record add "Understood SELECT, WHERE, JOIN" --workspace "sql-for-research"
learn start
```

## Usage

Learn is CLI-first. Create workspaces and lessons from the terminal.
The web UI is a read-only dashboard for what you've tracked.

### Workspaces

```bash
learn init "<name>"            # Create a new workspace
learn init "<name>" --cwd      # Create in current directory
learn workspace list            # List all workspaces
learn workspace open "<name>"   # Show workspace details
learn workspace stats           # Show learning stats
```

### Lessons

```bash
learn lesson create "<title>" -w "<workspace>"    # Create lesson scaffold
learn lesson create "<title>" -w "<workspace>" --open  # Create & open
learn lesson list -w "<workspace>"                # List lessons
```

### Learning Records

```bash
learn record add "<title>" -w "<workspace>"       # Add a record
learn record list -w "<workspace>"                # List records
```

### Mission & Resources

```bash
learn mission -w "<workspace>"                    # Show mission
learn mission -w "<workspace>" --edit             # Edit mission
learn resources -w "<workspace>"                  # Show resources
learn glossary -w "<workspace>"                   # Show glossary
```

### Web UI

```bash
learn start                        # Start dashboard
learn start --port 9090            # Custom port
```

## AI Integration

Learn ships a skill file that teaches AI coding assistants how to use
the CLI and generate learning content.

```bash
learn skills install --agent pi.dev
```

Supported agents: `pi.dev`, `claude-code`, `codex`, `opencode`.

When an AI agent creates lessons, it:
1. Reads the workspace `MISSION.md` and `learning-records/`
2. Designs a self-contained lesson HTML file
3. Links it to the shared stylesheet and other lessons
4. Creates a learning record to capture what was learned

## Workspace Structure

Each workspace follows the teach skill format:

```
<workspace>/
├── MISSION.md              # Why you're learning
├── RESOURCES.md            # Curated sources and communities
├── GLOSSARY.md             # Canonical terminology
├── NOTES.md                # Preferences and notes
├── lessons/                # Lesson HTML files
│   ├── 0001-slug.html
│   └── ...
├── learning-records/       # ADR-style records
│   ├── 0001-slug.md
│   └── ...
├── reference/              # Cheat sheets
└── assets/                 # Reusable components
    └── style.css
```

## Data

SQLite at `~/.learn-tool/learn.db`. 4 tables: `workspaces`, `lessons`,
`learning_records`, `settings`. FTS5 full-text search on lessons and records.

## License

MIT
