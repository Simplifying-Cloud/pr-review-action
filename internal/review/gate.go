package review

import "fmt"

// Gate decides whether the Argus check passes. Callers gate merges (including
// auto-merge) on the check result, so a review that requests changes, or one
// that could not see the whole diff, must fail the check, not just post a review.
func Gate(verdict string, truncated bool) error {
	if verdict == "request_changes" {
		return fmt.Errorf("Argus requested changes")
	}
	if truncated {
		return fmt.Errorf("diff was truncated, so part of the PR was not reviewed")
	}
	return nil
}
