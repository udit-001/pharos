# Workbench Reference

Fixture authoring, embed patterns, and journal interpretation for the SQL workbench.

## Fixture authoring

A fixture is a self-contained practice environment: a database state, a starter query, and feedback that confirms the learner got it right.

**Authoring flow:**
1. Write the SQL that creates the fixture state
2. Write the starter query the learner should run
3. Write the expected result and the "what you should see" explanation
4. Test the fixture end-to-end before shipping

**Check workflow:**
- Run the fixture yourself before the learner sees it
- Verify the expected result matches the query output
- Confirm the explanation matches what the learner will observe

## Embed pattern

The workbench is a web component, not an iframe:

```html
<script src="/path/to/sql-workbench.js"></script>
<sql-workbench namespace="lesson-N" mode="card"></sql-workbench>
```

**Attributes:**
- `namespace`: isolates per-lesson journals (each lesson gets its own namespace)
- `mode`: `"card"` for interactive practice, `"full"` for expanded view
- `dataset`: pre-loads data into the database
- `sql`: pre-loads a starter query

**Important:** Use the web component, not an iframe. The component shares the page's context and journals runs to the host.

## Drill-block template

A drill-block is a focused practice unit with three parts:

1. **Prompt** — what the learner should try (e.g., "Write a query that finds all users who signed up last week")
2. **What you should see** — the expected result, explained in plain language
3. **Reveal** — the actual SQL that produces the result

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

**Mark runnable snippets with `data-copy`:** the server auto-injects copy buttons on `<pre>` blocks with this attribute. This lets learners copy the solution and run it directly.

## Two-tier vocabulary

When introducing SQL concepts, use two tiers:

- **Tier 1: Everyday words** — use these in explanations ("find", "filter", "group", "count")
- **Tier 2: Technical terms** — introduce the SQL keyword when the concept is understood ("SELECT", "WHERE", "GROUP BY", "COUNT")

The pattern: explain the concept in everyday language first, then name it. Never lead with the keyword — the learner needs to understand *what* before they remember *what it's called*.

## Journal interpretation

The workbench journals every query the learner runs. Use this to:
- See what the learner has already practiced
- Identify patterns (repeated mistakes, areas of confusion)
- Decide what to teach next

**Reading the journal:**
- `workbench-event` — fires on each query execution
- `bench.events()` — returns the event history
- `exportMarkdown()` — exports the journal as readable markdown

**Before scaffolding the next lesson:**
1. Read the journal to see what the learner has tried
2. Note which queries succeeded and which failed
3. Identify the gap between what they practiced and what they need
4. Scaffold the next lesson to fill that gap, not to repeat what they already know

**Namespace isolation:** each lesson's `namespace` attribute keeps journals separate. A query in lesson-1 doesn't appear in lesson-2's journal.
