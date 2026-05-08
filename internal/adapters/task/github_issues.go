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
			State: "open",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		for {
			issues, resp, err := g.client.Issues.ListByRepo(ctx, owner, repo, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch issues for %s: %w", repoPath, err)
			}

			for _, issue := range issues {
				if issue.IsPullRequest() {
					continue
				}
				prURLs := g.fetchLinkedPRs(ctx, owner, repo, issue.GetNumber())
				metadata := map[string]string{
					"repo": repoPath,
					"url":  issue.GetHTMLURL(),
				}
				if len(prURLs) > 0 {
					metadata["linked_prs"] = strings.Join(prURLs, ",")
				}
				allTasks = append(allTasks, core.Task{
					ID:          fmt.Sprintf("%s#%d", repo, issue.GetNumber()),
					Title:       issue.GetTitle(),
					Description: issue.GetBody(),
					Status:      issue.GetState(),
					Provider:    "github",
					Metadata:    metadata,
				})
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	return allTasks, nil
}

func (g *GitHubIssueProvider) fetchLinkedPRs(ctx context.Context, owner, repo string, issueNumber int) []string {
	opts := &github.ListOptions{PerPage: 25}
	events, _, err := g.client.Issues.ListIssueTimeline(ctx, owner, repo, issueNumber, opts)
	if err != nil {
		return nil
	}
	var prURLs []string
	for _, event := range events {
		if event.GetEvent() == "cross-referenced" {
			src := event.Source
			if src != nil && src.Issue != nil && src.Issue.IsPullRequest() {
				prURLs = append(prURLs, src.Issue.GetHTMLURL())
			}
		}
	}
	return prURLs
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

	body := description
	input := &github.IssueRequest{
		Body: &body,
	}

	_, _, err = g.client.Issues.Edit(ctx, owner, repoName, number, input)
	return err
}
