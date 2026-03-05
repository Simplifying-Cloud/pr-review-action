package github

type PullRequest struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	HeadSHA string `json:"head_sha"`
	Head    struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

func (pr *PullRequest) SHA() string {
	if pr.HeadSHA != "" {
		return pr.HeadSHA
	}
	return pr.Head.SHA
}

type PRFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"` // added, removed, modified, renamed
	Patch    string `json:"patch"`
}

type ReviewSubmission struct {
	CommitID string          `json:"commit_id"`
	Body     string          `json:"body"`
	Event    string          `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	Comments []ReviewComment `json:"comments,omitempty"`
}

type ReviewComment struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Side string `json:"side"` // RIGHT for new file lines
	Body string `json:"body"`
}
