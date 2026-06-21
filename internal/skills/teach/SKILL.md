---
name: teach
description: Teach the user a skill or concept over multiple sessions, driving the pharos CLI to scaffold and track the workspace.
disable-model-invocation: true
argument-hint: "What would you like to learn about?"
---

Teach a topic over multiple sessions. The `learn` CLI owns the **workspace** — you drive it, you don't hand-write files. Every session follows the **pipeline**: mission → resources → lesson → record → reference.

For command syntax and flags, see [references/pharos-cli.md](references/pharos-cli.md). This skill decides _when_ and _why_ to run each, and how to teach well.

### Passing content

The create commands take content via `--body-file`, never inline — multiline HTML/MD breaks in the shell. Write the content to a temp file in the **system temp dir** (`mktemp`), pass `--body-file <path>`, then **remove the temp file once the command succeeds**. The content now lives in the workspace; the scratch copy should not linger anywhere.

```bash
tmp=$(mktemp)        # in $TMPDIR (or /tmp)
# ... write content to "$tmp" ...
pharos lesson create "<title>" -w "<name>" --body-file "$tmp" && rm "$tmp"
```

Never write scratch content into the workspace dir or cwd — those create duplicate copies that drift from the recorded one.

## The workspace

A workspace is one topic, created by `pharos init`. The directory layout and file naming conventions are documented in [references/pharos-cli.md](references/pharos-cli.md) — that is the single source of truth for how a workspace is structured.

The format docs ([MISSION-FORMAT.md](MISSION-FORMAT.md), [RESOURCES-FORMAT.md](RESOURCES-FORMAT.md), [LEARNING-RECORD-FORMAT.md](LEARNING-RECORD-FORMAT.md), [GLOSSARY-FORMAT.md](GLOSSARY-FORMAT.md)) describe the expected content layout for each file type — consult them when structuring your content.

## Pipeline

### 1. Ground in the mission

If `MISSION.md` is empty or the user's "why" is unclear, interview them before teaching anything. A vague mission steers every future lesson wrong — push for a concrete real-world outcome, not "understand X".

`pharos init "<topic>"` (or `pharos workspace open "<name>"` if it exists) scaffolds the workspace; then fill `MISSION.md` per [MISSION-FORMAT.md](MISSION-FORMAT.md).

**Done when**: `MISSION.md` states a concrete, observable outcome the user is chasing.

### 2. Gather resources

Before teaching, find high-trust sources. **Never trust your parametric knowledge** — cite. Populate `RESOURCES.md` per [RESOURCES-FORMAT.md](RESOURCES-FORMAT.md). Prefer primary sources, recognised experts, peer-reviewed work, well-moderated communities. Surface gaps explicitly where no good source exists.

**Done when**: every claim you intend to teach traces to a cited source, or a gap is logged.

### 3. Teach a lesson

A **lesson** is one self-contained HTML file teaching one tightly-scoped thing, tied to the mission, in the user's zone of proximal development.

Read `learning-records/` first to find the zone — don't re-teach what is recorded as known. Create with `pharos lesson create "<title>" --body-file <path>` (see _Passing content_ above). A lesson should be:

- **Short** — one tangible win, within working memory
- **Beautiful** — clean typography (think Tufte); reuse `assets/style.css` as the shared component
- **Cited** — link a primary source
- **Connected** — HTML anchors to other lessons and references
- **Practised** — build storage strength, not just fluency (see Philosophy)

Open the lesson for the user when done.

**Done when**: the user has completed the lesson's single win.

### 4. Record

After a lesson, capture non-obvious insights with `pharos record add "<insight>" --body-file <path>` (see _Passing content_ above). Write a record when the user: showed genuine understanding, disclosed prior knowledge, had a misconception corrected, or the mission shifted. See [LEARNING-RECORD-FORMAT.md](LEARNING-RECORD-FORMAT.md).

**Done when**: the record states _what_ is now known and _why_ it changes what to teach next.

### 5. Reference

Compress a lesson's essence into `reference/*.html` — cheat sheets, algorithms, glossaries. Create with `pharos reference create "<title>" --body-file <path>` (see _Passing content_ above). Maintain `GLOSSARY.md` as terms settle. References outlive lessons; they are what the user returns to.

**Done when**: the reference is the compressed essence, designed for quick lookup.

## Philosophy

### Knowledge, skills, wisdom

- **Knowledge** — gathered from trusted resources, cited.
- **Skills** — built through interactive lessons with tight feedback loops.
- **Wisdom** — comes from real-world practice; delegate to a community (forum, class, group) when the user is ready.

### Fluency vs storage strength

Fluency (in-the-moment retrieval) gives an illusory sense of mastery; **storage strength** (long-term retention) is the real goal. Build it with:

- **Retrieval practice** — recall from memory
- **Spacing** — distribute practice across sessions
- **Interleaving** — mix related topics (skills practice only)

For knowledge, difficulty is the enemy — it eats the working memory needed for understanding. For skills, difficulty is the tool — effortful retrieval is what builds storage.

### Zone of proximal development

Each lesson challenges just enough. Read `learning-records/` before designing one: they show what is known, what has been introduced, where misconceptions were corrected. Teach the most relevant thing that fits the gap.

### The mission is the compass

Every lesson traces to `MISSION.md`. If the mission shifts, update it (confirm with the user first) and record the change. A mission left behind makes lessons feel abstract and leaves no way to judge what comes next.

## After each session

- `pharos workspace open "<name>"` — touches `last_studied`
- Suggest the next lesson from the zone of proximal development
- Check `NOTES.md` for user preferences
