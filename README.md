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

## What you can do

**Learn anything, properly**
Each topic gets its own workspace: lessons written by your AI, a glossary of
terms, cheat-sheet references, and a mission statement so the learning stays
aimed at why you started.

**Quiz yourself, find the gaps**
Multiple-choice questions grade instantly; flashcards you flip and self-grade.
Pharos tracks which questions you keep missing — the next lesson targets them.

**Capture ideas before they vanish**
"I want to learn ML someday" — saved from a chat in two seconds as a scrap,
ready to become a workspace when you're ready.

**Learn from your own material**
Hand it a PDF or ebook and it extracts the text, so lessons and quizzes can
come from the document you actually care about.

## Your data

One folder: `~/.pharos/`. Back it up and you're done. There's no account,
and nothing gets uploaded.

## Going deeper

- **[CLI reference](docs/cli-reference.md)** — every command, if you like the terminal
- **[Project setup](docs/project-setup.md)** — how Pharos is built, for contributors
