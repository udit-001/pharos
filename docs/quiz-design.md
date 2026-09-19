# Quiz UI Design — Pharos

## Design principle

**Khan Academy style, inline in the app.** Quizzes are a content type, not a separate mode. The quiz feels like the next paragraph of the lesson — just interactive.

## Layout overview

### Quiz Library (`/workspace/{name}/quizzes`)

Standard document page with heading, subtitle, and bordered list.

```
Quizzes                              ← h1 text-xl font-semibold text-slate-800 tracking-tight
1 quizzes                            ← p text-sm text-slate-400 mt-0.5 mb-5

In progress: Genetics foundations (0/1)  ← blue callout (when applicable)

Genetics foundations                  ← quiz title
1 questions · Core genetic factors in ASD   ← subtitle
                    badge (1/1 or 0/1 or amber partial)    ›
```

- Standard heading + subtitle pattern (matching workspace/record pages)
- `py-3 border-b border-slate-100 last:border-0` list items
- Score badge: emerald for perfect, amber for partial, slate for not started
- Chevron `›` on the right side

### Quiz Intro (`/workspace/{name}/quiz/{slug}`)

Document-style start page — no centering, no big icon.

```
Genetics foundations                   ← h1 text-xl font-semibold text-slate-800 tracking-tight
1 questions · Core genetic factors in ASD  ← p text-sm text-slate-400 mt-0.5 mb-5

[ Start quiz ]                         ← bg-slate-800 text-white rounded-lg

Previous attempts                      ← text-xs font-medium text-slate-400 uppercase tracking-wider
  29 Jun 2026                  1/1    Review
```

- Top-anchored, no flex centering
- No decorative big icon
- Standard heading treatment
- Button is `py-2.5 px-5` (not full-width)
- Previous attempts: same style as before (`bg-slate-50 border border-slate-200 rounded-lg`)

### Quiz Attempt — Before answering (`/workspace/{name}/quiz/{slug}/attempt/{id}`)

Immersive, minimal chrome.

```
(no heading — breadcrumbs handle context)

● ● ○ ○ ○                              ← w-3 h-3 rounded-full dots (current has ring-2 ring-slate-400)

Which gene has the strongest known association with
Autism Spectrum Disorder?              ← h3 text-lg font-medium text-slate-800 leading-relaxed mb-5

┌───────────────────────────────────────┐
│ (A)  CHD8                            │ ← option-btn: w-full text-left flex items-center
├───────────────────────────────────────┤   gap-3 p-3 rounded-lg border border-slate-200
│ (B)  FMR1                            │   hover:bg-slate-50 hover:border-slate-300
├───────────────────────────────────────┤
│ (C)  MECP2                           │
├───────────────────────────────────────┤
│ (D)  SHANK3                          │
└───────────────────────────────────────┘

                                       Quit →   ← text-xs text-slate-400, right-aligned below

                                       [ Check ]  ← right-aligned, disabled until selection
```

- **Dots**: `w-3 h-3 rounded-full` (not `w-5 h-1.5`), read-only during attempt
- **Current dot**: `bg-blue-700 ring-2 ring-slate-400`
- **Answered correct**: `bg-emerald-600`
- **Answered incorrect**: `bg-red-600`
- **Unanswered**: `bg-slate-200`
- **No card wrapper** — content sits directly on the page background
- **Question text**: `text-lg` (larger than before)
- **Option buttons**: same styling, full-width with letter circles
- **Check button**: `bg-slate-800 text-white py-2 px-5`, right-aligned via `flex justify-end`
- **Quit button**: below the question, right-aligned, `text-xs text-slate-400`

### Quiz Attempt — After answering (correct)

```
● ● ● ○ ○

Which gene has the strongest known association with
Autism Spectrum Disorder?

┌───────────────────────────────────────┐
│ ✓ (A)  CHD8             emerald bg   │ ← border-emerald-600 bg-emerald-50 + checkmark svg
├───────────────────────────────────────┤
│ (B)  FMR1              dim/strikethru│ ← border-slate-200 text-slate-400 line-through
├───────────────────────────────────────┤
│ (C)  MECP2             dim/strikethru│
├───────────────────────────────────────┤
│ (D)  SHANK3            dim/strikethru│
└───────────────────────────────────────┘

┌───────────────────────────────────────┐
│ ✓ Correct!                           │ ← flex items-center gap-2 p-3 rounded-lg
└───────────────────────────────────────┘   bg-emerald-50 border border-emerald-600 text-sm

2/3 correct so far                    [ Next → ]   ← next right-aligned, score left-aligned
```

- Correct option: `border-emerald-600 bg-emerald-50` with checkmark SVG
- Wrong options: `border-slate-200 text-slate-400 line-through`
- Feedback box: `flex items-center gap-2 p-3 rounded-lg bg-emerald-50 border border-emerald-600 text-sm`
- Score line: `text-xs text-slate-500` showing "X/Y correct so far"
- Next button: `bg-slate-800 text-white py-2 px-5`, right-aligned

### Quiz Attempt — Recall (self-grade flashcard)

```
● ● ● ● ○

What is the "double empathy problem" in autism
research?                              ← h3 text-lg

┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐ ← border-dashed border-slate-300 bg-slate-50
│ Both autistic and non-autistic      │   rounded-lg p-4
│ people struggle to read each        │
│ other's communication               │
└ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘

┌──────────────────────┐  ┌──────────────────────┐
│      Got it          │  │      Not yet         │ ← flex gap-3
└──────────────────────┘  └──────────────────────┘   flex-1 each
```

- Flashcard: same flip animation, but no `max-w-lg mx-auto` constraint
- Grade buttons: `flex-1` each, filling the width

### Quiz Review (`/workspace/{name}/quiz/{slug}/review/{id}`)

Score summary + navigable question walkthrough.

```
Genetics foundations                   ← h1 text-xl font-semibold text-slate-800 tracking-tight
1/1 correct                            ← p text-sm text-slate-400 mt-0.5 mb-2

████████████████████░░░░░░░░░░░░       ← h-1.5 bg-slate-200 + emerald fill

● ● ● ● ○                              ← clickable dots, w-3 h-3

Strongest ASD risk gene                 ← h4 text-lg font-medium

✓ Correct: CHD8                        ← border-emerald-600 bg-emerald-50
FMR1                                    ← border-slate-200 text-slate-400
MECP2                                   ← border-slate-200 text-slate-400
SHANK3                                  ← border-slate-200 text-slate-400

                                       [ Retake quiz ]  ← right-aligned
```

- Heading + subtitle with score
- Thin progress bar (`h-1.5 rounded-full`)
- Clickable dots for navigation (no prev/next buttons)
- Review answers styled same as attempt feedback
- Retake button right-aligned

## Implementation details

### Files changed

| File | Changes |
|------|---------|
| `internal/render/models.go` | Added `BestScore int`, `BestTotal int` to `QuizEntry` |
| `internal/render/views.go` | Added `BestScoreText()`, `quizBadgeClass()` helpers |
| `internal/render/views.templ` | Rewrote `quizLibrary`, `quiz`, `quizAttempt`, `quizReview` templates |
| `internal/render/quiz_scripts.go` | Rewrote attempt + review JS: new element IDs, no card wrapper, right-aligned buttons |
| `internal/server/mux.go` | Populate `BestScore`/`BestTotal` in quiz library handler |

### Key CSS classes used

- **Dots**: `w-3 h-3 rounded-full` with status colors
- **Option buttons**: `w-full text-left flex items-center gap-3 p-3 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-slate-300 transition-colors cursor-pointer text-sm text-slate-700`
- **Primary button**: `bg-slate-800 text-white text-sm font-medium py-2 px-5 rounded-lg hover:bg-slate-700 transition-colors`
- **Disabled button**: add `disabled:opacity-40 disabled:cursor-not-allowed`
- **Feedback correct**: `flex items-center gap-2 p-3 rounded-lg bg-emerald-50 border border-emerald-600 text-sm`
- **Feedback incorrect**: `flex items-center gap-2 p-3 rounded-lg bg-red-50 border border-red-600 text-sm`
- **Score badge**: `text-xs font-medium px-2 py-0.5 rounded` + colors

### JS rendering

- Attempt renders into `#question-area` (previously `#question-card`)
- Dots render into `#attempt-dots` (previously `#progress-dots`)
- Dots use `w-3 h-3` (previously `w-5 h-1.5`)
- Current dot gets `bg-blue-700 ring-2 ring-slate-400`
- Review drops prev/next nav — dots handle navigation
- Attempt drops `#progress-label` (no more text progress)
