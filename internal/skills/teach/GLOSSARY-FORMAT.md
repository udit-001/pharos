# Glossary Format

The glossary is the canonical language for this teaching workspace. All explainers, exercises, and learning records should adhere to its terminology. Building it is itself part of learning: compressing a concept into a tight definition is evidence the user understands it.

Terms are stored in the database and managed via the CLI. Each term has four fields:

- **term** — the canonical name for the concept
- **definition** — one or two sentences describing what it is
- **category** (optional) — a heading to group under, e.g. "Diagnostic & Clinical"
- **avoid** (optional) — synonyms or phrasing to avoid, e.g. "Bulking, getting big"

```bash
pharos glossary create "<term>" "<definition>"                            # create a term
pharos glossary create "<term>" "<definition>" --category "<group>"       # with a category
pharos glossary create "<term>" "<definition>" --category "<group>" --avoid "<aliases>"  # with avoid list
pharos glossary list                                                      # list all terms
pharos glossary list --json                                               # machine-readable output
```

## Rules

- **Add a term only when the user understands it.** The glossary is a record of compressed knowledge, not a dictionary the user reads to learn. If the user has just been introduced to a concept, wait until they can use it correctly before promoting it here.
- **Be opinionated.** When several words exist for the same concept, pick the best one and list the rest as the `--avoid` field. This is how language compresses.
- **Keep definitions tight.** One or two sentences. Define what the term IS, not what it does or how to do it.
- **Use the glossary's own terms inside definitions.** Once a term is in the glossary, prefer it everywhere — including inside other definitions. This is what makes complex terms easier to grasp later.
- **Group under categories** when natural clusters emerge. Pass `--category "Anatomy"` to assign a term to a subheading. A flat list is fine when terms cohere.
- **Flag ambiguities explicitly.** If a term is used loosely in the wider field, note the resolution in the definition: "In this workspace, 'set' always means a working set — warm-ups are tracked separately."
- **Revise as understanding deepens.** A definition the user wrote in week one may be wrong by week six. Update in place with `pharos glossary revise "<term>" --definition "..."`; do not leave stale entries.
