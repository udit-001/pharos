# Pharos

A CLI tool to create and manage learning workspaces, with a read-only web dashboard and AI skill integration.

Inspired by [Waypoint](https://github.com/SwatiBio/waypoint) — same architecture: Go CLI, SQLite, AI skill integration, and web dashboard.

## Install

### From source

```bash
git clone <repo-url> && cd pharos
make build           # builds binary (one-step: CSS + Go)
```

See [docs/project-setup.md](docs/project-setup.md) for Tailwind CLI setup and development instructions.

### Or install directly

```bash
go install ./cmd/learn
```

## Quick Start

```bash
learn init "sql-for-research"
cd ~/.learn-tool/workspaces/sql-for-research
learn lesson create "SELECT Basics" --body-file /tmp/lesson.html   # Content via file
learn record add "Understood SELECT, WHERE, JOIN" --body-file /tmp/record.md
learn start
```

> Content for lessons, records, and references is always passed via `--body-file`
> (never inline) — multiline HTML/MD breaks in the shell. Write to a temp file first,
> then create.

## AI Integration

Pharos ships a teach skill that teaches AI coding assistants how to guide your learning using the CLI. Install it for your agent:

```bash
learn skills install --agent pi.dev
```

Supported agents: `pi.dev`, `claude-code`, `codex`, `opencode`.

Once installed, your agent can:
- Create and manage learning workspaces
- Generate beautiful, self-contained lesson HTML
- Track progress with ADR-style learning records
- Build reference cheat sheets as you learn

### How it works

The teach skill drives the `learn` CLI — the agent doesn't hand-write workspace files. Every session follows a pipeline: **define the mission → gather resources → teach a lesson → record insights → build references**. Lessons, records, and references are created via CLI commands with content passed through `--body-file` from temp files, never inline (multiline HTML/MD is fragile in shell invocations).

## Documentation

- [CLI Reference](docs/cli-reference.md) — full command list, flags, workspace layout
- [Project Setup](docs/project-setup.md) — building from source, architecture overview

## License

MIT
