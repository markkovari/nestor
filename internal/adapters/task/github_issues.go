package task

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/markkovari/nestor/internal/core"
)

type GitHubIssueProvider struct {
	client *github.Client
	repos  []string // owner/repo format
}

func NewGitHubIssueProvider(token string, repos []string) *GitHubIssueProvider {
	client := github.NewClient(nil).WithAuthToken(token)
	return &GitHubIssueProvider{
		client: client,
		repos:  repos,
	}
}

func (g *GitHubIssueProvider) Name() string {
	return "github"
}

func (g *GitHubIssueProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	var allTasks []core.Task

	for _, repoPath := range g.repos {
		parts := strings.Split(repoPath, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		opts := &github.IssueListByRepoOptions{
			State: "all",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		issues, _, err := g.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch issues for %s: %w", repoPath, err)
		}

		for _, issue := range issues {
			if issue.IsPullRequest() {
				continue
			}
			allTasks = append(allTasks, core.Task{
				ID:          fmt.Sprintf("%s#%d", repo, issue.GetNumber()),
				Title:       issue.GetTitle(),
				Description: issue.GetBody(),
				Status:      issue.GetState(),
				Provider:    "github",
				Metadata: map[string]string{
					"repo": repoPath,
					"url":  issue.GetHTMLURL(),
				},
			})
		}
	}

	return allTasks, nil
}

func (g *GitHubIssueProvider) UpdateTask(ctx context.Context, taskID string, description string) error {
	parts := strings.Split(taskID, "#")
	if len(parts) != 2 {
		return fmt.Errorf("invalid github task id: %s", taskID)
	}
	repoName, numberStr := parts[0], parts[1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", numberStr)
	}

	owner := ""
	for _, r := range g.repos {
		if strings.HasSuffix(r, "/"+repoName) {
			owner = strings.Split(r, "/")[0]
			break
		}
	}

	if owner == "" {
		return fmt.Errorf("could not determine owner for repo %s", repoName)
	}

	input := &github.IssueRequest{
		Body: github.String(description),
	}

	_, _, err = g.client.Issues.Edit(ctx, owner, repoName, number, input)
	return err
}
