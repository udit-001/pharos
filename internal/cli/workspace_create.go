package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new learning workspace",
	Long: `Create a new learning workspace.

The workspace is a directory under your data directory's workspaces/ containing:
  MISSION.md          — Why you're learning this topic
  RESOURCES.md        — Curated sources and communities
  NOTES.md            — Preferences and working notes
  lessons/            — Self-contained lesson HTML files
  learning-records/   — ADR-style learning records
  reference/          — Cheat sheets and reference docs
  assets/             — Reusable components (stylesheets, quizzes)

Use '--dir <path>' to place the workspace elsewhere.

Examples:
  pharos workspace create "SQL for Research"
  pharos workspace create "Jump Start a Car" --dir ./my-workspace`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		displayName := args[0]
		slug := db.Slugify(displayName)

		// Determine workspace path
		customDir, _ := cmd.Flags().GetString("dir")

		var wsPath string
		if customDir != "" {
			wsPath = customDir
		} else {
			wsPath = filepath.Join(defaultWorkspacesDir(), slug)
		}

		// Create workspace directory
		if err := os.MkdirAll(wsPath, 0755); err != nil {
			return fmt.Errorf("create workspace directory: %w", err)
		}

		// Create workspace subdirectories
		for _, d := range []string{"lessons", "learning-records", "reference", "assets"} {
			if err := os.MkdirAll(filepath.Join(wsPath, d), 0755); err != nil {
				return fmt.Errorf("create %s directory: %w", d, err)
			}
		}

		// Create default assets/style.css if not present
		cssFile := filepath.Join(wsPath, "assets", "style.css")
		if _, err := os.Stat(cssFile); os.IsNotExist(err) {
			if err := os.WriteFile(cssFile, []byte(defaultStyleCSS), 0644); err != nil {
				return fmt.Errorf("write assets/style.css: %w", err)
			}
		}

		// Create default assets/glossary-tooltip.js if not present
		glossaryJSFile := filepath.Join(wsPath, "assets", "glossary-tooltip.js")
		if _, err := os.Stat(glossaryJSFile); os.IsNotExist(err) {
			if err := os.WriteFile(glossaryJSFile, []byte(defaultGlossaryTooltipJS), 0644); err != nil {
				return fmt.Errorf("write assets/glossary-tooltip.js: %w", err)
			}
		}

		// Create MISSION.md template
		missionFile := filepath.Join(wsPath, "MISSION.md")
		if _, err := os.Stat(missionFile); os.IsNotExist(err) {
			missionContent := fmt.Sprintf(`# Mission: %s

## Why
{1-3 sentences. What changes in your life or work when you have this skill?}

## Success looks like
- {Specific, observable outcome}
- {Another specific outcome}

## Constraints
- {Time, budget, prior commitments}

## Out of scope
- {Adjacent topics you do not want to chase right now}
`, displayName)
			if err := os.WriteFile(missionFile, []byte(missionContent), 0644); err != nil {
				return fmt.Errorf("write MISSION.md: %w", err)
			}
		}

		// Create RESOURCES.md template
		resourcesFile := filepath.Join(wsPath, "RESOURCES.md")
		if _, err := os.Stat(resourcesFile); os.IsNotExist(err) {
			resourcesContent := fmt.Sprintf(`# %s Resources

## Knowledge

- [Resource title](https://example.com)
  What it covers and when to reach for it.

## Wisdom (Communities)

- [Community name](https://example.com)
  What kind of interaction to expect here.

## Gaps
- {Areas where no good resource exists yet}
`, displayName)
			if err := os.WriteFile(resourcesFile, []byte(resourcesContent), 0644); err != nil {
				return fmt.Errorf("write RESOURCES.md: %w", err)
			}
		}

		// Create NOTES.md
		notesFile := filepath.Join(wsPath, "NOTES.md")
		if _, err := os.Stat(notesFile); os.IsNotExist(err) {
			if err := os.WriteFile(notesFile, []byte("# Notes\n\nPreferences and working notes for this workspace.\n"), 0644); err != nil {
				return fmt.Errorf("write NOTES.md: %w", err)
			}
		}

		// Add to database
		topic, _ := cmd.Flags().GetString("topic")
		if topic == "" {
			topic = displayName
		}
		ws := db.Workspace{
			Name:  slug,
			Topic: topic,
			Path:  wsPath,
		}
		created, err := s.AddWorkspace(ws)
		if err != nil {
			return formatError("failed to create workspace", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Created workspace: %s\n", created.DisplayName())
		fmt.Printf("    Path: %s\n", wsPath)
		// Auto-set as current workspace
		_ = s.SetCurrentWorkspace(slug)

		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    cd " + wsPath)
		fmt.Println("    pharos lesson create \"Your first lesson\"")
		fmt.Println("    pharos record add \"What you learned\"")
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCreateCmd.Flags().String("dir", "", "Create workspace at a custom path")
	workspaceCreateCmd.Flags().String("topic", "", "Friendly display title for the workspace (default: the name you passed)")
}

// defaultStyleCSS is the initial assets/style.css seeded into every new
// workspace. Matches the Nord-inspired theme from LESSON-THEME.md.
const defaultStyleCSS = `:root {
  --slate-900: #2e3440;
  --slate-800: #3b4252;
  --slate-700: #4c566a;
  --slate-500: #6b7689;
  --slate-400: #8891a0;
  --slate-200: #e5e9f0;
  --slate-100: #eceff4;
  --slate-50:  #f8fafc;
  --white:     #ffffff;
  --blue-700:  #5e81ac;
  --emerald-600: #a3be8c;
  --emerald-100: #e6f0e6;
  --red-600:   #bf616a;
  --red-100:   #fce4e4;
  --amber-600: #d08770;
}

[data-theme="dark"] {
  --slate-900: #eceff4;
  --slate-800: #d8dee9;
  --slate-700: #aebbcf;
  --slate-500: #94adcb;
  --slate-400: #81a1c1;
  --slate-200: #434c5e;
  --slate-100: #353b4a;
  --slate-50:  #2e3440;
  --white:     #3b4252;
  --blue-700:  #81a1c1;
  --emerald-600: #a3be8c;
  --emerald-100: #2e3440;
  --red-600:   #bf616a;
  --red-100:   #4c566a;
  --amber-600: #d08770;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: 'Inter', ui-sans-serif, system-ui, sans-serif;
  background: var(--white);
  color: var(--slate-700);
  line-height: 1.75;
  font-size: 0.9375rem;
}

.container {
  max-width: 56rem;
  margin: 0 auto;
  padding: 2rem;
}

h1 { font-size: 1.375rem; font-weight: 700; color: var(--slate-800); margin-top: 1.5rem; margin-bottom: 0.5rem; letter-spacing: -0.01em; }
h2 { font-size: 1.125rem; font-weight: 600; color: var(--slate-800); margin-top: 1.25rem; margin-bottom: 0.4rem; }
h3 { font-size: 1rem;    font-weight: 600; color: var(--slate-700); margin-top: 1rem;    margin-bottom: 0.3rem; }

p { margin: 0.6rem 0; }

a { color: var(--blue-700); text-decoration: underline; text-underline-offset: 2px; }
a:hover { color: var(--slate-800); }

ul, ol { margin: 0.5rem 0; padding-left: 1.5rem; }
ul { list-style-type: disc; }
ol { list-style-type: decimal; }
li { margin: 0.2rem 0; }

blockquote {
  border-left: 3px solid var(--slate-200);
  padding-left: 1rem;
  margin: 0.75rem 0;
  color: var(--slate-500);
  font-style: italic;
}

code {
  background: var(--slate-100);
  padding: 0.15em 0.35em;
  border-radius: 4px;
  font-size: 0.875em;
  font-family: ui-monospace, 'SF Mono', 'Cascadia Code', monospace;
}

pre {
  background: var(--slate-100);
  border: 1px solid var(--slate-200);
  border-radius: 8px;
  padding: 0.9rem 1rem;
  margin: 0.75rem 0;
  overflow-x: auto;
  font-size: 0.85rem;
  line-height: 1.6;
}
pre code { background: none; padding: 0; border-radius: 0; font-size: inherit; }

hr { border: none; border-top: 1px solid var(--slate-200); margin: 1.25rem 0; }

strong { font-weight: 600; color: var(--slate-800); }

table { width: 100%; border-collapse: collapse; margin: 0.75rem 0; font-size: 0.875rem; }
th { text-align: left; font-weight: 600; padding: 0.4rem 0.75rem; border-bottom: 2px solid var(--slate-200); color: var(--slate-800); }
td { padding: 0.35rem 0.75rem; border-bottom: 1px solid var(--slate-100); }

img { max-width: 100%; border-radius: 6px; margin: 0.75rem 0; }

.correct { background: var(--emerald-100); border: 1px solid var(--emerald-600); color: var(--emerald-600); padding: 0.5rem 1rem; border-radius: 6px; }
.incorrect { background: var(--red-100); border: 1px solid var(--red-600); color: var(--red-600); padding: 0.5rem 1rem; border-radius: 6px; }

button, .btn {
  font-family: 'Inter', ui-sans-serif, system-ui, sans-serif;
  background: var(--blue-700);
  color: #fff;
  border: none;
  border-radius: 8px;
  padding: 0.5rem 1rem;
  cursor: pointer;
  font-size: 0.875rem;
}
button:hover, .btn:hover { opacity: 0.9; }

/* Glossary tooltips */
.glossary-term { cursor: help; border-bottom: 1px dashed currentColor; position: relative; }
.glossary-tooltip { position: fixed; background: var(--slate-900); color: var(--slate-50); padding: 10px 14px; border-radius: 8px; font-size: 13px; line-height: 1.5; max-width: 320px; white-space: normal; box-shadow: 0 8px 24px rgba(0,0,0,0.2); pointer-events: none; opacity: 0; transition: opacity 0.15s ease; z-index: 9999; }
.glossary-tooltip.visible { opacity: 1; }
.glossary-tooltip::after { content: ''; position: absolute; left: 50%; margin-left: -6px; border: 6px solid transparent; }
/* Arrow points down (below tooltip) when tooltip is above the term */
.glossary-tooltip::after { top: 100%; border-top-color: var(--slate-900); }
/* Arrow points up (above tooltip) when tooltip is below the term — toggled by JS */
.glossary-tooltip.tooltip-below::after { top: auto; bottom: 100%; border-top-color: transparent; border-bottom-color: var(--slate-900); }
`

// defaultGlossaryTooltipJS is the runtime script that fetches glossary
// terms from the API and shows tooltips on hover over .glossary-term spans.
// Seeded into assets/glossary-tooltip.js for every new workspace.
const defaultGlossaryTooltipJS = `(function() {
  var m = window.location.pathname.match(/\/api\/lesson-html\/([^/]+)\//);
  if (!m) return;
  var wsName = m[1];
  var tip = document.createElement('div');
  tip.className = 'glossary-tooltip';
  document.body.appendChild(tip);
  var els = document.querySelectorAll('.glossary-term');
  if (!els.length) return;
  var defs = null;
  fetch('/api/workspaces/name/' + encodeURIComponent(wsName) + '/glossary-terms')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      defs = {};
      data.forEach(function(t) { defs[t.term] = t.definition; });
    })
    .catch(function() {});
  els.forEach(function(el) {
    el.addEventListener('mouseenter', function(e) {
      if (!defs) return;
      var term = el.getAttribute('data-term') || el.textContent.trim();
      var def = defs[term];
      if (!def) return;
      tip.textContent = def;
      var r = tip.getBoundingClientRect();
      var x = Math.min(e.clientX - r.width / 2, window.innerWidth - r.width - 10);
      var yAbove = e.clientY - r.height - 12;
      var yBelow = e.clientY + 12;
      var useAbove = yAbove >= 0;
      var y = useAbove ? yAbove : yBelow;
      tip.classList.toggle('tooltip-below', !useAbove);
      tip.style.left = Math.max(10, x) + 'px';
      tip.style.top = y + 'px';
      tip.classList.add('visible');
    });
    el.addEventListener('mouseleave', function() {
      tip.classList.remove('visible');
    });
  });
})();`
