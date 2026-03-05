package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Simplifying-Cloud/pr-review-action/internal/config"
	"github.com/Simplifying-Cloud/pr-review-action/internal/github"
	"github.com/Simplifying-Cloud/pr-review-action/internal/llm"
	"github.com/Simplifying-Cloud/pr-review-action/internal/review"
)

func main() {
	log.SetFlags(0)

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	log.Printf("Reviewing PR #%d in %s/%s using model %s", cfg.PRNumber, cfg.RepoOwner, cfg.RepoName, cfg.LLMModel)

	// 2. Initialize clients
	gh := github.NewClient(cfg.GitHubToken, cfg.APIURL)
	ai := llm.NewClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)

	// 3. Fetch PR data
	pr, err := gh.GetPR(cfg.RepoOwner, cfg.RepoName, cfg.PRNumber)
	if err != nil {
		log.Fatalf("fetching PR: %v", err)
	}

	diff, err := gh.GetPRDiff(cfg.RepoOwner, cfg.RepoName, cfg.PRNumber)
	if err != nil {
		log.Fatalf("fetching diff: %v", err)
	}

	if strings.TrimSpace(diff) == "" {
		log.Println("No diff found, skipping review")
		return
	}

	files, err := gh.GetPRFiles(cfg.RepoOwner, cfg.RepoName, cfg.PRNumber)
	if err != nil {
		log.Fatalf("fetching files: %v", err)
	}

	// Truncate large diffs
	const maxDiffSize = 100_000
	truncated := false
	if len(diff) > maxDiffSize {
		diff = diff[:maxDiffSize]
		truncated = true
		log.Printf("Warning: diff truncated to %d bytes", maxDiffSize)
	}

	log.Printf("PR: %s (%d files changed, %d bytes diff)", pr.Title, len(files), len(diff))

	// 4. Build prompt
	messages := review.BuildPrompt(pr, diff, files, cfg.ReviewFocus, cfg.ExtraPrompt)

	// 5. Call LLM
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("Calling LLM for review...")
	response, err := ai.Complete(ctx, messages, cfg.MaxTokens)
	if err != nil {
		log.Fatalf("LLM error: %v", err)
	}

	// 6. Parse response
	output, err := review.ParseOutput(response)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}
	log.Printf("Review: verdict=%s, %d comments", output.Verdict, len(output.Comments))

	// 7. Filter comments to diff-visible lines
	diffFiles := github.ParseDiffFiles(diff)
	var validComments []github.ReviewComment
	var skippedComments []review.Comment

	for _, c := range output.Comments {
		if github.IsCommentable(diffFiles, c.Path, c.Line) {
			body := fmt.Sprintf("**Argus** %s **%s**: %s", c.SeverityEmoji(), c.Severity, c.Message)
			validComments = append(validComments, github.ReviewComment{
				Path: c.Path,
				Line: c.Line,
				Side: "RIGHT",
				Body: body,
			})
		} else {
			skippedComments = append(skippedComments, c)
		}
	}

	// 8. Build summary
	summary := formatSummary(output, skippedComments, truncated, cfg.LLMModel, cfg.ReviewFocus, len(files), len(diff))

	// 9. Map verdict to GitHub review event
	event := "COMMENT"
	switch output.Verdict {
	case "approve":
		event = "APPROVE"
	case "request_changes":
		event = "REQUEST_CHANGES"
	}

	// 10. Submit review
	submission := &github.ReviewSubmission{
		CommitID: pr.SHA(),
		Body:     summary,
		Event:    event,
		Comments: validComments,
	}

	if err := gh.SubmitReview(cfg.RepoOwner, cfg.RepoName, cfg.PRNumber, submission); err != nil {
		log.Fatalf("submitting review: %v", err)
	}

	log.Printf("Review submitted: %s with %d inline comments", event, len(validComments))
}

func formatSummary(output *review.Output, skipped []review.Comment, truncated bool, model, reviewFocus string, fileCount, diffSize int) string {
	var sb strings.Builder

	sb.WriteString("## 👁️ Argus Review\n\n")

	switch output.Verdict {
	case "approve":
		sb.WriteString("✅ **Approved** — all clear.\n\n")
	case "request_changes":
		sb.WriteString("🔴 **Changes Requested**\n\n")
	default:
		sb.WriteString("💬 **Comments**\n\n")
	}

	sb.WriteString(output.Summary)
	sb.WriteString("\n")

	if truncated {
		sb.WriteString("\n> **Note**: The diff was truncated due to size. Some files may not have been reviewed.\n")
	}

	if len(skipped) > 0 {
		sb.WriteString("\n### Additional Notes\n")
		sb.WriteString("The following comments could not be placed inline (lines outside the diff):\n\n")
		for _, c := range skipped {
			sb.WriteString(fmt.Sprintf("- **%s** `%s:%d` — %s\n", c.Severity, c.Path, c.Line, c.Message))
		}
	}

	// Stats table
	diffLabel := formatBytes(diffSize)
	focus := "Default"
	if reviewFocus != "" {
		focus = summarizeFocus(reviewFocus)
	}

	sb.WriteString("\n| Reviewed | Detail |\n")
	sb.WriteString("|----------|--------|\n")
	sb.WriteString(fmt.Sprintf("| Files    | %d     |\n", fileCount))
	sb.WriteString(fmt.Sprintf("| Diff     | %s  |\n", diffLabel))
	sb.WriteString(fmt.Sprintf("| Model    | %s |\n", model))
	sb.WriteString(fmt.Sprintf("| Focus    | %s |\n", focus))

	sb.WriteString(fmt.Sprintf("\n---\n*[Argus](https://github.com/Simplifying-Cloud/pr-review-action) · automated review*"))

	return sb.String()
}

func formatBytes(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1f KB", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func summarizeFocus(focus string) string {
	var areas []string
	for _, line := range strings.Split(focus, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		if idx := strings.Index(line, ":"); idx > 0 {
			line = line[:idx]
		}
		if line != "" {
			areas = append(areas, line)
		}
	}
	if len(areas) == 0 {
		return "Default"
	}
	return strings.Join(areas, ", ")
}
