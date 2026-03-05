package github

import (
	"regexp"
	"strconv"
	"strings"
)

type DiffFile struct {
	Path  string
	Lines map[int]bool // line numbers in the new file that are commentable
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// ParseDiffFiles extracts commentable line numbers from a unified diff.
// Returns a map of file path → set of new-file line numbers within hunks.
func ParseDiffFiles(diff string) map[string]*DiffFile {
	files := make(map[string]*DiffFile)
	var current *DiffFile

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			current = nil
			continue
		}
		if strings.HasPrefix(line, "+++ b/") {
			path := strings.TrimPrefix(line, "+++ b/")
			current = &DiffFile{Path: path, Lines: make(map[int]bool)}
			files[path] = current
			continue
		}
		if current == nil {
			continue
		}

		matches := hunkHeader.FindStringSubmatch(line)
		if matches != nil {
			startLine, _ := strconv.Atoi(matches[1])
			lineCount := 1
			if matches[2] != "" {
				lineCount, _ = strconv.Atoi(matches[2])
			}
			// Mark all lines in this hunk range as commentable
			for i := startLine; i < startLine+lineCount; i++ {
				current.Lines[i] = true
			}
			continue
		}
	}
	return files
}

// IsCommentable checks if a (file, line) pair can receive an inline comment.
func IsCommentable(files map[string]*DiffFile, path string, line int) bool {
	f, ok := files[path]
	if !ok {
		return false
	}
	return f.Lines[line]
}
