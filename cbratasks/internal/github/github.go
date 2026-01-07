package github

import "time"

type Issue struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"` // Alias for ID
	Title     string    `json:"title"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	Assignee  string    `json:"assignee,omitempty"`
	Repo      string    `json:"repo,omitempty"` // Added for stub
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func FetchIssues(owner, repo string) ([]*Issue, error) {
	return []*Issue{}, nil
}

func CreateIssue(owner, repo, title, body, assignee string, labels, assignees []string) error {
	return nil
}
