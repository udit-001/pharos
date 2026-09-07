package web

import (
	"regexp"
	"strings"
	"testing"
)

func TestJSBundleURLVersioned(t *testing.T) {
	for _, name := range []string{"presence.js", "pharos-theme.js", "pharos-toc.js"} {
		u := JSBundleURL(name)
		prefix := "/js/" + name + "?v="
		if !strings.HasPrefix(u, prefix) {
			t.Errorf("%s: url = %q, want prefix %q", name, u, prefix)
			continue
		}
		if v := strings.TrimPrefix(u, prefix); !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(v) {
			t.Errorf("%s: version %q is not 12 lowercase hex chars", name, v)
		}
		if again := JSBundleURL(name); again != u {
			t.Errorf("%s: JSBundleURL not stable: %q then %q", name, u, again)
		}
	}
}

func TestJSBundleURLUnknown(t *testing.T) {
	if u := JSBundleURL("not-a-bundle.js"); u != "" {
		t.Errorf("unknown bundle: url = %q, want empty", u)
	}
}

func TestCSSURLVersioned(t *testing.T) {
	u := CSSURL()
	if !strings.HasPrefix(u, "/css/app.css?v=") {
		t.Errorf("css url = %q, want /css/app.css?v=…", u)
	}
	if v := strings.TrimPrefix(u, "/css/app.css?v="); !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(v) {
		t.Errorf("css version %q is not 12 hex chars", v)
	}
	if CSSURL() != u {
		t.Error("CSSURL not stable")
	}
}
