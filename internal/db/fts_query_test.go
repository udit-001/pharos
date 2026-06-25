package db

import "testing"

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"  ", ""},
		{"neuro", "neuro*"},
		{"neuro plasticity", "neuro* plasticity*"},
		{"neuro*", "neuro*"},
		{"neuro**", "neuro*"},
		{"  neuro  plasticity  ", "neuro* plasticity*"},
		{"***", ""},
		{"join", "join*"},
		{"SQL basics", "SQL* basics*"},
		{"running", "running*"},
	}
	for _, tt := range tests {
		got := buildFTSQuery(tt.in)
		if got != tt.want {
			t.Errorf("buildFTSQuery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
