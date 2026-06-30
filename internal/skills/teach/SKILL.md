---
name: teach
description: Teach the user a new skill or concept, driving the pharos CLI to scaffold and track the workspace.
disable-model-invocation: true
argument-hint: "What would you like to learn about?"
---

The user has asked you to teach them something. This is a stateful request - they intend to learn the topic over multiple sessions.

Before creating anything, check whether the work already exists. A workspace is curated through **revision**, not accumulation — the same principle applies at every level. Run `pharos workspace list` to check for an existing workspace on the topic; if one exists, switch to it with `pharos workspace use "<name>"` and continue where the learner left off. Only run `pharos workspace create "<topic>"` when no workspace covers the topic yet — follow the [title rules](#titles) for a concise display name. All file operations below use the pharos CLI — see [references/pharos-cli.md](references/pharos-cli.md) for the full command reference.

## Teaching Workspace

Treat the current directory as a teaching workspace. The state of their learning is captured in this directory in several files:

- `MISSION.md`: The _reason_ the user is learning. Read with `pharos mission read` (or `pharos mission read --json`), update with `pharos mission edit --body-file <path>`. Use the format in [MISSION-FORMAT.md](./MISSION-FORMAT.md).
- `RESOURCES.md`: Curated knowledge sources and communities. Read with `pharos resources read` (or `pharos resources read --json`), update with `pharos resources edit --body-file <path>`. Use the format in [RESOURCES-FORMAT.md](./RESOURCES-FORMAT.md).
- **Glossary**: Canonical terminology for the workspace, stored in the database. Add terms with `pharos glossary create "<term>" "<definition>"`, display with `pharos glossary list` or the dashboard. See [GLOSSARY-FORMAT.md](./GLOSSARY-FORMAT.md) for term format convention.
- **Questions**: The item bank the workspace's quizzes draw from, stored in the database. Create with `pharos question create "<title>" --mode choice|recall --body-file <path>`, list with `pharos question list`. See [QUESTION-FORMAT.md](./QUESTION-FORMAT.md) for authoring craft.
- **Quizzes**: Ordered question sets the learner takes in the dashboard, stored in the database. Create with `pharos quiz create "<title>" --items "slug1,slug2"`, list with `pharos quiz list`.
- `NOTES.md`: Scratchpad for preferences and working notes. Read with `pharos notes read` (or `pharos notes read --json`), update with `pharos notes edit --body-file <path>` or `pharos notes edit --append --body-file <path>`.
- `./lessons/*.html`: Self-contained lesson HTML files. Create with `pharos lesson create "<title>" --body-file <path>`.
- `./learning-records/*.md`: ADR-style records of what was learned. Create with `pharos record create "<title>" --body-file <path>`.
- `./reference/*.html`: Reference documents — cheat sheets, syntax guides. Create with `pharos reference create "<title>" --body-file <path>`.
- `./assets/*`: Reusable **components** shared across lessons. Create with `pharos asset create <filename> --body-file <path>`.

Every workspace **mutation** goes through the CLI — never write files directly with the agent's write tool. The CLI validates the workspace, keeps the database in sync, and the dashboard up to date. A direct write bypasses all of that, so the dashboard goes stale and stats drift. **Zero exceptions.**

The `-w` flag is optional — if you've set a current workspace with `pharos workspace use`, all commands default to it. `pharos workspace create` auto-sets the new workspace as current.

The create/revise commands take content via `--body-file`, never inline — multiline HTML/MD breaks in the shell. Write the content to a temp file in the **system temp dir** (`mktemp`), pass `--body-file <path>`, then **remove the temp file once the command succeeds**:

```bash
tmp=$(mktemp)
# ... write content to "$tmp" ...
pharos lesson create "My Lesson" --body-file "$tmp" && rm "$tmp"
```

## Philosophy

To learn at a deep level, the user needs three things:

- **Knowledge**, captured from high-quality, high-trust resources
- **Skills**, acquired through highly-relevant interactive lessons devised by you, based on the knowledge
- **Wisdom**, which comes from interacting with other learners and practitioners

Before the `RESOURCES.md` is well-populated, your focus should be to find high-quality resources which will help the user acquire knowledge. Never trust your parametric knowledge.

Some topics may require more skills than knowledge. Learning more about theoretical physics might be more knowledge-based. For yoga, more skills-based.

### Fluency vs Storage Strength

You should be careful to split between two types of learning:

- **Fluency strength**: in-the-moment retrieval of knowledge
- **Storage strength**: long-term retention of knowledge

Fluency can give the user an illusory sense of mastery, but storage strength is the real goal. Try to design lessons which build long-term retention by desirable difficulty:

- Using retrieval practice (recall from memory)
- Spacing (distributing practice over time)
- Interleaving (mixing up different but related topics in practice - for skills practice only)

## Titles

Every title in a workspace — workspace display name, lesson titles, record titles, reference titles — appears in breadcrumbs, the sidebar, and search results. Long titles crowd the chrome and hide the concept.

- **Max 50 characters.** A longer title means the scope is too wide — split it.
- **Noun phrases, not sentences.** "SQL JOINs", not "How to join tables in SQL".
- **One concept per title.** "Indexed SELECTs" and "Covering indexes" are two lessons, not one called "SELECTs and indexes".
- **Drop filler words.** No "Understanding", "Introduction to", "Deep dive into", "Basics of".

When creating a workspace, pass a display-name–worthy string as the positional arg; the CLI slugifies it for the directory and uses the original as the display title. Only use `--topic` when the display title should differ from the slug source:

```bash
pharos workspace create "SQL Joins"                      # slug: sql-joins, title: SQL Joins
pharos workspace create "sql-joins" --topic "SQL Joins"  # same, but explicit topic
```

## Lessons

A lesson is the main thing you produce — the unit in which knowledge and skills reach the user. Each lesson is one self-contained HTML file, saved to `./lessons/` and titled `0001-<dash-case-name>.html` where the number increments each time.

Before creating a lesson, search for an existing one on the same topic: `pharos search "<topic>"`. Same principle — if a lesson already covers the topic, **revise** it with `pharos lesson revise <seq> --body-file <path>` instead of creating a duplicate under a new number. The sequence stays tight; the learner isn't served two lessons on the same thing.

A lesson should be **beautiful** — clean, readable typography and layout — since the user will return to these later to review. Think Tufte. When a lesson compares two concepts or shows set overlap, see [references/venn-diagram.md](references/venn-diagram.md) — text goes in callout boxes, never crammed inside circles. Link shared stylesheets with root-relative paths (`assets/style.css`, never `../assets/style.css`).

The lesson should be short, and completable very quickly. Learners' working memory is very small, and we need to stay within it. But each lesson should give the user a single tangible win that they can build on. It should be directly tied to the mission, and should be in the user's zone of proximal development.

A lesson isn't done when the file is written — it's done when the user is looking at it in the dashboard. After creating or revising a lesson, **present** it: `pharos lesson show <seq>`. This starts the dashboard if needed and opens the lesson in the browser. The dashboard renders the lesson with correct assets, navigation, and styling — the user should never open the raw HTML file directly.

The dashboard owns **navigation** between lessons — sidebar, sequencing, prev/next. Don't rebuild that chrome inside the lesson: a `← Previous` / `Next →` footer duplicates the dashboard and goes stale the moment lessons are reordered or inserted. What a lesson *does* carry is **contextual links** — mid-prose anchors to another lesson or a reference document that illuminates the point being made, placed where the reader would want it, not where it falls in the sequence. These links need special routing because a lesson renders inside an iframe — see [references/pharos-cli.md](references/pharos-cli.md) for the route table.

Each lesson should recommend a primary source for the user to read or watch. This should be the most high-quality, high-trust resource you found on the topic.

Every external link in a lesson must use `target="_blank" rel="noopener noreferrer"` so it opens in a new tab without exposing the page to `window.opener` abuse.


## Assets

HTML pages (lessons and references) are built from reusable **assets** in
`./assets/`. Two kinds:

- **User components** — stylesheets, inline quiz widgets, simulators you author with
  `pharos asset create <filename> --body-file <path>`.
- **Vendored or seeded assets** — third-party libraries: **mermaid** for
  diagrams, **highlightjs** for code highlighting (plus mermaid-lightbox for
  full-size diagram view); and the framework's universal files (style.css,
  glossary-tooltip.js, copy-code.js, the Inter font). Install with
  `pharos asset add <name>`; force-sync to the current binary with
  `pharos asset redeploy <name>`.

Reuse is the default, not the exception. Before authoring a lesson or
reference, run `pharos asset list` and reuse an existing asset for every
shared concern rather than inlining code a second page would duplicate.

A shared stylesheet ships with every workspace (`assets/style.css`, seeded) —
extend it rather than creating per-page styles. See [PAGE-THEME.md](./PAGE-THEME.md)
for the design system (Nord palette, component patterns, theming
conventions). As the workspace grows, so should the component library.

Asset paths are **root-relative** — `assets/style.css`, never
`../assets/style.css` (the iframe serves pages from
`/api/lesson-html/…`). Contextual links to other dashboard pages use
absolute routes with `target="_top"`; see [references/pharos-cli.md](references/pharos-cli.md)
for the route table.

## The Mission

Every lesson should be tied into the mission - the reason that the user is interested in learning about the topic.

If the user is unclear about the mission, or the `MISSION.md` is not populated, your first job should be to question the user on why they want to learn this.

Failing to understand the mission will mean knowledge acquisition is not grounded in real-world goals. Lessons will feel too abstract. You will have no way of judging what the user should do next.

Missions may change as the user develops more skills and knowledge. This is normal - make sure to update the `MISSION.md` and add a learning record to capture the change. Confirm with the user before changing the mission.

## Zone Of Proximal Development

Each lesson, the user should always feel as if they are being challenged 'just enough'.

The user may specify an exact thing they want to learn. If they don't, figure out their zone of proximal development by:

- Reading their `learning-records`
- Checking `pharos quiz list --weak` and `pharos question list --weak` for where retrieval is weakest
- Figuring out the right thing to teach them based on their mission
- Teach the most relevant thing that fits in their zone of proximal development

## Knowledge

Lessons should be designed around a skill the user is going to learn. The knowledge in the lesson should be only what's required to acquire that skill. You teach the knowledge first, then get the user to practice the skills via an interactive feedback loop.

Knowledge should first be gathered from trusted resources. Use `RESOURCES.md` to keep track of them. Lessons should be littered with citations - links to external resources to back up any claim made. This increases the trustworthiness of the lesson.

For acquiring knowledge, difficulty is the enemy. It eats working memory you need for understanding.

## Skills

If knowledge is all about acquisition, skills are about durability and flexibility. Make the knowledge stick.

For skill acquisition, difficulty is the tool. Effortful retrieval is what builds storage strength. Skills are taught through interactive lessons, then practiced and tracked through the quiz subsystem. Each retrieval moment has its own instrument — recognise the moment, then reach for the one that fits:

- **In-lesson check** — when the learner has just read an explanation and you want to confirm they understood, not just nodded along, embed a short inline quiz in the lesson HTML. Formative and in-context; the feedback is immediate. It is ephemeral: do not track or supersede it.
- **Practice quiz** — when a stretch is done and the goal is to pull it *out of memory*, build a `pharos quiz` from `pharos question` items. Scored, with a review page, taken in the dashboard. Pick mode by the retrieval you want: `choice` checks recognition (fluency-leaning), `recall` demands free recall (storage-strength-leaning). Size it to the coherent chunk the learner just finished, and link it to that lesson with `--lesson <seq>` — the lesson↔quiz join is explicit in the DB, so the skill area a quiz practices is named, not inferred from topic naming. After creating, present it with `pharos quiz show <slug>` — the learner starts when ready.
- **Weakness signal** — when the learner returns with limited time, or you're choosing what to teach next, run `pharos quiz list --weak` (weakest skill areas) then `pharos question list --weak` (specific items dragging them down). Never-attempted sort first, then by accuracy ascending; the `Last` column tells you whether a weakness is stale (possibly since fixed) or fresh — weight fresh misses heavier. The workspace's storage-strength signal, feeding both the next quiz and the next lesson.
- **Progress** — tracked attempts and the review page let the learner watch a score climb across retakes. To see the trajectory (is accuracy improving?), run `pharos quiz attempts <slug>` — it prints the completed-attempt history with a trend summary; the motivation loop, distinct from the retrieval itself.
- **Real-world steps** — for procedural skills (yoga poses, lifts), the instrument is a lesson that walks the learner through the steps and asks them to perform them. The performance itself is the retrieval; feedback comes from doing it.

Each is a **feedback loop** — tight as possible, immediate, and automatic where it can be.

### Promoting an inline check

An inline check is ephemeral, but a struggle it reveals is worth keeping. When a learner repeatedly fails an inline check on a concept, **promote** it: `pharos question create` the concept, then `pharos quiz revise --items` to add it to the relevant quiz. The formative check graduates into the tracked record; `--weak` then sees it.

For multiple-choice — inline or `choice` mode — keep all options the same length and character count, so the correct answer isn't leaked by formatting. See [QUESTION-FORMAT.md](./QUESTION-FORMAT.md) for config shapes and authoring rules.

## Acquiring Wisdom

Wisdom comes from true real-world interaction - testing your skills outside the learning environment.

When the user asks a question that appears to require wisdom, your default posture should be to attempt to answer - but to ultimately delegate to a **community**.

A community is a place (online or offline) where the user can test their skills in the real world. This might be a forum, a subreddit, a real-world class (budget permitting) or a local interest group.

You should attempt to find high-reputation communities the user can join. If the user expresses a preference that they don't want to join a community, respect it.

## Learning Records

Records follow the ADR convention: you don't edit them, you **supersede** them. When understanding changes:

```bash
pharos record supersede <seq> --title "Revised understanding" --body-file <path>
```

This atomically creates a new record and marks the old one as superseded. The old record is still visible (status: superseded) — it shows how understanding evolved.

## Reference Documents

While creating lessons, you should also create reference documents. Lessons can reference these documents - they are useful for tracking raw units of knowledge useful across lessons.

Lessons will rarely be revisited later - reference documents will be. They should be the compressed essence of the lesson, in a format designed for quick reference.

References are addressed by **slug** (descriptive name derived from the title), not sequence numbers. If a reference needs updating, revise it: `pharos reference revise <slug> --body-file <path>`.

References are HTML files that render in the same iframe as lessons — they must link `assets/style.css` and follow the [PAGE-THEME.md](./PAGE-THEME.md) boilerplate (theme sync, root-relative asset paths, FOUC prevention). A reference that omits the stylesheet renders unstyled.

Some learning topics lend themselves to reference:

- Syntax and code snippets for programming
- Algorithms and flowcharts for processes
- Yoga poses and sequences for yoga
- Exercises and routines for fitness
- Glossaries for any topic with its own nomenclature

Glossaries, in particular, are an essential reference. Once one is created, it should be adhered to in every lesson. When writing a lesson, fetch the glossary terms with `pharos glossary list --json` and wrap matching terms with `<span class="glossary-term" data-term="...">` for inline tooltip previews — see [PAGE-THEME.md](./PAGE-THEME.md) for the tooltip convention.

## `NOTES.md`

The user will sometimes express preferences of how they want to be taught, or things you should keep in mind. Record these with `pharos notes edit --body-file <path>` or append to them with `pharos notes edit --append --body-file <path>`. To review existing notes, use `pharos notes read` (or `pharos notes read --json`).

After each session, check `NOTES.md` for user preferences before starting the next session. The dashboard's "Continue where you left off" feature tracks which workspace and lesson the user last viewed — it picks up automatically.

## Followup Questions

After presenting a lesson, ask the user in chat if they have any followup questions.
