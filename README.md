# Pharos

Ask your AI to teach you something. Pharos turns that into lessons, quizzes,
and a running record of what you've actually learned — all on your computer.

You read something once and forget it. Pharos is the other loop: lessons
written for you, quizzes that find your weak spots, and the next lesson
aimed at exactly those gaps.

## Get started

**1. Install Pharos**

```bash
curl -sfL https://raw.githubusercontent.com/udit-001/pharos/main/install.sh | sh
```

(If you have Go installed, `go install github.com/udit-001/pharos/cmd/pharos@latest` works too.)

**2. Connect your AI**

```bash
pharos init
pharos skills install --agent pi.dev
```

Works with `pi.dev`, `claude-code`, `codex`, `opencode`. Then just ask:
*"teach me linear algebra"* or *"teach me French cooking"*. Your AI builds
the lessons; Pharos keeps track of everything.

**3. Learn**

```bash
pharos start    # opens the dashboard
```

Take quizzes, review lessons, watch your progress. Update later with
`pharos upgrade`.

## What it does

- **Learn anything** — each topic gets a workspace: AI-written lessons, a glossary, cheat-sheet references, and a mission for why you started.
- **Quiz the gaps** — instant multiple-choice, flip-and-grade flashcards. The questions you keep missing drive the next lesson.
- **Save "someday" ideas** — "I want to learn ML" becomes a scrap in two seconds, ready to be a workspace when you are.
- **Learn from your files** — hand over a PDF or ebook; lessons and quizzes come from the document you actually care about.

## Your data

One folder: `~/.pharos/`. Back it up and you're done. There's no account,
and nothing gets uploaded.

## Going deeper

- **[CLI reference](docs/cli-reference.md)** — every command, if you like the terminal
- **[Project setup](docs/project-setup.md)** — how Pharos is built, for contributors
