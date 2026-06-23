# Lesson Theme — Nord-Inspired Design System

Lessons render inside an iframe within the Pharos dashboard. They must match the dashboard's Nord-inspired palette so they feel integral to the app, not embedded.

The dashboard controls theme via `data-theme` attribute on `<html>` — light or dark. Lessons sync by reading `localStorage` on load and listening for `postMessage` theme events at runtime.

---

## Architecture

| Concern | Mechanism |
|---|---|
| Palette | CSS custom properties on `:root` / `[data-theme="dark"]` |
| FOUC prevention | Blocking `<script>` in `<head>` reads `localStorage('pharos_theme')`, falls back to `prefers-color-scheme`, sets `data-theme` |
| Runtime theme sync | `postMessage` listener — dashboard sends `{type:'theme', theme:'dark'|'light'}` to iframes on toggle |
| Shared styles | `assets/style.css` (variables, typography, layout, component classes) + `assets/quiz.css` (quiz-specific classes) |
| Quiz interactivity | Inline `<script>` before `</body>` — binds to `.q` elements |

---

## Nord Palette (CSS Variables)

These are the shared variables every lesson must use — never hardcoded color values.

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

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Lesson Title</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="assets/style.css">
<link rel="stylesheet" href="assets/quiz.css">
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
<script>window.addEventListener('message',function(e){if(e.data&&e.data.type==='theme')document.documentElement.dataset.theme=e.data.theme})</script>
</body>
</html>
```

Key rules:
- **No `data-theme` on `<html>`** — the blocking script sets it dynamically
- **Three scripts in order**: (1) FOUC prevention in `<head>`, (2) quiz logic before `</body>`, (3) postMessage listener before `</body>`
- **CSS links are root-relative** — no `../`, the iframe serves from `/api/lesson-html/<workspace>/<file>`

---

## Component Patterns (not CSS — design free)

These are the functional building blocks. The teach skill should create appropriate CSS for each.

### Quiz

An inline knowledge check. Structure: a container `.q` with `data-answer` attribute, multiple `<button>` options, and an `.fb` feedback element. On click: disable all buttons, compare clicked text to `data-answer`, mark correct/incorrect, show feedback text.

Classes: `.correct` (green), `.incorrect` (red) — applied to buttons after answer.

### Callout

A key takeaway or insight. Visually distinct from body paragraphs — uses accent border/background to draw attention.

### Source box

A recommendation for further reading. Subtle background, distinguishes it from callout.

### Ask prompt

A reminder at lesson end that the user can ask followup questions. Dashed border to feel provisional / conversational.

### Tables

Standard data tables with bordered rows, left-aligned headers, muted dividers.

### Buttons

Used for quiz options and any clickable action. Rounded, filled, hover feedback.

---

## Layout

- `.container` — centered, max-width matches the dashboard's reading column (~56rem), padded
- Lessons are self-contained HTML — no prev/next nav, the dashboard sidebar handles sequencing
- Content is single-column, stacked vertically

---

## Contextual Links (Iframe Escape)

Links that navigate outside the lesson (to any dashboard page) must use `target="_top"` with an absolute route. See [references/pharos-cli.md](references/pharos-cli.md) for the complete route table — never guess a URL pattern. Relative links like `../lesson/0002.html` load inside the iframe and lose the dashboard chrome.

---

## Principles

1. **Everything uses CSS variables**, never hardcoded hex values
2. **Dark mode is free** — switching `data-theme` toggles all variable values; using variables makes it work automatically
3. **No dashboard chrome in lessons** — the dashboard owns navigation
4. **Reusable components live in `assets/`** — extract shared CSS with `pharos asset create`
5. **Do not repeat FOUC-prevention or postMessage logic** across assets — it exists in the boilerplate; `assets/style.css` and `assets/quiz.css` should be purely presentational
