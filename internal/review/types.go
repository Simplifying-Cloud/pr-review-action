package review

type Output struct {
	Summary  string    `json:"summary"`
	Verdict  string    `json:"verdict"` // approve, request_changes, comment
	Comments []Comment `json:"comments"`
}

type Comment struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	EndLine  int    `json:"end_line,omitempty"`
	Severity string `json:"severity"` // critical, warning, suggestion, nitpick
	Message  string `json:"message"`
}

func (c *Comment) SeverityEmoji() string {
	switch c.Severity {
	case "critical":
		return "🔴"
	case "warning":
		return "🟡"
	case "suggestion":
		return "🔵"
	case "nitpick":
		return "⚪"
	default:
		return "💬"
	}
}
