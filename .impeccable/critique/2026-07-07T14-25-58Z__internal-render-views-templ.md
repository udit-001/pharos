---
target: dashboard
total_score: 31
p0_count: 1
p1_count: 2
p2_count: 2
timestamp: 2026-07-07T14-25-58Z
slug: internal-render-views-templ
---
# Design Critique — Pharos Dashboard

**Target:** dashboard (home) — `internal/render/views.templ` + `frame.templ`, live at `http://localhost:9090/`
**Register:** product · **Slug:** `internal-render-views-templ`

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | SSE live-sync present but no reconnect/loading indicator; silent on disconnect |
| 2 | Match System / Real World | 4 | Vocabulary (workspace/lesson/record/reference/glossary) consistent and well-chosen |
| 3 | User Control & Freedom | 4 | Cmd+K, 3-state theme toggle, sidebar collapse persistence, breadcrumbs, Esc, quiz Quit |
| 4 | Consistency & Standards | 3 | "Continue" rendered two ways: subtle inline link on dashboard, bordered card on workspace page |
| 5 | Error Prevention | 3 | Quiz Quit has no visible confirm; otherwise few destructive paths |
| 6 | Recognition Rather Than Recall | 3 | Sidebar hidden on dashboard; nav is workspace list + Cmd+K (which is itself undiscoverable) |
| 7 | Flexibility & Efficiency | 3 | Cmd+K + recents + scroll persistence; but no shortcut hint, no j/k, Continue is mouse-only |
| 8 | Aesthetic & Minimalist Design | 4 | Nord restraint, hairlines over cards, no ornament — genuinely calm |
| 9 | Error Recovery | 2 | `swapRegion` does `.catch(function(){})` — silent failure, no "couldn't reach server" state |
| 10 | Help & Documentation | 2 | About page + empty-state commands; no in-app keyboard cheatsheet, no ⌘K hint anywhere |
| **Total** | | **31/40** | **Good — solid foundation, address weak areas** |

## Anti-Patterns Verdict

**Would a user fluent in Linear/Notion/Raycast trust this, or pause?** They'd trust it. This reads hand-crafted.

**LLM assessment:** The dashboard avoids every absolute ban. No gradient text, no glassmorphism-as-default (blur only on overlays — appropriate), no hero-metric tiles (the stat row is inline prose, not big-number cards), no identical icon+heading+text card grid (hairline-divided rows instead), no uppercase-tracked eyebrow on every section, no numbered scaffolding, no sketchy SVG, no stripe/grid backgrounds, no meta-criticism copy. Card radius stays at 8px (`rounded-lg`); 12px only on the palette panel. The sidebar active-link 2px `border-left` (`input.css:372`) is a legitimate nav active marker (VS Code/Linear pattern), and the `.callout` 4px `border-left` is a scoped emphasis block — neither is the decorative side-stripe tell. The Nord commitment and "hairlines over cards" discipline read as deliberate authorship.

**Deterministic scan (detect.mjs + browser overlay):** The detector ran in the live page (overlays visible in the browser). Findings:

| # | Rule | Severity | Verdict |
|---|---|---|---|
| 1 | `gpt-thin-border-wide-shadow` on `.palette-panel` (1px border + 32px shadow) | warn | **False positive.** Standard modal treatment (`input.css:776-778`); the 0.71px reading is a `transform: scale(0.96)` artifact. Not the ghost-card pattern. |
| 2 | `low-contrast` — `#81a1c1` on `#353b4a` = 4.2:1 (dark mode) | warn | **Real.** Just under AA 4.5:1. Affects the stat row + metadata. |
| 3 | `overused-font` — Inter (100% of text) | warn | **False positive for product register.** `reference/product.md`: "One family is often right." Inter-only is correct, not a defect. |
| 4 | `single-font` — no display/body pairing | warn | **False positive for product register.** Same reasoning. |
| 5 | `flat-type-hierarchy` — 12/14/16px, ratio 1.3:1 | warn | **Borderline.** Product UI targets 1.125–1.2 ratios (`reference/product.md`); 1.3:1 is within product norm, but the dashboard's *load-bearing* rhythm (stat row, Continue, titles) is genuinely flat. Worth a typeset pass, not a defect. |

The detector caught the dark-mode contrast failure that the LLM review's light-mode contrast finding complemented — the two assessments agree contrast is the real issue (see Priority Issue #2).

**Visual overlays:** The detector's `✦` annotation panels are live in the browser tab, highlighting: the palette-panel border+shadow, the low-contrast stat row, and the font/hierarchy findings. You can see them now.

## Overall Impression

A calm, well-disciplined dashboard that honors its own "hairlines over cards" and "empty states teach" principles — rare. It clears the AI-slop test comfortably. What holds it back from excellent is two things: (1) the **primary action — resuming where you left off — is visually undersized and buried beneath a stat row that's noise for returning users**, and (2) **muted text fails WCAG AA in both themes** (seriously in light mode, marginally in dark). Both are fixable in a single pass. The single biggest opportunity: make "Continue" the hero of the dashboard.

## What's Working

1. **Hairlines over cards.** The workspace list (`views.templ:34-48`, `divide-y divide-slate-100`) resists the boxed-card default. The reading surface stays quiet — matching "calm, focused, expert." A stated DESIGN.md principle the code actually honors.
2. **Empty states teach the next action.** The empty dashboard (`views.templ:49-81`) shows literal agent prompts (`"Teach me about topic"`) in a code chip, closing the loop on what will happen. The product's defining idea made visible at the moment of highest confusion.
3. **Dark mode as first-class.** Every color is a token; `[data-theme="dark"]` flips luminance while hue stays; the FOUC-prevention script runs before paint; iframes sync via postMessage. For a primary persona who studies in the evening, this is load-bearing craft.

## Priority Issues

### [P0] "Continue" is undersized for its stakes
- **What:** The dashboard Continue link (`views.templ:24-32`) is a `text-sm` inline link with a `→` glyph, sitting *below* the 5-item stat row. Confirmed in the live page: stat row → Continue link → workspace list.
- **Why it matters:** For the primary persona (returning evening learner), resuming is the #1 action. The product's stated success metric is "sustained retention — users keep returning." An undersized resume affordance directly undermines the core loop. The workspace-page Continue (`views.templ:125-133`) is a proper bordered card — the dashboard version should match, not shrink beneath it. The label `data-visualization — Lesson: Bar Charts...` also reads like a system log (uses the slug, not the human title).
- **Fix:** Promote to a bordered card matching `views.templ:126-132`; reorder above the stat row (or remove the stat row — see Questions); replace `→` with a Lucide `arrow-right`; lead with the human-readable lesson title, not the slug.
- **Suggested command:** `$impeccable clarify`

### [P1] Muted text fails WCAG AA in both themes
- **What:** `text-slate-400` in **light** mode resolves to Tailwind's default `#94a3b8` (~2.6:1 on white) because `--color-slate-400` is defined only under `[data-theme="dark"]` (`input.css:40`), not in `:root`. In **dark** mode, `#81a1c1` on the panel = 4.2:1 (detector-measured). Affects the stat row, workspace metadata, last-studied dates, and the hardcoded `#94a3b8` on `.sidebar-section-label`/`.sidebar-section-count` (`input.css:287,316`).
- **Why it matters:** The committed bar is WCAG AA (`PRODUCT.md`). Metadata is readable content — last-studied dates are how a returning user orients. The LLM review caught light-mode; the detector caught dark-mode — both real.
- **Fix:** Define `--color-slate-400` in `:root` at a darker value (e.g. `#64748b`, ~4.6:1 on white); nudge the dark `--color-slate-400` up to clear 4.5:1; replace the two hardcoded `#94a3b8` with the token.
- **Suggested command:** `$impeccable harden`

### [P1] Cmd+K is undiscoverable
- **What:** The command palette exists and is keyboard-complete (`frame.templ:527-845`), but no surface hints at `⌘K`. The search placeholder is `"Search..."` (`frame.templ:45`).
- **Why it matters:** A user fluent in Linear/Raycast expects Cmd+K — and won't find it without a hint. The product bar is "earned familiarity." Power users should discover this in seconds, not by accident. Also: the palette's quick actions are only "Toggle sidebar"/"Toggle theme" (`frame.templ:634-639`) — no "Continue last lesson," the actual primary action.
- **Fix:** Add a `⌘K` kbd hint inside the search input (Linear-style: `Search… ⌘K`); add a "Continue last lesson" quick action to the palette.
- **Suggested command:** `$impeccable clarify`

### [P2] Inconsistent focus-visible indicators
- **What:** Buttons have `focus:ring-2 focus:ring-blue-700` (`views.templ:242`), but dashboard links (workspace rows `views.templ:36`, Continue `views.templ:26`) have no `:focus-visible` styling — they fall back to the browser default outline. No global `:focus-visible` rule in `input.css`.
- **Why it matters:** Keyboard users get rings on buttons, default outlines on links — inconsistent, breaking "the tool disappears into the task."
- **Fix:** Add a global `:focus-visible` rule (`outline: 2px solid var(--color-blue-700); outline-offset: 2px`) or extend `focus:ring-2` to the link classes.
- **Suggested command:** `$impeccable polish`

### [P2] Silent failure on SSE live-sync
- **What:** `swapRegion`'s fetch does `.catch(function(){})` (`frame.templ:481`), swallowing errors. No SSE-disconnect UI, no retry indicator.
- **Why it matters:** The brand is "calm" — calm comes from legibility, not hidden failures. If the agent pushes an update and the fetch fails, the user sees nothing; trust erodes silently.
- **Fix:** Surface a subtle status pill ("sync paused — retrying") on failure; retry with backoff; wire `EventSource.onerror`.
- **Suggested command:** `$impeccable harden`

## Persona Red Flags

**Alex (impatient power user):** Cmd+K exists but is undiscoverable — no `⌘K` hint anywhere, and Alex won't guess it. The palette's quick actions don't include the primary action ("Continue last lesson"), so Alex can't resume from the keyboard at all. The "Continue" label format `slug — Lesson: Title` reads like a system log. Alex will tolerate the hairline density but the missing power path is a real abandonment risk for this persona.

**Sam (accessibility-dependent):** Muted `text-slate-400` metadata fails AA (~2.6:1 light / 4.2:1 dark) — Sam can't read last-studied dates or workspace counts comfortably. Focus indicators are inconsistent: rings on buttons, browser-default outlines on the workspace list links and Continue (no global `:focus-visible`). Color is NOT alone (icons + text labels back up the 4 type hues) — that's solid. `prefers-reduced-motion` is honored. Keyboard tab order is logical. The contrast + focus gaps are the failures.

**Maya (returning evening self-learner):** Dark mode is genuinely first-class — token flip, FOUC prevention, iframe sync, scroll persistence across sessions (`frame.templ:419-465`). Her evening study experience is well-supported. BUT: the first thing her eye hits is the 5-item stat row (aggregate noise she doesn't care about), then a `text-sm` Continue link. For a tired returning user, the resume affordance should win the page — it currently loses to the stats. Fix this and Maya's loop tightens immediately.

## Minor Observations

- The empty dashboard state lists 3 content types (Lessons/Records/References, `views.templ:56-71`) but omits Quizzes — inconsistent with the stat row and sidebar. First-timers learn an incomplete mental model.
- The stat row (`views.templ:12-22`) is `flex` (not `flex-wrap`) with 5 items + 4 separators at `gap-6` — will overflow horizontally under ~480px. No responsive collapse.
- `text-slate-200` separators in the stat row (`#e5e9f0` on white) are nearly invisible — intentional quietness, but they barely read as separators.
- `uppercase tracking-wider` on "Previous attempts" (`views.templ:254`) and glossary category headers (`views.templ:356`) — the only uppercase eyebrows on content pages; minor off-stance per DESIGN.md §3.
- The PWA install button's perpetual `pwa-bounce` (`input.css:710-719`) is mildly attention-grabbing for a "calm" tool — borderline, though it stops on hover.
- The constellation logo at 28px in the topbar reads as "dots and lines"; the beacon metaphor is strong at full res but lost at display size.

## Questions to Consider

1. **Is the stat row earning its pixels for a returning user?** Maya doesn't care that she has "4 lessons across 3 workspaces" — she cares where she was. The counts live inside each workspace anyway. Would removing the aggregate stat row and letting "Continue" be the first thing the eye lands on make the dashboard calmer and the resume loop tighter? Or do first-time users need the counts to understand what a workspace contains?
2. **Should the dashboard keep hiding the sidebar?** Currently `frame.go:248` hides the sidebar when `ActiveWS == ""`, so dashboard nav is the workspace list or Cmd+K. Would a persistent "Recent workspaces" rail (always-visible sidebar with recents) reduce the cost to resume — or does the full-width reading layout win for calm?
3. **Would a human-readable Continue label feel less like a system log?** Dropping the slug and leading with the title ("Continue reading: Bar Charts & Scatter Plots") — calmer invitation, or does the slug help disambiguate multi-workspace users?
