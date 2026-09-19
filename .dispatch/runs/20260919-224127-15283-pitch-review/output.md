I have no file-writing tools (read-only harness), so the report below is my final answer for `output.md`.

---

# Pitch review: Pharos README vs. the pitch skill

## 1. Verdict

**Fail — on the action path, not the writing.** The 5-second budget is won: title + one-line pitch state the product, the reader, and the differentiator ("all on your computer"). The 30-second budget fails where it hurts most: the quick-start's second command (`pharos skills install --agent pi.dev`) errors out — `--agent` is not a real flag — so the skimmer who pastes the path in hits a wall on step 2. The skill is explicit that "a pitch that promises an install the product can't deliver is a worse bug than a blank install section" (SKILL.md §0). Fix one flag and this passes both budgets.

## 2. Grounded-claims audit

| Claim (README line) | Verdict | Evidence |
|---|---|---|
| `curl …/install.sh \| sh` installs on macOS/Linux (L17) | ✅ Grounded | `install.sh` exists; detects OS/arch, pulls GoReleaser archive, installs to `/usr/local/bin` or `~/.local/bin` |
| `winget install udit-001.Pharos` (L23) | ✅ Grounded | `.github/workflows/winget.yml:38` publishes `identifier: udit-001.Pharos` with `installers-regex: '\.exe$'` |
| `go install github.com/udit-001/pharos/cmd/pharos@latest` (L26) | ✅ Grounded | `go.mod:1` (`module github.com/udit-001/pharos`) + `cmd/pharos/main.go` exists |
| **`pharos skills install --agent pi.dev` (L32)** | ❌ **Ungrounded — the flag does not exist** | `internal/cli/skills.go:57–60` registers only `--agents-only`, `--claude-only`, `--all`, `--project`. Cobra returns `unknown flag: --agent`. The flag is a phantom copied from `docs/cli-reference.md:200`, which documents the same nonexistent flag (companion-doc drift — report, per SKILL.md §1) |
| Works with `pi.dev`, `claude-code`, `codex`, `opencode` (L35) | ✅ Grounded | `internal/cli/skills.go:37–42` — the `providers` list is exactly these four |
| `pharos init` (L31) | ✅ Grounded | `internal/cli/init.go` — creates config + SQLite DB, idempotent |
| `pharos start` "opens the dashboard" (L42) | ✅ Grounded | `internal/cli/start.go:51` ("Start the web UI dashboard"), auto-open browser flag at `start.go:169` |
| `pharos upgrade` (L45) | ⚠️ Grounded with a caveat | `internal/cli/upgrade.go` exists, **but it upgrades via `go install`** (upgrade.go Long help: "This compiles from source — no binary download"). A curl-installed user without Go gets "Go is not installed on your PATH." The README never says upgrade needs Go — a reader who installed via the script will assume it works the same way |
| `winget upgrade udit-001.Pharos` (L46) | ✅ Grounded | Portable .exe in winget; `winget upgrade` applies |
| Lessons, glossary, cheat-sheet references, mission (L50) | ✅ Grounded | `internal/cli/lesson.go`, `glossary.go`, `reference.go`, `mission.go` all exist |
| Instant multiple-choice, flip-and-grade flashcards (L51) | ✅ Grounded | `internal/render/quiz-attempt.js:113+` (flip card with self-grade), `docs/quiz-design.md:118` ("Recall (self-grade flashcard)") |
| Scraps — "I want to learn ML" becomes a scrap (L52) | ✅ Grounded | `internal/cli/scrap.go` (Long help literally cites "I want to be an ML engineer") |
| PDF or ebook → lessons and quizzes (L53) | ✅ Grounded | `internal/extract/pdf.go`, `internal/extract/epub.go`, `internal/cli/document.go:19` (`pharos document extract ~/book.pdf`) |
| "running record of what you've actually learned" (L4) | ✅ Grounded | `internal/cli/record.go`, `record_supersede.go` |
| "One folder: `~/.pharos/`" (L57) | ⚠️ Mostly grounded, imprecise | `internal/config/config.go:36` — `DefaultDataDir()` is `~/.pharos` for the *data*. But the config file lives in the XDG config dir (`~/.config/pharos/pharos.toml`, config.go:32), and `skills install` writes to `~/.agents/skills/` and `~/.claude/skills/` (skills.go:44–58). "One folder" overclaims by one config file and two skill dirs |
| "watch your progress" (L45) | ✅ Grounded | `internal/cli/workspace_stats.go` |

**Ungrounded/unverifiable list:** the `--agent` flag (fatal — breaks the documented action path); the upgrade-requires-Go omission; "One folder" precision; and "becomes a scrap in two seconds" (L52) — an unverifiable performance claim, though harmless as idiom.

## 3. Budget audit

**5-second read (title + pitch):** Pass. "Ask your AI to teach you something. Pharos turns that into lessons, quizzes, and a running record of what you've actually learned — all on your computer." What it does, who it's for (people with an AI agent), differentiator (local). Mechanism (Go, SQLite, templ) stays out of the lead, per SKILL.md §2.

**30-second read:**
- **Install before features** ✅ — `## Get started` (L10) precedes `## What it does` (L48).
- **Action path** ❌ — fails at step 2 (the `--agent` flag). Steps 1 and 3 are clean.
- **Bold-verb bullets saying what the reader gets** ✅ — "Learn anything / Quiz the gaps / Save 'someday' ideas / Learn from your files." All four are reader-outcome framed; only "a glossary, cheat-sheet references" names artifacts, but they're what the reader receives, so it holds. Skim test on title + pitch + bold verbs sells the product.
- **Privacy line earned?** ✅ — "local" is the pitch's own differentiator (L4), so `## Your data` is required, and it's real (no server code phones home; data dir is local). The *content* slightly overclaims ("One folder") — see audit above.
- **Contributor content behind a pointer line** ✅ — `## Going deeper` is two links; build/architecture detail lives in `docs/project-setup.md`, which exists.

## 4. AI-tells audit (per TELLS.md)

1. **Rule-of-three** — two hits, adjacent:
   - L3–4: "lessons, quizzes, and a running record" — **justified**: it's the literal product scope (lesson/quiz/record are the three artifact types; `lesson.go`, `quiz.go`, `record.go`). May stay per TELLS #1's carve-out.
   - L6–8: "lessons written for you, quizzes that find your weak spots, and the next lesson aimed at exactly those gaps" — **same triple again**, one paragraph later. A triple may survive when it's product scope; a *second* restatement of it is machine rhythm. **Fix:** cut to the one thing this paragraph adds — the weak-spot loop:
     > You read something once and forget it. Pharos is the other loop: the next lesson is aimed at exactly the gaps your last quiz exposed.
2. **Reassurance tails** — none. "Back it up and you're done" (L57) is an instruction, not a tail.
3. **Participle tails** — none.
4. **Smoothed generics** — none. No "and more," "various," "at a glance." The bullets enumerate real features. Good.
5. **Promotional register** — none. No "seamless/powerful/effortless." Clean.
6. **Rhetorical questions** — none.
7. **Em-dash spray** — within budget: one in the pitch, one per bullet (structural, prescribed by the skill's own `- **Bold verb** — one line.` format), one in "Going deeper." No section exceeds two.
8. **Significance puffery** — none.
9. **Stock phrase pairs** — none.
10. **"Whether you're X or Y"** — none.

Minor word-echo (not a TELLS item): "actually" appears twice — L4 ("actually learned") and L53 ("the document you actually care about"). Drop the second: "lessons and quizzes come from that document."

## 5. Prioritized fixes

1. **Fix the broken quick-start command (L32).** Replace with a real invocation — bare `skills install` auto-detects installed agents and prompts (`internal/cli/runSkillsInstall`):
   ```bash
   pharos init
   pharos skills install
   ```
   Keep L35's agent list as-is; it's accurate. If per-agent selection is wanted, the honest form is `pharos skills install --agents-only` / `--claude-only` / `--all` — but bare install is the right default for a skimmer.
2. **Fix the upgrade caveat (L45–46).** `pharos upgrade` compiles from source and needs Go; a curl-installed user may not have it. Replace:
   > Take quizzes, review lessons, watch your progress. To update: re-run the install script, or `pharos upgrade` if you have Go (it compiles from source). On Windows: `winget upgrade udit-001.Pharos`.
3. **Cut the doubled triple (L6–8).** Replace the second paragraph with:
   > You read something once and forget it. Pharos is the other loop: the next lesson is aimed at exactly the gaps your last quiz exposed.
4. **Make the privacy line precise (L57–58).** Replace:
   > Your learning data lives in one folder, `~/.pharos/` — back it up and you're done. No account, nothing uploaded.
   (Drops the "One folder" overclaim about the config file and skill dirs without turning the line into a directory tour.)
5. **Trim the echoes.** L52: "becomes a scrap in two seconds" → "becomes a scrap — one line, ready to become a workspace when you are." L53: "the document you actually care about" → "that document."
6. **Companion-doc drift to report, not repair here:** `docs/cli-reference.md:200` documents the same phantom `--agent` flag. It will mislead anyone who follows the README's pointer; commissioning that fix is separate accuracy work (SKILL.md §1).
