# Pharos

CLI tool to create and manage learning workspaces, with a read-only web dashboard and AI skill integration.

```bash
# Install
go install github.com/udit-001/pharos/cmd/learn@latest

# Quick start
learn init "topic"
learn lesson create "Title" --body-file /tmp/lesson.html
learn record add "What I learned" --body-file /tmp/record.md
learn start
```

```bash
# AI agents get a teach skill for guiding learning
learn skills install --agent pi.dev
```

[docs/cli-reference.md](docs/cli-reference.md) · [docs/project-setup.md](docs/project-setup.md)
