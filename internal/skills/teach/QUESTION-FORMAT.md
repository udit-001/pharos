# Question Format

Questions are the item bank a workspace's quizzes draw from. They are DB-only (no file on disk), managed via the CLI, and addressed by **slug** (derived from the title). A question's `--mode` selects its config shape and how the dashboard grades a response.

## Modes

### choice

`--body-file` is a JSON object. The dashboard grades by comparing the selected option index to `key`.

```json
{
  "options": ["CHD8", "FMR1", "MECP2", "SHANK3"],
  "key": 0
}
```

- **options** — the answer choices, in display order. At least two.
- **key** — the 0-based index of the correct answer (`0` = `CHD8` here).

### recall

`--body-file` is plain text — the reveal shown after the learner self-grades. The server does not grade recall; the learner marks *Got it* / *Not yet* in the dashboard.

## Creating

```bash
pharos question create "<title>" --mode choice --body-file <path>   # --body-file is the choice JSON above
pharos question create "<title>" --mode recall --body-file <path>   # --body-file is the reveal text
pharos question list                                                 # list all questions
pharos question list --weak                                          # sort by accuracy ascending
pharos question read <slug>                                          # inspect one question's config (correct option marked for choice)
```

## Rules

- **One concept per question.** "What is a JOIN?" and "When does the planner pick a hash join?" are two questions, not one. A question that tests two things can't tell you which the learner missed.
- **Reuse slugs across quizzes.** A question is a reusable item — `pharos question list` before creating to avoid a near-duplicate under a new slug. Several quizzes can share the same question.
- **Pick mode by the retrieval goal.** See the [Skills](./SKILL.md) section for which retrieval moment each serves, and the equal-length-options rule for `choice`.
- **Revise the quiz, not the question.** To change a quiz's contents, `pharos quiz revise <slug> --items "slug1,slug2"` with the new slug list. This blocks while the quiz has in-progress attempts — wait for them to complete or be abandoned first.
