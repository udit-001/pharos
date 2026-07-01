# Pharos

CLI tool to create and manage learning workspaces, with a read-only web dashboard and AI skill integration.

```bash
# Install
go install github.com/udit-001/pharos/cmd/pharos@latest

# Quick start
pharos init                        # Create the database, offer the agent skill
pharos workspace create "Topic Name"  # Creates ~/.pharos/workspaces/topic-name/
pharos lesson create "Title" --body-file /tmp/lesson.html
pharos record create "What I learned" --body-file /tmp/record.md
pharos start
```

```bash
# AI agents get a teach skill for guiding learning
pharos skills install --agent pi.dev
```

[docs/cli-reference.md](docs/cli-reference.md) · [docs/project-setup.md](docs/project-setup.md)
