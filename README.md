# Pharos

CLI tool for AI-guided learning workspaces, with a read-only web dashboard for taking quizzes and browsing lessons.

## Quick start

```bash
go install github.com/udit-001/pharos/cmd/pharos@latest
pharos init                            # create database, install teach skill
pharos workspace create "Topic Name"   # creates ~/.pharos/workspaces/topic-name/
pharos start                           # open the web dashboard (:9090)
```

Install the teach skill so your AI agent can drive the workflow:

```bash
pharos skills install               # auto-detects your agent
```

Then ask your agent: *"teach me about [topic]"* — it creates lessons, quizzes, and tracks your progress.

## How learning works

Pharos builds a **workspace** per topic. Each workspace has:

- **Mission** — why you're learning this
- **Lessons** — self-contained HTML, authored by your AI agent
- **Quizzes** — retrieval practice (choice and recall questions)
- **Learning records** — ADR-style notes of what you understood
- **References** — cheat sheets
- **Glossary** — canonical terminology with in-lesson tooltips

The loop: lessons teach, quizzes test, weak questions drive the next lesson.

## Web dashboard

`pharos start` opens a read-only dashboard where you:

- Browse lessons, records, and references
- **Take quizzes** with instant feedback and review
- Search across all content
- Toggle light/dark/system theme

## Quizzes

Two question modes:

- **Choice** — select an option, get graded instantly
- **Recall** — flip a flashcard, self-grade (Got it / Not yet)

Enable auto-submit to skip the Check button on choice questions:

```bash
pharos config set auto_submit_choice on
```

Track weak spots:

```bash
pharos quiz list --weak          # quizzes by lowest accuracy
pharos question list --weak      # specific questions dragging you down
```

## Configuration

```bash
pharos config read                          # show current settings
pharos config set data_dir ~/my-pharos      # move the data directory
pharos config set auto_submit_choice on      # auto-submit on option select
```

## Data & privacy

All data is local — SQLite at `~/.pharos/pharos.db`, workspace files under `~/.pharos/workspaces/`. No telemetry, no accounts, no cloud.

## Maintenance

```bash
pharos upgrade          # upgrade to latest release
pharos migrate status   # check database migrations
```

## Docs

[CLI reference](docs/cli-reference.md) · [Project setup](docs/project-setup.md)
