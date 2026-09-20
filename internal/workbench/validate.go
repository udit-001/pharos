// Package workbench provides validation for sql-workbench browser events
// and dataset files. Error strings are written to be displayed verbatim —
// they are agent-facing copy (decision 5, LEARN-206).
//
// Three consumers: ingest handler, "workbench check", "workbench add".
package workbench

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// validEventTypes is the set of recognised bench event types (decision 3).
var validEventTypes = map[string]bool{
	"query":   true,
	"error":   true,
	"info":    true,
	"dataset": true,
}

// datasetIDRe restricts dataset id / filename stem to lowercase slug.
var datasetIDRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)

// ─── Event validation ───────────────────────────────────────────────

// WorkbenchEvent is the wire shape for a single bench event.
// The payload is kept as raw JSON so extra fields are stored verbatim.
type WorkbenchEvent struct {
	ID        string          `json:"id"`
	Namespace string          `json:"namespace"`
	Type      string          `json:"type"`
	Ts        string          `json:"ts"`
	Payload   json.RawMessage `json:"payload"`
}

// ValidateEvent checks that an event has the required shape for its type.
// Returns nil on success, or a human-readable error string suitable for
// display (verbatim contract).
func ValidateEvent(ev WorkbenchEvent) error {
	if ev.ID == "" {
		return fmt.Errorf("event: missing required field \"id\"")
	}
	if ev.Type == "" {
		return fmt.Errorf("event %q: missing required field \"type\"", ev.ID)
	}
	if !validEventTypes[ev.Type] {
		return fmt.Errorf("event %q: unknown type %q (valid: query, error, info, dataset)", ev.ID, ev.Type)
	}
	if ev.Ts == "" {
		return fmt.Errorf("event %q: missing required field \"ts\"", ev.ID)
	}

	// Parse payload as generic map to check required fields per type.
	var p map[string]json.RawMessage
	if ev.Payload != nil {
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			return fmt.Errorf("event %q: payload is not valid JSON: %s", ev.ID, err)
		}
	}

	switch ev.Type {
	case "query":
		return validateQueryPayload(ev.ID, p)
	case "error":
		return validateErrorPayload(ev.ID, p)
	case "info":
		// info events have no required payload fields.
	case "dataset":
		return validateDatasetEventPayload(ev.ID, p)
	}
	return nil
}

// validateQueryPayload checks that a query event has sql + ok.
func validateQueryPayload(id string, p map[string]json.RawMessage) error {
	if p == nil {
		return fmt.Errorf("event %q (query): payload is required", id)
	}
	if _, ok := p["sql"]; !ok {
		return fmt.Errorf("event %q (query): missing required field \"sql\" in payload", id)
	}
	if _, ok := p["ok"]; !ok {
		return fmt.Errorf("event %q (query): missing required field \"ok\" in payload", id)
	}

	// When ok is false, error is required.
	var okVal bool
	if raw, ok := p["ok"]; ok {
		if err := json.Unmarshal(raw, &okVal); err != nil {
			return fmt.Errorf("event %q (query): \"ok\" must be a boolean", id)
		}
	}
	if !okVal {
		if _, ok := p["error"]; !ok {
			return fmt.Errorf("event %q (query): ok is false but missing required field \"error\" in payload", id)
		}
	}
	return nil
}

// validateErrorPayload checks that an error event has an error field.
func validateErrorPayload(id string, p map[string]json.RawMessage) error {
	if p == nil {
		return fmt.Errorf("event %q (error): payload is required", id)
	}
	if _, ok := p["error"]; !ok {
		return fmt.Errorf("event %q (error): missing required field \"error\" in payload", id)
	}
	return nil
}

// validateDatasetEventPayload checks that a dataset event has dataset field.
func validateDatasetEventPayload(id string, p map[string]json.RawMessage) error {
	if p == nil {
		return fmt.Errorf("event %q (dataset): payload is required", id)
	}
	if raw, ok := p["dataset"]; !ok {
		return fmt.Errorf("event %q (dataset): missing required field \"dataset\" in payload", id)
	} else {
		var ds string
		if err := json.Unmarshal(raw, &ds); err != nil {
			return fmt.Errorf("event %q (dataset): \"dataset\" must be a string", id)
		}
		if ds == "" {
			return fmt.Errorf("event %q (dataset): \"dataset\" must not be empty", id)
		}
	}
	return nil
}

// ─── Dataset validation ─────────────────────────────────────────────

// ValidateDataset checks that a dataset id (filename stem) meets the
// naming contract: [a-z0-9]([a-z0-9\-]*[a-z0-9])?, min length 1.
func ValidateDataset(id string) error {
	if id == "" {
		return fmt.Errorf("dataset id must not be empty")
	}
	if !datasetIDRe.MatchString(id) {
		return fmt.Errorf("dataset id %q is invalid: must be lowercase alphanumeric with optional hyphens (e.g. \"my-data\", \"sample1\")", id)
	}
	return nil
}

// ValidateDatasetUpload checks that a dataset file is valid for install:
// non-empty, valid JSON, and under the 50 MB cap. The filename stem must
// pass ValidateDataset.
func ValidateDatasetUpload(id string, data []byte) error {
	if err := ValidateDataset(id); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("dataset %q: file is empty", id)
	}
	const maxBytes = 50 * 1024 * 1024
	if len(data) > maxBytes {
		return fmt.Errorf("dataset %q: file exceeds 50 MB limit (%d bytes)", id, len(data))
	}
	if !json.Valid(data) {
		// Find the line/col of the first parse error for a useful message.
		var syntax json.SyntaxError
		if json.Unmarshal(data, &syntax) != nil {
			return fmt.Errorf("dataset %q: file is not valid JSON: %s", id, truncateJSONErr(syntax.Error()))
		}
		return fmt.Errorf("dataset %q: file is not valid JSON", id)
	}
	return nil
}

// truncateJSONErr shortens verbose json.SyntaxError messages to the
// useful part (offset info), keeping the message scannable.
func truncateJSONErr(s string) string {
	if idx := strings.Index(s, "offset "); idx != -1 {
		return s[:idx+len("offset N")]
	}
	return s
}
