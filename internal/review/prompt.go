package review

import (
	"fmt"
	"strings"

	"github.com/Simplifying-Cloud/pr-review-action/internal/github"
	"github.com/Simplifying-Cloud/pr-review-action/internal/llm"
)

const systemPrompt = `You are a senior code reviewer. Review the pull request diff and provide feedback as a JSON object.

Your output MUST be a single JSON object with this exact structure:
{
  "summary": "2-3 sentence overall assessment",
  "verdict": "approve" | "request_changes" | "comment",
  "comments": [
    {
      "path": "relative/file/path.go",
      "line": 42,
      "severity": "critical" | "warning" | "suggestion" | "nitpick",
      "message": "Explanation of the issue and suggested fix"
    }
  ]
}

Rules:
- "line" must refer to line numbers in the NEW version of the file
- Only comment on lines that appear in the diff (added or modified lines)
- Use "critical" for security issues, data loss risks, or correctness bugs
- Use "warning" for potential bugs, error handling gaps, or performance issues
- Use "suggestion" for improvements to readability, structure, or maintainability
- Use "nitpick" for style, naming, or minor formatting issues
- Set verdict to "approve" if there are no critical or warning issues
- Set verdict to "request_changes" if there are critical issues
- Set verdict to "comment" otherwise
- If there are no issues at all, return an empty comments array with verdict "approve"
- For approvals, write a brief summary of what the PR does and why it looks good (1-2 sentences)
- Output ONLY the JSON object, no other text`

func BuildPrompt(pr *github.PullRequest, diff string, files []github.PRFile, reviewFocus, extraPrompt string) []llm.ChatMessage {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Pull Request: %s\n\n", pr.Title))

	if pr.Body != "" {
		sb.WriteString("### Description\n")
		sb.WriteString(pr.Body)
		sb.WriteString("\n\n")
	}

	sb.WriteString("### Changed Files\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", f.Filename, f.Status))
	}
	sb.WriteString("\n")

	sb.WriteString("### Diff\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n")

	sys := systemPrompt
	if reviewFocus != "" {
		sys += "\n\nFocus areas:\n" + reviewFocus
	}
	if extraPrompt != "" {
		sys += "\n\nAdditional instructions:\n" + extraPrompt
	}

	return []llm.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: sb.String()},
	}
}
