package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	token  string
	apiURL string
	http   *http.Client
}

func NewClient(token, apiURL string) *Client {
	return &Client{
		token:  token,
		apiURL: apiURL,
		http:   &http.Client{},
	}
}

func (c *Client) GetPR(owner, repo string, prNum int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.apiURL, owner, repo, prNum)
	body, err := c.get(url, "application/vnd.github+json")
	if err != nil {
		return nil, fmt.Errorf("fetching PR: %w", err)
	}
	var pr PullRequest
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("parsing PR: %w", err)
	}
	return &pr, nil
}

func (c *Client) GetPRDiff(owner, repo string, prNum int) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.apiURL, owner, repo, prNum)
	body, err := c.get(url, "application/vnd.github.v3.diff")
	if err != nil {
		return "", fmt.Errorf("fetching diff: %w", err)
	}
	return string(body), nil
}

func (c *Client) GetPRFiles(owner, repo string, prNum int) ([]PRFile, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=100", c.apiURL, owner, repo, prNum)
	body, err := c.get(url, "application/vnd.github+json")
	if err != nil {
		return nil, fmt.Errorf("fetching files: %w", err)
	}
	var files []PRFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("parsing files: %w", err)
	}
	return files, nil
}

func (c *Client) SubmitReview(owner, repo string, prNum int, review *ReviewSubmission) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", c.apiURL, owner, repo, prNum)
	payload, err := json.Marshal(review)
	if err != nil {
		return fmt.Errorf("marshaling review: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("submitting review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("review submission failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) get(url, accept string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}
