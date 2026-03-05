package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseOutput extracts a ReviewOutput from the LLM response.
// Handles: raw JSON, markdown-fenced JSON, JSON embedded in text.
func ParseOutput(raw string) (*Output, error) {
	raw = strings.TrimSpace(raw)

	// Try 1: direct unmarshal
	var out Output
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return &out, nil
	}

	// Try 2: extract from markdown code fence
	if idx := strings.Index(raw, "```json"); idx != -1 {
		start := idx + len("```json")
		end := strings.Index(raw[start:], "```")
		if end != -1 {
			extracted := strings.TrimSpace(raw[start : start+end])
			if err := json.Unmarshal([]byte(extracted), &out); err == nil {
				return &out, nil
			}
		}
	}
	if idx := strings.Index(raw, "```"); idx != -1 {
		start := idx + len("```")
		// skip optional language tag on same line
		if nl := strings.Index(raw[start:], "\n"); nl != -1 {
			start += nl + 1
		}
		end := strings.Index(raw[start:], "```")
		if end != -1 {
			extracted := strings.TrimSpace(raw[start : start+end])
			if err := json.Unmarshal([]byte(extracted), &out); err == nil {
				return &out, nil
			}
		}
	}

	// Try 3: find first { and last } and try to parse
	first := strings.Index(raw, "{")
	last := strings.LastIndex(raw, "}")
	if first != -1 && last > first {
		extracted := raw[first : last+1]
		if err := json.Unmarshal([]byte(extracted), &out); err == nil {
			return &out, nil
		}
	}

	// Fallback: return raw response as summary
	return &Output{
		Summary: fmt.Sprintf("Could not parse structured review. Raw response:\n\n%s", raw),
		Verdict: "comment",
	}, nil
}
