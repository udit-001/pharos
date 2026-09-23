# Workbench Reference

Fixtures, embeds, and journals for the SQL workbench. Loaded when the lesson
involves SQL practice or database queries.

## Embed pattern

The workbench is a web component, not an iframe:

```html
<sql-workbench namespace="lesson-N" mode="card"
               dataset="/api/workspaces/name/{ws}/datasets/<id>"
               sql="SELECT * FROM movies LIMIT 10;"></sql-workbench>
```

The server injects the component script when it sees the `<sql-workbench>`
element — the page ships no scripts. The component shares the page's context
and journals every run to the host.

**Attributes:**

- `namespace` — isolates per-lesson journals; one namespace per lesson
  (a query in lesson-1 never appears in lesson-2's journal)
- `mode` — `"card"` for interactive practice, `"full"` for expanded view
- `dataset` — a **reference**: a bare slug loads `fixtures/<id>.json` from
  the app root; a root-relative path or absolute URL is fetched verbatim.
  In pharos lessons use the workspace datasets API path shown above.
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
4. Reference it: `dataset="/api/workspaces/name/{ws}/datasets/<id>"`

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

The server auto-injects a relay next to the bench when it embeds the component:
every journaled event is posted to the workspace's ingest endpoint within
seconds (batched; buffered if the tab closes first). Events land in
`pharos workbench log` with the element's `namespace` — read it before the
next lesson rather than asking the learner what they ran.

## Two-tier vocabulary

When introducing SQL concepts, use two tiers:

- **Tier 1: Everyday words** — use these in explanations ("find", "filter",
  "group", "count")
- **Tier 2: Technical terms** — introduce the SQL keyword once the concept
  is understood ("SELECT", "WHERE", "GROUP BY", "COUNT")

Explain the concept in everyday language first, then name it — the learner
needs to understand *what* before they remember *what it's called*.
