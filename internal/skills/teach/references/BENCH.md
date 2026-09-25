# Workbench Reference

Fixtures, embeds, and journals for the SQL workbench. Loaded when the lesson
involves SQL practice or database queries.

## Embed pattern

The workbench is a web component, not an iframe:

```html
<sql-workbench namespace="lesson-N" mode="card"
               dataset="movies"
               sql="SELECT * FROM movies LIMIT 10;"></sql-workbench>
```

The server injects the component script when it sees the `<sql-workbench>`
element — the page ships no scripts. It also resolves a bare-slug `dataset`
to the workspace's installed dataset, so authors never write workspace routes.
Path or URL references (v0.5.1 host-owned form) pass through verbatim. The
component shares the page's context and journals every run to the host.

**Attributes:**

- `namespace` — isolates per-lesson journals; one namespace per lesson
  (a query in lesson-1 never appears in lesson-2's journal)
- `mode` — `"card"` for interactive practice, `"full"` for expanded view
- `dataset` — the installed dataset's bare slug (e.g. `movies`); the server
  resolves it when it serves the page. A root-relative path or absolute URL
  is fetched verbatim instead.
- `sql` — starter query pre-loaded in the editor

## Fixtures (datasets)

A fixture is a JSON file that seeds the bench — the reset state, nothing
pedagogical:

```json
{
  "id": "movies",
  "kind": "sqlite-dataset",
  "version": 1,
  "title": "Movies",
  "reset": { "sql": "CREATE TABLE movies (...); INSERT ...;" }
}
```

`reset.sql` is the full seed script (CREATE + INSERT, executed verbatim on
load and on reset). The `id` slug must equal the filename stem.

**Lifecycle:**

1. Write `<id>.json` — `<id>` is a lowercase slug (`movies`, `books-fixture`)
2. `pharos workbench check ./<id>.json` — catches invalid JSON and bad
   filenames; the bench itself validates the full shape when it loads
3. `pharos workbench add ./<id>.json` — installs into the workspace
4. Reference it: `dataset="<id>"` (the server resolves the slug)

`pharos workbench remove <id>` uninstalls. Full CLI syntax:
[pharos-cli.md](pharos-cli.md#sql-workbench).

## Fixture authoring flow

1. Write the fixture (SQL that creates the fixture state) and install it
2. Set the embed's `dataset` + `sql` attributes
3. Run the fixture yourself before the learner sees it

Completion: the expected result matches the query output, and the drill
explanation matches what the learner will actually observe — verify both
before shipping.

If the bench shows a boot error, its copy names the field and the fix —
surface it verbatim and fix the file it points at.

### The exec channel (agent runs SQL in a live bench tab, LEARN-237)

Verification runs in the learner's runtime: the agent opens the lesson
page (`pharos nav <url>` — the page must be open in the dashboard),
then drives the bench through the CLI. The server broadcasts commands
to the page over the live SSE relay; the bench executes them, and the
reply carries the engine outcome verbatim plus the verdict.

```bash
pharos workbench exec --namespace lesson-1 \
    --sql "SELECT count(*) FROM movies"
#   PASS — attempt matched the problem's test   (or FAIL + first-diff detail)
#   outcome: {"kind":"ok","rowCount":12,...}

pharos workbench verify ./revenue-top5.json --sql ./reference.sql \
    --namespace lesson-1
#   composes: setProblem → run reference → reset (always)
```

`verify` fails (exit 1) when the reference solution misses the
problem's own `test` — the hallucination guard, mechanical. It always
resets afterward, so the lesson page stays clean for learners.

No live tab → the error says so and names the fix (`pharos nav <url>`).
Commands serialize behind the human's in-flight run — the agent never
preempts the learner. Exec runs journal with `actor: agent`, visibly
tagged in the lesson's history.

### Verify protocol (state hygiene)

The author bench is the lesson page itself. The agent composes the
hygiene from three commands — the bench executes exactly what arrives:

```
reset  → run(reference.sql) → expect pass → reset  → run(wrong) → expect miss
```

Done when: the reference solution passes against the problem's `test`,
a deliberately-wrong query misses, and the journal shows both runs
(`workbench log --namespace <lesson> --type step` — agent-attributed).
The final reset restores clean state for the learner.

## Problem slots (graded practice, LEARN-236)

When a lesson needs the learner to **attempt and be verified** (not just
run demonstration queries), embed a **problem slot**: one element is one
problem. The bench grades each attempt against the element's `test` and
shows a verdict; the page or your glue decides what problem comes next
(`setProblem()`, or a re-render) — the component renders one problem,
nothing more.

For glue-driven navigation, swap the slot programmatically:

```js
bench.setProblem({
  title: "Ratings per title",
  concept: "aggregate-null",
  prompt: "Average stars per title. Include the book with no reviews.",
  sql: "",               // omit/empty = write from memory (recall problem)
  test: { rows: [["Database Internals", 4.5]], order: false },
});
```

`test` may also carry `columns: ["title", "avg"]` (exact column-name
check) and `error: "no such column"` (the problem PASSES on that
verbatim engine error).

```html
<sql-workbench namespace="lesson-3" dataset="books" mode="card"
               concept="left-join" label="Books nobody reviewed"
               test='[["The Pragmatic Programmer"],["Database Internals"]]'>
  Find every book that has no reviews.
</sql-workbench>
```

Attributes:

- `test` — JSON rows the learner's attempt is compared against. Grading
  is on the RESULT (any correct query passes), order-insensitive unless
  the lesson is about ordering. An unparsable `test` fails the boot with
  an error naming the attribute.
- `concept` — free string tag naming the skill; journals carry it so you
  can read misses by concept.
- `label` — short problem title for journals and history.
- `sql` — starter query. **Omit it for a recall problem**: the learner
  writes from memory, and the same test grades the result.
- Child text — the prompt, in tier-1 language (see below).

### Author self-verify (the hallucination guard)

You wrote the `test` — so verify it before the learner does. The same
bench grades you:

1. Embed the problem in a scratch page (or the draft lesson)
2. Run your own reference solution in the editor
3. **Pass = your test is consistent. Miss = your truth was wrong —
   regenerate `test` from the actual output and re-run.**

Completion: your reference solution passes, and a deliberately wrong
query (one column off, one filter dropped) misses. Then the learner
ships.

### Reading step events

Every attempt on a graded slot journals a `step` event next to its
`query` event — `title`, `concept`, `outcome` (pass|miss):

```
pharos workbench log --namespace lesson-3 --type step
```

Read them like journal events: scaffold into the miss
([journal interpretation](#journal-interpretation)). A concept the
learner repeatedly misses on gets **promoted** to a tracked question —
same rule as an inline check they keep failing.

## Drill-block template

A drill-block is a focused practice unit with three parts:

1. **Prompt** — what the learner should try
2. **What you should see** — the expected result in plain language
3. **Reveal** — the SQL that produces it

```html
<div class="drill-block">
  <div class="drill-prompt">
    <strong>Try this:</strong> Write a query that counts users by country.
  </div>
  <div class="drill-expected">
    <strong>What you should see:</strong> A table with two columns — country
    and count — showing how many users come from each country.
  </div>
  <div class="drill-reveal">
    <pre data-copy>SELECT country, COUNT(*) as count FROM users GROUP BY country;</pre>
  </div>
</div>
```

Styles are pre-seeded in `assets/style.css`. Mark the reveal's `<pre>` with
`data-copy` per the [copy-code convention](../PAGE-THEME.md#copy-code).

## Journal interpretation

Every query the learner runs is journaled by the bench. Read it with the CLI
(`pharos workbench log` — full reference at
[pharos-cli.md](pharos-cli.md#sql-workbench)):

```bash
pharos workbench log --namespace lesson-1 --type query --limit 50
pharos workbench log --json   # machine-readable, all namespaces merged
```

Event types: `query` (sql + ok), `error`, `info`, `dataset`. A `query` with
`ok: false` carries the learner's `error` message — fresh misses are the
gap between what they practiced and what they need.

**Before scaffolding the next lesson:** read the journal, note which queries
succeeded and which failed, and scaffold forward into the gap — never repeat
what they already practiced.

The journal feeds the CLI on its own: the server injects a relay next to the
bench, and every journaled event lands in `pharos workbench log` under the
element's `namespace` — within ~30 seconds, immediately when the tab closes,
and buffered while the tab is offline.

## Two-tier vocabulary

When introducing SQL concepts, use two tiers:

- **Tier 1: Everyday words** — use these in explanations ("find", "filter",
  "group", "count")
- **Tier 2: Technical terms** — introduce the SQL keyword once the concept
  is understood ("SELECT", "WHERE", "GROUP BY", "COUNT")

Explain the concept in everyday language first, then name it — the learner
needs to understand *what* before they remember *what it's called*.
