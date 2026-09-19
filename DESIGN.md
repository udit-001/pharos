# DESIGN.md — Pharos Visual System

> Extracted from the codebase by `$impeccable extract`. Captures every committed
> design decision so future work stays on-brand. Source files are cited as
> `path:line` so the system can be audited against the implementation.
>
> **Register:** product (CLI tool + read-only web dashboard; design serves the product).

---

## 1. Design idea at a glance

Pharos is a **Nord-themed, reading-first learning dashboard**. The aesthetic is
*tool-native restraint*: a cool, low-chroma slate palette (the [Nord](https://www.nordtheme.com)
color system) carried through every surface — dashboard chrome, lesson iframes,
diagrams, charts, and code — so embedded content feels integral rather than
nested. There is no marketing layer; the whole surface is an app shell built for
focused study.

**The defining ideas:**
1. **One palette, three coexisting token systems** that all derive from Nord —
   the dashboard (`web/input.css`), Tailwind utility classes, and the
   iframe-resident lesson stylesheet (`internal/db/seed/style.css`).
2. **Dark mode is free, not bolted on.** A single `data-theme` attribute flips
   every variable; luminance reverses while hue stays constant.
3. **Content renders inside iframes** that theme-sync at runtime via
   `postMessage` — the dashboard owns navigation, pages own content.
4. **Retint, don't re-layout.** Diagrams/charts re-colour on theme toggle
   without re-running layout (mermaid copies `<style>`; vega re-renders from JSON).
5. **Vendored, offline-capable assets.** Inter font, mermaid, KaTeX, highlight.js,
   and vega-lite are all bundled per-workspace — no CDN at runtime.

---

## 2. Color — the Nord palette

The palette is [Nord](https://www.nordtheme.com) (hex, not OKLCH). It is a
16-color arctic, north-bluish palette built from four named groups. Pharos maps
them to semantic Tailwind-style token names.

### 2.1 Semantic roles

| Role | Light | Nord | Dark | Used for |
|---|---|---|---|---|
| Page bg | `#ffffff` | snow storm | `#3b4252` (nord1) | body / cards |
| Surface | `#eceff4` | nord6 | `#2e3440` (nord0) | code bg, sidebar bg |
| Border | `#e5e9f0` | nord4 | `#434c5e` (nord3) | dividers, card borders |
| Subtle divider | `#d8dee9` | nord4 | `#4c566a` (nord2) | table rows, hairlines |
| Body text | `#4c566a` | nord3 | `#aebbcf` | prose, labels |
| Strong text | `#3b4252` | nord1 | `#d8dee9` (nord4) | headings, bold |
| Heading | `#2e3440` | nord0 | `#eceff4` (nord6) | h1, page titles |
| Muted/caption | `#8891a0` | — | `#81a1c1` (nord9) | metadata, placeholders |
| **Accent / links** | `#5e81ac` | **nord9** | `#81a1c1` (nord9) | links, active state, primary btn |
| Success | `#4a7a2e` | — | `#95c088` (nord14-ish) | records, correct, active tag |
| Success bg | `#e6f0e6` | — | `#2e3440` | success tint |
| Error | `#bf4e5a` | — | `#e8a0a0` | superseded, incorrect |
| Error bg | `#fce4e4` | — | `#4c566a` | error tint |
| Warning/refs | `#d08770` | **nord12** | `#d08770` | references, attention |

> Nord's four groups: Polar Night (`#2e3440 #3b4252 #434c5e #4c566a`),
> Snow Storm (`#d8dee9 #e5e9f0 #eceff4 #f8fafc`), Frost
> (`#8fbcbb #88c0d0 #81a1c1 #5e81ac`), Aurora (`#bf616a #d08770 #ebcb8b #a3be8c #b48ead`).
> Pharos leans on Frost (nord9 `#5e81ac` as the single accent) and uses Aurora
> hues only inside diagrams (`cScale0..12` in `mermaid-theme.js:31-43`).

### 2.2 The two token vocabularies (extraction note)

The system has **two parallel sets of CSS variables** that must stay in sync:

| Surface | File | Prefix | Example |
|---|---|---|---|
| Dashboard chrome | `web/input.css:6-60` | `--color-*` | `--color-blue-700: #5e81ac` |
| Lesson iframe | `internal/db/seed/style.css:9-43` | `--slate-*` / `--blue-700` | `--blue-700: #5e81ac` |

Both define the *same Nord values* but under different names, because the
dashboard uses Tailwind v4 (`--color-*` is Tailwind's token convention) while
the iframe stylesheet is standalone. **Consolidation opportunity:** a single
shared `tokens.css` could feed both, eliminating the dual-maintenance surface.
See §10.

### 2.3 Contrast posture

WCAG AA is enforced by construction:
- Body/prose text uses `--slate-700`/`--slate-800` (not the muted `--slate-400`)
  for all readable copy; `--slate-400` is reserved for metadata/captions.

---

## 3. Typography

- **Family:** Inter, single sans-serif, weights 400/500/600/700
  (`web/input.css:8`, `frame.templ:27`). No serif, no display pairing — the
  contrast axis is weight, not family.
- **Delivery:** Google Fonts `<link>` in the dashboard
  (`frame.templ:25-27`); **vendored** `inter-latin.woff2` `@font-face` in lesson
  iframes (`seed/style.css:1-7`) so lessons work offline.
- **Prose scale** (shared by dashboard `.prose` and iframe base styles):

| Element | Size | Weight | Color | Letter-spacing |
|---|---|---|---|---|
| h1 | 1.375rem (22px) | 700 | slate-800 | -0.01em |
| h2 | 1.125rem (18px) | 600 | slate-800 | — |
| h3 | 1.0rem (16px) | 600 | slate-700 | — |
| body | 0.9375rem (15px) | 400 | slate-700 | — |
| line-height | 1.75 | | | |
| code | 0.875em | — | — | ui-monospace stack |

- **Page titles** (dashboard, non-prose): `text-xl font-semibold tracking-tight`
  — `tracking-tight` is the only display tightening; no `-0.04em+` letter-spacing.
- **Metadata/eyebrows:** `text-xs font-medium text-slate-400` (e.g.
  "Lessons" section headers, `views.templ:134`). Used sparingly as section
  labels, **not** as all-caps tracked eyebrows on every section.
- **Numerals:** `tabular-nums` on all counts (`views.templ:13-21`) so stats
  align in the dashboard header.
- **Max line length:** reading column `max-w-4xl` (~56rem) in the dashboard
  (`frame.go:291`); `.container { max-width: 56rem }` in lessons
  (`seed/style.css:55-59`).

---

## 4. Layout & the app shell

### 4.1 Frame structure

```
┌─────────────┬──────────────────────────────────┐
│             │ topbar  (min-h-12, bg-stone-50)   │
│  sidebar    ├──────────────────────────────────┤
│  240px      │                                   │
│  (or 60px   │  content                          │
│  collapsed) │  max-w-4xl mx-auto, p-6          │
│             │  (or p-0 + iframe, "frame" mode)  │
└─────────────┴──────────────────────────────────┘
```
Source: `frame.templ:32-65`, `frame.go:255-292`.

- **Sidebar:** `fixed md:relative`, 240px wide, collapses to 60px icon-rail
  (`web/input.css:391-433`). State persists in `localStorage('pharos_sidebar_collapsed')`.
  Section collapse state persists per-workspace-per-section
  (`frame.templ:187-201`).
- **Topbar:** `min-h-12 px-4 md:px-6 py-2.5 bg-stone-50 border-b`. Holds
  breadcrumbs (left), centered brand on dashboard only (`topbar-brand`,
  absolutely centered, `web/input.css:108-113`), and search + actions (right).
- **Content modes:** `frameMaxWidthClass` + `contentPaddingClass` switch
  between *reading* (`max-w-4xl p-6`) and *frame* (no padding, no max-width,
  iframe fills edge-to-edge) — used by lessons and references
  (`frame.go:277-292`).

### 4.2 Spacing rhythm

- Card padding: `p-4` to `p-6`; empty-state cards `p-10`.
- Section header rhythm: `mt-6 mb-3` on `h2` section labels.
- List rows: `py-2 border-b border-slate-100 last:border-0` — hairline
  dividers, no cards. This is the dominant list pattern
  (`views.templ:139,152,168,213`).
- `-mx-3` negative margin on hover-bg rows so the hover fill bleeds to the
  column edge (`views.templ:26,36`).

---

## 5. Components

### 5.1 Sidebar (`frame.go:37-118`, `web/input.css:283-510`)

A collapsible, sectioned nav. Key ideas:
- **Section label:** 0.6875rem, 0.04em tracking, flex with chevron + icon.
  Clicking toggles `max-height` collapse with a measured `scrollHeight`
  (JS-driven, not pure CSS, so it animates from dynamic content height).
- **Active link:** 2px `border-left` in accent blue + tinted bg `#e0e7ff`
  (light) / `rgba(129,161,193,0.15)` (dark). **Note:** this is the one
  intentional side-stripe in the system — a navigational active marker, not
  decorative card accent.
- **Collapsed mode:** icons only; section click opens a `position: fixed`
  **flyout** (`web/input.css:464-488`) rather than expanding the rail — avoids
  layout shift. JS-managed tooltips on hover (`frame.templ:218-242`).
- **Count badges:** `0.625rem`, `rgba(slate,0.15)` pill, right of section label.

### 5.2 Breadcrumbs (`frame.go:135-219`)

`Workspace / Item` trail. Dashboard is reachable via the logo, so it earns no
crumb. Quiz pages get a 3-level trail (`Workspace / Quizzes / Quiz`). Each crumb
is `truncate max-w-[40vw]`. Prefixed by a sidebar-collapse toggle button.

### 5.3 Command palette — Cmd+K (`frame.templ:527-845`, `web/input.css:746-897`)

A two-tier fuzzy launcher:
- **Tier 1:** inline JSON of the active workspace's items, scored by a
  subsequence matcher (exact < prefix < substring < subsequence, tighter
  groupings win; `frame.templ:575-593`).
- **Tier 2:** debounced (150ms) `/api/search` appended under a
  "More across workspaces" divider.
- **Visuals:** `position: fixed` overlay, `backdrop-filter: blur(4px)`, panel
  `scale(0.96)→1` + opacity fade-in (100ms ease-out), `border-radius: 12px`,
  `box-shadow: 0 8px 32px rgba(0,0,0,0.18)`. Selected row gets
  `rgba(94,129,172,0.12)` tint. Recents from localStorage populate the empty
  state. Suppressed on in-progress quiz attempts.
- **Grouping:** results grouped by type in a fixed order
  (lesson, record, ref, quiz, doc, workspace, action, home) with plural labels.

### 5.4 Lightbox pattern (`web/input.css:634-708`)

Two consumers share one visual language — a small white bordered square expand
button (28px, `border-radius:6px`, opacity 0.4→1 on hover) and a full-screen
overlay (`rgba(0,0,0,0.4)` + `backdrop-filter: blur(4px)`, panel `90vw×90vh`,
`border-radius:8px`, `box-shadow:0 4px 24px rgba(0,0,0,0.12)`, top-right close):
- **Stimulus lightbox** (`views.templ:304-320`, `frame.templ:310-360`) —
  iframe payload, no zoom/pan.
- **Mermaid lightbox** (`internal/cli/mermaid-lightbox.*`) — SVG payload with
  zoom/pan toolbar.

### 5.5 Badges & tags

| Class | Light | Dark | Meaning |
|---|---|---|---|
| status `active` | emerald-100 / emerald-600 | (same) | record status |
| status `superseded` | red-100 / red-600 | (same) | record status |
| quiz score | emerald (perfect) / amber (partial) / slate (not started) | | `views.go:259-267` |

Pattern: `inline-flex items-center text-xs font-medium px-2 py-0.5 rounded`.
The four content types each own a hue, reused across sidebar icon and stat
number (blue=lesson, emerald=record, amber=ref) — **color is the type system.**

### 5.6 Buttons

- **Primary:** `bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-lg
  hover:bg-blue-700/90 focus:ring-2 focus:ring-blue-700`
  (`views.templ:242,247,341`).
- **Secondary/ghost:** `border border-slate-200 hover:border-slate-300
  text-slate-500 py-2 px-4 rounded-lg` (`views.templ:273,278`).
- **Icon button (topbar):** `p-1.5 rounded hover:bg-slate-200 text-slate-600`
  (`frame.templ:48-52`).
- **Lesson inline button** (`seed/style.css:196-206`): `bg-blue-700`, `8px`
  radius, `opacity:0.9` on hover (no border, no shadow).

`border-radius: 8px` (lg) is the button radius; `6px` for small controls;
`12px` only for the command palette panel. Cards top out at `8px`/`lg`.

### 5.7 Empty states (`views.templ:49-81`, `views.go:178-219`)

A consistent recipe: centered, `py-20`/`p-10`, a `bigIcon` (40–48px Lucide in
`text-slate-300`), a one-line title + helper, and a *prompt block* showing the
exact agent command to run (e.g. `"Teach me about topic"` in a `bg-slate-100`
code chip). The empty state *teaches the user how to fill it* — a core
product idea, since Pharos is agent-driven.

### 5.8 Quiz review (`views.templ:286-344`, `views.go:83-125`)

Collapsible per-question rows. Correct/incorrect shown via 24px filled circles
(emerald `✓` / red `✗`) — the *only* place saturated red/green circles appear.
Choice review highlights the correct option with `border-2 border-emerald-600
bg-emerald-100` and the user's wrong pick with `border-2 border-red-600
bg-red-100`; unchosen options stay neutral. Recall review uses a dashed-border
`bg-slate-50` reveal box.

### 5.9 Lesson content components (`seed/style.css:174-194`)

These live *inside* the iframe and are the author-facing component library:

| Component | Class | Visual |
|---|---|---|
| Inline quiz | `.q` + `.options` + `.fb` | `bg-slate-50`, `8px` radius, `1px` border; buttons 6px radius; `.correct`/`.incorrect` tint |
| Callout (key takeaway) | `.callout` | `border-left: 4px solid var(--blue-700)`, `bg-slate-50`, `0 8px 8px 0` radius |
| Source box (further reading) | `.source-box` | `bg-slate-100`, `8px` radius, no border |
| Glossary term | `.glossary-term` | dotted underline in blue-700, 8% tint on hover |

> The `.callout` uses a 4px left accent border — a deliberate exception for
> emphasis blocks. It is *not* replicated on cards or list items (which would
> violate the side-stripe ban); it is scoped to a single semantic component.

---

## 6. Motion

### 6.1 Easing tokens (`web/input.css:27-30`)

| Token | Curve | Used for |
|---|---|---|
| `--ease-out` | `cubic-bezier(0.23, 1, 0.32, 1)` (ease-out-quint) | hover color/opacity transitions |
| `--ease-in-out` | `cubic-bezier(0.77, 0, 0.175, 1)` | chevron rotation |
| `--ease-drawer` | `cubic-bezier(0.32, 0.72, 0, 1)` | sidebar collapse, section expand |
| `--ease` | `ease` | fast micro-interactions (section label color) |

No bounce, no elastic. Exponential ease-outs throughout. Durations stay short:
0.1–0.25s for chrome, 0.15s for icon-button feedback.

### 6.2 Transition patterns

- **Color/hover:** `transition: background 0.1s, color 0.1s, border-color 0.1s`
  (sidebar links) or `transition-colors` (Tailwind, dashboard rows).
- **Drawer:** sidebar width/transform on `--ease-drawer` 0.25s; section
  `max-height` 0.25s on `--ease-drawer`.
- **Theme toggle icon:** crossfade + scale(0.4) + per-icon rotation (sun +90°,
  moon −90°) on 0.15s ease-out (`web/input.css:721-744`) — gives the toggle
  character without bounce.
- **Command palette:** overlay opacity 0.1s + panel `scale(0.96)→1` 0.1s.
- **Glossary tooltip:** opacity + `scale(0.95)→1` on ease-out-quint 0.15s,
  origin flips based on position (`seed/style.css:291-315`).
- **PWA install button:** `pwa-bounce` keyframe (translateY -3px, 1s) — the
  one intentional bounce, and it stops on hover (`web/input.css:710-719`).

### 6.3 Reduced motion (`web/input.css:62-80`, `seed/style.css:336-340`)

Every animated surface gets a `@media (prefers-reduced-motion: reduce)` block
that force-disables transitions on sidebar, palette, chevrons, theme toggle,
and glossary tooltips (tooltip keeps opacity fade, drops transform). Active
sidebar link scroll uses `behavior: 'auto'` under reduced motion
(`frame.templ:206-207`).

### 6.4 Retint, don't re-layout (the mermaid idea)

On theme toggle, mermaid diagrams are **retinted, not re-rendered**: a
throwaway render produces a fresh `<style>`, which is copied into the live SVG
(plus gradient stops), leaving geometry/viewBox untouched
(`mermaid-theme.js:104-110`, AGENTS.md "Mermaid theming"). A plain hex-swap
fails on mindmap nodes (emitted as `hsl()`), so the copy-`<style>` approach is
load-bearing. Vega-Lite, by contrast, re-renders cleanly from its JSON spec
on toggle (no cached SVG). KaTeX needs no retint — it inherits `currentColor`.

---

## 7. Iconography

- **Library:** [Lucide](https://lucide.dev), inline SVG, 16×16, `stroke-width: 2`,
  `stroke-linecap: round`, `stroke-linejoin: round`, `fill: none`,
  `stroke: currentColor`, `class="shrink-0"`.
  Source: `internal/render/icons.go` (one func per icon, returns the SVG
  string). The command palette duplicates the set inline in JS
  (`frame.templ:596-604`).
- **Empty-state icons:** same Lucide set, scaled to 40–48px via `bigIcon()`
  (`views.go:199-205`), in `text-slate-300`.
- **Logo:** a **constellation node network** — a central filled circle with 4
  satellite nodes connected by lines (`icons.go:102-117`, inline 28px). The
  full-res `design/logos/pharos-logo.svg` is a 23-ray compass/ Pharos-lighthouse
  emanation. The logo carries the product metaphor: a beacon mapping knowledge
  nodes.
- **Chevrons:** inline 14px in sidebar, rotated via `transform` JS, not
  swapped SVGs.

---

## 8. Theming architecture

### 8.1 The FOUC-prevention script

A blocking inline `<script>` in `<head>` reads `localStorage('pharos_theme')`,
resolves `'system'`/`null` via `matchMedia('(prefers-color-scheme:dark)')`, and
sets `document.documentElement.dataset.theme` **before paint**
(`frame.templ:29`). The same script runs in lesson iframes
(`PAGE-THEME.md:83`). No `data-theme` is ever hardcoded on `<html>`.

### 8.2 Token flip

`[data-theme="dark"]` (`web/input.css:33-60`, `seed/style.css:27-43`) redefines
every variable. Luminance reverses; hue stays. The `--color-white` token itself
becomes `#3b4252` (nord1) in dark mode — "white" is semantic (the page bg
role), not literal.

### 8.3 Iframe sync

Dashboard toggle sends `postMessage({type:'theme', theme})` to every iframe;
each lesson page has a listener (`PAGE-THEME.md:118`). The `theme-color` meta
tag is also updated for mobile browser chrome (`frame.templ:20,29`).

### 8.4 Three-state toggle

The theme toggle cycles light → dark → system, with three icons (moon/sun/
contrast) crossfading (`frame.templ:52-56`, `web/input.css:721-744`).

---

## 9. Content rendering stack

Lessons are authored as HTML (goldmark markdown → HTML) and rendered inside
iframes. The rendering stack is **on-demand and vendored per-workspace**:

| Concern | Asset | Theming | Add command |
|---|---|---|---|
| Base styles | `assets/style.css` (seeded) | CSS vars (auto) | (auto) |
| Inter font | `assets/fonts/inter-latin.woff2` | — | (auto) |
| Code highlight | `highlight.min.js` + `highlight.css` | Nord token CSS vars | `pharos asset add highlightjs` |
| Math (KaTeX) | `katex.*` + `katex-render.js` | inherits `currentColor` | `pharos asset add katex` |
| Diagrams (mermaid) | `mermaid.min.js` + `mermaid-theme.js` + lightbox | retint on toggle | `pharos asset add mermaid` |
| Charts (vega-lite) | `vega*.min.js` + `vega-theme.js` | re-render from JSON | `pharos asset add vega` |
| Copy code | `copy-code.js` (seeded) | — | (auto) |
| Glossary tooltips | `glossary-tooltip.js` (seeded) | CSS vars | (auto) |

**Authoring convention — pure JSON, zero JS** for vega: a `<script
type="application/json" id="x">` spec paired with `<div class="chart"
data-vega="x">`. The companion auto-discovers and renders. Specs never hardcode
colours; they come from the Nord config.

KaTeX delimiters: `$...$` / `$$...$$` / `\(...\)` / `\[...\]`.

---

## 10. Design principles (extracted)

1. **One palette everywhere.** Nord spans dashboard, iframe, diagrams, charts,
   and code — embedded content never looks embedded.
2. **Dark mode is free.** Use variables, get dark mode automatically; never
   hardcode a hex.
3. **The dashboard owns navigation; pages own content.** No dashboard chrome
   inside iframes; lessons are self-contained.
4. **Retint, don't re-layout.** Theme toggle changes colour, never geometry.
5. **Vendored and offline.** No CDN at runtime; assets ship per-workspace.
6. **Color is the type system.** The four content types (lesson/record/ref/quiz)
   each own a hue, reused across sidebar icon and stat number.
7. **Empty states teach.** They show the exact agent command that fills them.
8. **Hairlines over cards.** Lists use `border-b border-slate-100`, not
   boxed cards — keeps the reading surface quiet.

---

## 11. Extraction opportunities (consolidation backlog)

These are *observed duplications*, not yet refactored. Each is a candidate for
`$impeccable extract` to consolidate:

1. **Dual token vocabularies.** `web/input.css` (`--color-*`) and
   `seed/style.css` (`--slate-*`) define the same Nord values under different
   names. A shared `tokens.css` could feed both. **Highest value.**
2. **Icon duplication.** `icons.go` and the command-palette JS
   (`frame.templ:596-604`) each carry the Lucide set inline. A single
   icon registry (Go map → JSON) could feed both server and client.
3. **Lightbox pattern duplication.** `.stimulus-expand` / `.stimulus-lightbox`
   and the mermaid lightbox reimplement the same overlay+panel+close-button
   visual. A shared `.lightbox` primitive would unify them.
4. **Copy-code-button duplication.** The copy button CSS exists in both
   `web/input.css:192-244` (dashboard `.prose pre`) and
   `seed/style.css:117-169` (lesson `pre`), near-identical. Could be one
   shared class.
5. **Prose duplication.** `.prose` rules in `web/input.css:116-281` mirror the
   base element styles in `seed/style.css:61-111`. A shared prose stylesheet
   would close the gap.
6. **Badge dark-mode lifting.** The manual per-badge dark tints
   (`web/input.css:539-554`) could be a single `color-mix(in srgb, ...)` formula
   (already used for glossary hover, `seed/style.css:287`) to auto-lift any
   hue to AA.

---

## 12. File map

| Concern | File |
|---|---|
| Dashboard tokens + custom CSS | `web/input.css` |
| Tailwind input | `web/input.css:1` |
| App frame (templ) | `internal/render/frame.templ` |
| Frame helpers (sidebar, breadcrumbs) | `internal/render/frame.go` |
| Views (dashboard, quiz, etc.) | `internal/render/views.templ`, `views.go` |
| Icons + logo | `internal/render/icons.go` |
| Lesson iframe stylesheet | `internal/db/seed/style.css` |
| Mermaid Nord theme | `internal/cli/mermaid-theme.js` |
| Theme toggle JS | `internal/web/pharos-theme.js` |
| Embed registry | `internal/cli/asset_registry.go` |
| Theme doc (lessons) | `internal/skills/teach/PAGE-THEME.md` |
| Logo assets | `design/logos/pharos-logo.svg` |
