# Brief: Review the Pharos README against the pitch skill

You are reviewing a pitch document — a README — for quality as a *pitch*. This is a review only: **do not edit any files**. Your final answer goes to `output.md`.

## Reference material (read these first)

- `/home/udit/Dev/personal/vibe/skills/pitch/SKILL.md` — the pitch skill. This is the standard the README is judged against. Its sections define what a pitch must do: scope (skimmer, action path), grounded claims, skim budgets (5-second / 30-second), the skim test, and AI-tell scrubbing.
- `/home/udit/Dev/personal/vibe/skills/pitch/TELLS.md` — the AI-tells checklist used by step 3 of the skill.

## What you're reviewing

The target document is `README.md` in this directory (the Pharos repo). Read it fully. The product itself is here too — Go CLI + web dashboard, see `cmd/`, `internal/`, `web/`, `docs/`, `PRODUCT.md`, `install.sh`. You may consult these to verify claims.

## What to produce

A review report in `output.md` with these sections:

1. **Verdict** — 2–3 sentences: does the README win a skimmer in 5 seconds and 30 seconds? Pass or fail the skill's skim test (title + pitch + bold verbs only).
2. **Grounded-claims audit** — walk every factual claim the README makes (install paths, commands, flags, agent names, feature bullets like "winget install", `pharos skills install --agent pi.dev`, PDF/ebook lessons) and check it against the actual product: does the command exist, does the flag exist, does the installer exist, does the feature ship? Cite the file that proves or contradicts each. List ungrounded or unverifiable claims.
3. **Budget audit** — 5-second and 30-second read: is install before features? Are features bold-verb bullets saying what the reader gets (not what the code has)? Is the privacy line earned? Is contributor/dev content behind a pointer line?
4. **AI-tells audit** — check the README against TELLS.md line by line. Quote each tell found, with the specific fix.
5. **Prioritized fixes** — a numbered list, most important first, each with the exact replacement text or edit where possible.

Rules:
- Be specific and cite evidence (file + line where possible). No vague "could be improved".
- Do not review code quality — only the pitch document.
- Do not modify any file.
