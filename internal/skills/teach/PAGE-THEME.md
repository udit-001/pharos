# Page Theme — Nord-Inspired Design System

Lessons and references render inside an iframe within the Pharos dashboard. They must match the dashboard's Nord-inspired palette so they feel integral to the app, not embedded.

The dashboard controls theme via `data-theme` attribute on `<html>` — light or dark. HTML pages sync by reading `localStorage` on load and listening for `postMessage` theme events at runtime.

---

## Architecture

| Concern | Mechanism |
|---|---|
| Palette | CSS custom properties on `:root` / `[data-theme="dark"]` |
| FOUC prevention | Blocking `<script>` in `<head>` reads `localStorage('pharos_theme')`, falls back to `prefers-color-scheme`, sets `data-theme` |
| Runtime theme sync | `postMessage` listener — dashboard sends `{type:'theme', theme:'dark'|'light'}` to iframes on toggle |
| Shared styles | `assets/style.css` (variables, typography, layout, component classes); `assets/quiz.css` added per-workspace when a lesson has quizzes |
| Quiz interactivity | Inline `<script>` before `</body>` — binds to `.q` elements |
| Font delivery | `@font-face` in `assets/style.css` → `assets/fonts/inter-latin.woff2` (vendored — works offline, no CDN) |
| Copy code | `assets/copy-code.js` — adds copy button to `<pre>` on hover; opt-out per block via `data-no-copy` |

---

## Nord Palette (CSS Variables)

These are the shared variables every HTML page must use — never hardcoded color values.

### Light mode (`:root`)

| Variable | Nord | Usage |
|---|---|---|
| `--slate-900` | `#2e3440` | Headings |
| `--slate-800` | `#3b4252` | Strong emphasis |
| `--slate-700` | `#4c566a` | Body text |
| `--slate-500` | `#6b7689` | Muted / secondary text |
| `--slate-400` | `#8891a0` | Metadata / captions |
| `--slate-200` | `#e5e9f0` | Borders / dividers |
| `--slate-100` | `#eceff4` | Code / blockquote bg |
| `--slate-50` | `#f8fafc` | Subtle highlight |
| `--white` | `#ffffff` | Page background |
| `--blue-700` | `#5e81ac` | Links / accent |
| `--emerald-600` | `#4a7a2e` | Success / correct |
| `--emerald-100` | `#e6f0e6` | Success background |
| `--red-600` | `#bf4e5a` | Error / incorrect |
| `--red-100` | `#fce4e4` | Error background |
| `--amber-600` | `#d08770` | Warning / attention |

### Dark mode (`[data-theme="dark"]`)

Reverse the luminance: backgrounds become dark, text becomes light, keeping Nord's overall contrast ratio.

| Variable | Nord | Usage |
|---|---|---|
| `--slate-900` | `#eceff4` | Headings |
| `--slate-800` | `#d8dee9` | Strong emphasis |
| `--slate-700` | `#aebbcf` | Body text |
| `--slate-500` | `#94adcb` | Muted / secondary text |
| `--slate-400` | `#81a1c1` | Metadata / captions |
| `--slate-200` | `#434c5e` | Borders / dividers |
| `--slate-100` | `#353b4a` | Code / blockquote bg |
| `--slate-50` | `#2e3440` | Subtle background |
| `--white` | `#3b4252` | Page background |
| `--blue-700` | `#81a1c1` | Links / accent |
| `--emerald-600` | `#95c088` | Success / correct |
| `--emerald-100` | `#2e3440` | Success background |
| `--red-600` | `#e8a0a0` | Error / incorrect |
| `--red-100` | `#4c566a` | Error background |
| `--amber-600` | `#d08770` | Warning / attention |

---

## Required Boilerplate

Every HTML page — lessons and references alike — starts with this boilerplate. It links the shared stylesheet, prevents theme flash, and wires up runtime theme sync. A reference that omits it renders unstyled.

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Page Title</title>
<link rel="stylesheet" href="assets/style.css">
<script>(function(){var t=localStorage.getItem('pharos_theme');if(!t){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t})()</script>
</head>
<body>

<div class="container">

  <!-- lesson content -->

</div>

<script>
(function(){
  document.querySelectorAll('.q').forEach(function(q){
    var answer = q.getAttribute('data-answer');
    var buttons = q.querySelectorAll('button');
    var fb = q.querySelector('.fb');
    buttons.forEach(function(btn){
      btn.addEventListener('click', function(){
        buttons.forEach(function(b){ b.disabled = true; });
        if (btn.textContent.trim() === answer){
          btn.classList.add('correct');
          fb.textContent = 'Correct.';
        } else {
          btn.classList.add('incorrect');
          buttons.forEach(function(b){
            if (b.textContent.trim() === answer) b.classList.add('correct');
          });
          fb.textContent = 'Not quite — the right one is highlighted.';
        }
      });
    });
  });
})();
</script>
<script src="assets/copy-code.js"></script>
<script>window.addEventListener('message',function(e){if(e.data&&e.data.type==='theme')document.documentElement.dataset.theme=e.data.theme})</script>
</body>
</html>
```

Key rules:
- **No `data-theme` on `<html>`** — the blocking script sets it dynamically
- **Scripts in order**: FOUC prevention in `<head>`, then before `</body>`: quiz logic (lessons only), optional glossary tooltip (`<script src="assets/glossary-tooltip.js">`), `<script src="assets/copy-code.js">`, and postMessage listener
- **CSS links are root-relative** — no `../`, the iframe serves from `/api/lesson-html/<workspace>/<file>` or `/api/ref-html/<workspace>/<file>`

---

## Vendored Assets

Assets that ship with every workspace — no CDN required. The seed writes them to the workspace `assets/` directory at creation time.

### Inter font

The Inter variable font (latin subset, weight range 100–900) is bundled as `assets/fonts/inter-latin.woff2`. The `@font-face` declaration in `assets/style.css` loads it locally — no Google Fonts `<link>` needed. Just use `font-family: 'Inter'` in CSS (already the default in the boilerplate).

### Mermaid (on-demand)

For flowcharts, sequence diagrams, and other diagrams in lessons. Not auto-seeded — add it when needed:

```
pharos asset add mermaid
```

`pharos asset add mermaid` writes `mermaid.min.js` (downloaded from CDN) and `mermaid-theme.js` (Nord dark/light theme support) to `assets/`. Add the lightbox separately:

```
pharos asset add mermaid-lightbox
```

Include in the lesson `<head>`:

```html
<script src="assets/mermaid.min.js"></script>
<script src="assets/mermaid-theme.js"></script>
<link rel="stylesheet" href="assets/mermaid-lightbox.css">
<script src="assets/mermaid-lightbox.js"></script>
<script>
document.addEventListener('DOMContentLoaded', function() {
  window.mermaid.initialize({startOnLoad:false,theme:'base',themeVariables:window.mermaidTheme.themeVars()});
  window.mermaid.run({
    querySelector: '.mermaid',
    postRenderCallback: function(id) {
      var svg = document.getElementById(id);
      var el = svg && svg.closest ? svg.closest('.mermaid') : null;
      if (!el && svg) {
        var p = svg.parentNode;
        while (p && p !== document.body) {
          if (p.classList && p.classList.contains('mermaid')) { el = p; break; }
          p = p.parentNode;
        }
      }
      if (el && window.mermaidLightbox) window.mermaidLightbox.addToolbar(el);
    }
  });
});
</script>
```

> **Why `theme: 'base'`?** The `base` theme lets `themeVariables` control all diagram colors, enabling dark/light mode switching. On theme toggle, `mermaid-theme.js` re-renders each diagram to a throwaway via `mermaid.render` and copies only its `<style>` (and gradient stops) into the live SVG — this *retints* without re-running the layout, so the diagram doesn't jump size. (A plain hex string-swap can't retint mindmap nodes, which mermaid emits as `hsl()` derived from `primaryColor`; the dark palette also pins `cScale0..12` + `git0` to Nord frost/aurora hues so each mindmap branch is a distinct colour instead of collapsing to near-black.)
> **Why the `<script>` is wrapped in `DOMContentLoaded`?** `mermaid-theme.js` reads `document.documentElement.dataset.theme` on load to pick the correct initial color palette. Wrapping in `DOMContentLoaded` ensures the FOUC-prevention script (which sets `data-theme`) has already run. `mermaid-theme.js` also captures each `.mermaid` div's source into `data-mermaid-src` before mermaid replaces it, so retint can re-render later.

Wrap diagrams in a `<div class="mermaid">` with the diagram text inside. Container background styles (rounded, padded, dark/light swap) are auto-seeded in `assets/style.css` — no extra `<style>` needed. Diagrams are capped at `max-height: 65vh` so tall vertical flowcharts don't consume the whole page; the expand button (lightbox) opens them full-size for readable detail.

### Highlight.js (on-demand)

For syntax highlighting in code blocks. Add it once — the command downloads the JS and generates `highlight.css` (token colors using Nord variables that switch with light/dark mode):

```
pharos asset add highlightjs
```

Include in the lesson `<head>`:

```html
<link rel="stylesheet" href="assets/highlight.css">
<script src="assets/highlight.min.js"></script>
<script>hljs.highlightAll();</script>
```

Code block language is auto-detected, but explicit `<pre><code class="language-js">` is recommended for accuracy.

### Adding a new vendored asset

1. Place the file in `internal/db/seed/` (or a subdirectory like `fonts/`).
2. Add a `//go:embed` directive in `internal/db/seed.go`.
3. Wire it into the seed write loop in `seedWorkspaceDefaults` (following the text-files or binary-assets pattern).
4. For fonts, declare the corresponding `@font-face` in `seed/style.css`.

---

## Component Patterns (lessons only — design free)

These are the functional building blocks for interactive lessons. References typically don't need them — they're cheat sheets, not interactive content. The teach skill should create appropriate CSS for each.

### Quiz

An inline knowledge check. Structure: a container `.q` with `data-answer` attribute, multiple `<button>` options, and an `.fb` feedback element. On click: disable all buttons, compare clicked text to `data-answer`, mark correct/incorrect, show feedback text.

Classes: `.correct` (green), `.incorrect` (red) — applied to buttons after answer.

The `.fb` element starts empty and only fills on click — hide it with `.fb:empty{display:none}`. Without this rule the padding and background render an empty box under every quiz before any answer is selected.

### Callout

A key takeaway or insight. Visually distinct from body paragraphs — uses accent border/background to draw attention.

### Source box

A recommendation for further reading. Subtle background, distinguishes it from callout.

### Tables

Standard data tables with bordered rows, left-aligned headers, muted dividers.

### Diagrams

Interactive diagrams rendered with Mermaid. Wrap diagram text in a `<div class="mermaid">`:

```html
<div class="mermaid">
flowchart TD
  A[Start] --> B{Decision}
  B -->|Yes| C[Result]
  B -->|No| D[Alternate]
</div>
```

`mermaid-theme.js` already sets `fontFamily: 'Inter, sans-serif'` in both light and dark palettes — no extra `themeVariables` needed for font matching.

### Buttons

Used for quiz options and any clickable action. Rounded, filled, hover feedback.

---

## Layout

- `.container` — centered, max-width matches the dashboard's reading column (~56rem), padded
- HTML pages are self-contained — no prev/next nav, the dashboard sidebar handles sequencing
- Content is single-column, stacked vertically

---

## Contextual Links (Iframe Escape)

Links that navigate outside the page (to any dashboard page) must use `target="_top"` with an absolute route. See [references/pharos-cli.md](references/pharos-cli.md) for the complete route table — never guess a URL pattern. Relative links like `../lesson/0002.html` load inside the iframe and lose the dashboard chrome.

---

## Glossary Tooltips

When the workspace has glossary terms, wrap each occurrence of a term in page prose with a `<span class="glossary-term" data-term="TermName">TermName</span>` so the reader gets a hoverable definition preview. This applies to both lessons and references — any HTML page that uses workspace terminology.

**Convention:**

```html
The <span class="glossary-term" data-term="Hypertrophy">Hypertrophy</span>
response drives muscle growth.
```

**Tooltip CSS + JS** — both are seeded in `assets/` at workspace creation and linked automatically by pages that already include `<link rel="stylesheet" href="assets/style.css">`. The only addition needed is the script reference before `</body>` (before the postMessage listener):

```html
<script src="assets/glossary-tooltip.js"></script>
```

The JS fetches definitions from `GET /api/workspaces/name/{name}/glossary-terms` at runtime — no definitions are baked into the page HTML. The workspace name is extracted from the iframe URL automatically.

**Don't wrap every occurrence.** Use judgement: wrap the first occurrence in a section, or where re-reading the definition aids understanding. Over-wrapping makes text noisy and trains readers to ignore tooltips.

---

## Copy Code

Every `<pre>` block gets a copy button on hover (auto-seeded `assets/copy-code.js`, no extra step needed).

Blocks where typing builds storage strength — skill-phase exercise code — use `data-no-copy` to preserve desirable difficulty:

```html
<pre data-no-copy><code class="language-go">...</code></pre>
```

---

## Principles

1. **Everything uses CSS variables**, never hardcoded hex values
2. **Dark mode is free** — switching `data-theme` toggles all variable values; using variables makes it work automatically
3. **No dashboard chrome in pages** — the dashboard owns navigation
4. **Reusable components live in `assets/`** — extract shared CSS with `pharos asset create`
5. **Do not repeat FOUC-prevention or postMessage logic** across assets — it exists in the boilerplate; `assets/style.css` and `assets/glossary-tooltip.js` should be purely presentational/behavioural, not theme-detection
