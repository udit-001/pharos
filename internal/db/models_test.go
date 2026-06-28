package db

import "testing"

func TestChoiceConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     ChoiceConfig
		wantErr bool
	}{
		{"valid", ChoiceConfig{Options: []string{"a", "b"}, Key: 0}, false},
		{"last key", ChoiceConfig{Options: []string{"a", "b"}, Key: 1}, false},
		{"one option", ChoiceConfig{Options: []string{"a"}, Key: 0}, true},
		{"key out of range", ChoiceConfig{Options: []string{"a", "b"}, Key: 2}, true},
		{"negative key", ChoiceConfig{Options: []string{"a", "b"}, Key: -1}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if c.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestRecallConfigValidate(t *testing.T) {
	if err := (RecallConfig{RevealText: "the answer"}).Validate(); err != nil {
		t.Errorf("expected no error for non-empty reveal, got %v", err)
	}
	if err := (RecallConfig{RevealText: ""}).Validate(); err == nil {
		t.Error("expected error for empty reveal text, got nil")
	}
}

func TestQuestionParseConfig(t *testing.T) {
	// Choice mode parses into ChoiceConfig.
	q := Question{Mode: "choice", Config: `{"options":["a","b"],"key":1}`}
	cfg, err := q.ParseConfig()
	if err != nil {
		t.Fatalf("choice ParseConfig: %v", err)
	}
	if _, ok := cfg.(ChoiceConfig); !ok {
		t.Errorf("choice config type = %T, want ChoiceConfig", cfg)
	}

	// Recall mode parses into RecallConfig.
	q = Question{Mode: "recall", Config: `{"reveal_text":"x"}`}
	cfg, err = q.ParseConfig()
	if err != nil {
		t.Fatalf("recall ParseConfig: %v", err)
	}
	if _, ok := cfg.(RecallConfig); !ok {
		t.Errorf("recall config type = %T, want RecallConfig", cfg)
	}

	// Unknown mode errors.
	if _, err := (Question{Mode: "bogus", Config: "{}"}).ParseConfig(); err == nil {
		t.Error("expected error for unknown mode, got nil")
	}
}
