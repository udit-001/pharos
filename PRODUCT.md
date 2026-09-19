# Product

## Register

product

## Users

Three overlapping users, all of whom delegate the scaffolding of learning to an
AI agent and then study what it produces:

- **The self-learner** — picking up a topic who wants structured lessons,
  retained records, and self-tests instead of passive consumption. They arrive
  from an agent conversation ("teach me about X") and return to workspaces to
  keep studying.
- **The developer building skills** — systematically learning a new language,
  framework, or domain, using an agent to scaffold lessons and quizzes that
  stick.
- **The continuous knowledge worker** — a professional who learns constantly
  and wants a calm, bounded place to retain what they study rather than lose it
  to a browser-tab graveyard.

Context of use: focused study sessions, often returning, frequently in the
evening or low-light (which is why dark mode is a real mode, not a novelty).
The job to be done: *turn an agent conversation into structured, retainable
knowledge — then study it without distraction.*

## Product Purpose

Pharos is a CLI tool with a read-only web dashboard for AI-guided learning
workspaces. An AI agent (via the `teach` skill) creates a workspace around a
topic and scaffolds its contents — lessons, learning records, references, a
glossary, and quizzes — and the dashboard is where the human studies them.

The dashboard owns navigation and the reading surface; lesson and reference
content renders inside iframes that theme-sync with the chrome. Everything is
local-first: all data stored locally, no telemetry, vendored assets so lessons
render offline.

Success looks like three reinforcing outcomes:
1. **Sustained retention** — users keep returning to workspaces and actually
   retain what they learn.
2. **Agent-orchestrated study** — the agent does the heavy lifting
   (scaffolding, teaching, quizzing); the human just shows up and studies.
3. **Calm focus** — a distraction-free reading environment that respects
   attention: the anti-dopamine study tool.

## Brand Personality

**Calm, focused, expert.** Quiet authority over loud enthusiasm. The product
feels like a well-made reference book or a reading room, not an app vying for
attention. Confidence comes from legibility and structure, not ornament.

Voice: direct, low-volume, no exclamation. Emotional goals: confidence (not
anxiety), depth (not breadth), calm (not urgency). The surface never competes
with the content.

## Anti-references

No named anti-references specified by the owner. The confirmed personality
(calm, focused, expert) implies, by inversion, what the surface should not
become — listed as guardrails rather than named bad-example sites:

- **Not dopamine-driven.** No streaks, XP, confetti, leaderboards, or reward
  theater. Retention comes from structure, not pressure.
- **Not feature-crammed.** No side panels, command bars, and toggles fighting
  for the same pixel. One workspace, one focus.
- **Not a dense terminal.** Readability over density; the surface is a place
  to read in, not a control panel to operate.
- **Not the 2026 AI-editorial default.** No cream/beige warm-tinted bg with
  oversized serif headings. The Nord palette stays cool and restrained.

## Design Principles

1. **Agent does the lifting, human does the learning.** The tool orchestrates
   teaching and scaffolding so the learner only has to show up and study.
2. **Calm over dopamine.** Respect attention. No nagging, no streaks, no reward
   theater — structure is what makes knowledge stick.
3. **One workspace, one focus.** Each workspace is a bounded study context;
   the chrome never competes with the content it frames.
4. **Dark mode is a first-class study mode.** Evening reading is a real use
   case, not a novelty — theming is built into the token system, not bolted on.
5. **Empty states teach the next action.** Every empty surface shows the exact
   agent command that fills it; the tool is always one prompt away from useful.

## Accessibility & Inclusion

**WCAG AA**, matching what is already implemented in the codebase:

- Body and placeholder text ≥ 4.5:1 contrast; large/bold text ≥ 3:1. Search
  type badges are explicitly tuned for AA in both light and dark (bright
  aurora hues are lifted toward snow in dark mode to clear 4.5:1).
- `prefers-reduced-motion: reduce` fallbacks on every animated surface
  (sidebar, command palette, theme toggle, glossary tooltips).
- Keyboard navigation: command palette (Cmd+K, arrows, Enter, Esc), sidebar
  collapse/section toggle, focus-visible rings on interactive controls.

No stricter target (AAA) or additional cognitive/dyslexia measures specified
beyond the above; AA is the committed bar.
