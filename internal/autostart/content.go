package autostart

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// xmlEscape escapes text for embedding in the plist XML we write. The plist
// is our own output; only & < > can appear in a filesystem path and need
// escaping in XML text content.
func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// plistContent renders the launchd user-agent plist for the macOS startup
// entry. ProgramArguments lists the daemon command line exactly as pinned at
// enable time (exe + `start --background --no-open --daemon --port N`).
// RunAtLoad=true launches at login; KeepAlive=false deliberately does not
// respawn past a manual `pharos stop` — `pharos start`'s pidfile already
// makes the flow idempotent.
func plistContent(exe string, daemonArgs []string) []byte {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>com.udit001.pharos</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, s := range append([]string{exe}, daemonArgs...) {
		b.WriteString("\t\t<string>" + xmlEscape(s) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("\t<key>KeepAlive</key>\n\t<false/>\n")
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String())
}

var (
	plistProgArgsRe = regexp.MustCompile(`(?s)<key>ProgramArguments</key>\s*<array>(.*?)</array>`)
	plistStringRe   = regexp.MustCompile(`<string>(.*?)</string>`)
)

// parsePlistArgs extracts ProgramArguments from a plist's XML (our own
// format — best-effort on hand-edited files; enable is the fix either way).
func parsePlistArgs(xml string) []string {
	m := plistProgArgsRe.FindStringSubmatch(xml)
	if m == nil {
		return nil
	}
	var out []string
	for _, sm := range plistStringRe.FindAllStringSubmatch(m[1], -1) {
		out = append(out, html.UnescapeString(sm[1]))
	}
	return out
}

// desktopContent renders the XDG autostart .desktop file. Terminal=false —
// the entry must never open a terminal window at login.
func desktopContent(exe string, daemonArgs []string) []byte {
	execLine := strconv.Quote(exe) + " " + strings.Join(daemonArgs, " ")
	text := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Pharos\n" +
		"Comment=Start the Pharos dashboard at login\n" +
		"Exec=" + execLine + "\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
	return []byte(text)
}

var desktopExecRe = regexp.MustCompile(`(?m)^Exec=(.*)$`)

// parseDesktopArgs extracts the command-line tokens of a .desktop Exec line,
// honoring double-quoted arguments. (Exec lines may start with env
// assignments — ours don't.)
func parseDesktopArgs(text string) []string {
	m := desktopExecRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	return splitQuoted(m[1])
}

// splitQuoted splits on whitespace, keeping double-quoted groups intact.
// A backslash-escaped quote (as strconv.Quote emits inside a quoted path)
// is treated as a literal quote character, so what we write round-trips.
func splitQuoted(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && i+1 < len(rs) && rs[i+1] == '"':
			cur.WriteRune('"')
			i++
		case r == '"':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// psQuote wraps s in a PowerShell single-quoted string literal, escaping
// embedded quotes per PowerShell's ” doubling rule.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// lnkScript builds the PowerShell one-liner that writes the Windows Startup
// shortcut through WScript.Shell — the seamiest way to produce a .lnk from
// Go without a compile-time COM dependency. The command line pins the port,
// matching every other platform's entry.
func lnkScript(exe, lnkPath string, port int) string {
	return "$s = New-Object -ComObject WScript.Shell; " +
		"$c = $s.CreateShortcut(" + psQuote(lnkPath) + "); " +
		"$c.TargetPath = " + psQuote(exe) + "; " +
		"$c.Arguments = " + psQuote(strings.Join(args(port), " ")) + "; " +
		"$c.Save()"
}
