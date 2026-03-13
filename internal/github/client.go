package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client wraps the gh CLI
type Client struct{}

// PullRequest represents a PR
type PullRequest struct {
	Number    int
	Title     string
	Author    string
	MergedAt  *time.Time
	URL       string
	BaseBranch string // Used to store repo name
}

// Commit represents a commit with stats
type Commit struct {
	SHA       string
	Message   string
	Additions int
	Deletions int
}

// Repository represents a GitHub repository
type Repository struct {
	Name     string
	FullName string
	Owner    string
}

// ghPR represents the JSON structure from gh pr list
type ghPR struct {
	Number   int     `json:"number"`
	Title    string  `json:"title"`
	MergedAt *string `json:"mergedAt"`
	URL      string  `json:"url"`
	Author   struct {
		Login string `json:"login"`
	} `json:"author"`
}

// NewClient creates a new GitHub client using gh CLI
func NewClient() *Client {
	return &Client{}
}

// GetAuthenticatedUser returns the authenticated user's login using gh CLI
func (c *Client) GetAuthenticatedUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to run gh: %w", err)
	}

	login := strings.TrimSpace(string(output))
	return login, nil
}

// FetchMergedPRsFromRepos fetches merged PRs from multiple repos within a date range
func (c *Client) FetchMergedPRsFromRepos(repos []Repository, since, until time.Time, authorFilter string) ([]PullRequest, error) {
	var allPRs []PullRequest

	for _, repo := range repos {
		prs, err := c.fetchMergedPRsFromRepo(repo.FullName, since, until, authorFilter)
		if err != nil {
			// Skip repos that fail (might not have access)
			continue
		}
		allPRs = append(allPRs, prs...)
	}

	return allPRs, nil
}

func (c *Client) fetchMergedPRsFromRepo(repoFullName string, since, until time.Time, authorFilter string) ([]PullRequest, error) {
	args := []string{
		"pr", "list",
		"--repo", repoFullName,
		"--state", "merged",
		"--json", "number,title,mergedAt,url,author",
		"--limit", "100",
	}

	if authorFilter != "" {
		args = append(args, "--author", authorFilter)
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var ghPRs []ghPR
	if err := json.Unmarshal(output, &ghPRs); err != nil {
		return nil, err
	}

	var prs []PullRequest
	for _, gpr := range ghPRs {
		var mergedAt *time.Time
		if gpr.MergedAt != nil && *gpr.MergedAt != "" {
			if t, err := time.Parse(time.RFC3339, *gpr.MergedAt); err == nil {
				mergedAt = &t
			}
		}

		// Filter by date range
		if mergedAt != nil {
			mergedTime := mergedAt.UTC()
			sinceTime := since.UTC()
			untilTime := until.UTC()
			if mergedTime.Before(sinceTime) || mergedTime.After(untilTime) {
				continue
			}
		} else {
			continue
		}

		prs = append(prs, PullRequest{
			Number:     gpr.Number,
			Title:      gpr.Title,
			Author:     gpr.Author.Login,
			MergedAt:   mergedAt,
			URL:        gpr.URL,
			BaseBranch: repoFullName,
		})
	}

	return prs, nil
}

// GetPRCommits returns all commits in a PR with their stats
func (c *Client) GetPRCommits(owner, repo string, prNumber int) ([]Commit, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/commits", owner, repo, prNumber)
	cmd := exec.Command("gh", "api", endpoint, "--jq", ".[] | {sha: .sha, message: .commit.message}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []Commit
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var c struct {
			SHA     string `json:"sha"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}

		// Skip merge commits
		if strings.HasPrefix(c.Message, "Merge branch") || 
			strings.HasPrefix(c.Message, "Merge pull request") ||
			strings.HasPrefix(c.Message, "Merge remote-tracking branch") {
			continue
		}

		// Get stats for this commit
		statsEndpoint := fmt.Sprintf("repos/%s/%s/commits/%s", owner, repo, c.SHA)
		statsCmd := exec.Command("gh", "api", statsEndpoint, "--jq", ".stats")
		statsOutput, err := statsCmd.Output()
		if err != nil {
			commits = append(commits, Commit{SHA: c.SHA, Message: c.Message})
			continue
		}

		var stats struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		}
		json.Unmarshal(statsOutput, &stats)

		commits = append(commits, Commit{
			SHA:       c.SHA,
			Message:   c.Message,
			Additions: stats.Additions,
			Deletions: stats.Deletions,
		})
	}

	return commits, nil
}

// CheckGhInstalled verifies that gh CLI is installed and authenticated
func CheckGhInstalled() error {
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not installed or not authenticated. Run 'gh auth login' first")
	}
	return nil
}
