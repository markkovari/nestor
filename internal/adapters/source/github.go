package source

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/markkovari/nestor/internal/core"
)

type GitHubProvider struct {
	client *github.Client
	repos  []string
}

func NewGitHubProvider(ctx context.Context, token string, repos []string) *GitHubProvider {
	client := github.NewClient(nil).WithAuthToken(token)
	return &GitHubProvider{
		client: client,
		repos:  repos,
	}
}

func (g *GitHubProvider) FetchMetadata(ctx context.Context) ([]core.CodeComponent, error) {
	var allComponents []core.CodeComponent

	for _, repoPath := range g.repos {
		// Expecting "owner/repo"
		owner := ""
		repo := ""
		fmt.Sscanf(repoPath, "%s/%s", &owner, &repo) // Simple parse, needs improvement

		// Fetch basic repo info
		r, _, err := g.client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repo %s: %w", repoPath, err)
		}

		allComponents = append(allComponents, core.CodeComponent{
			Path:        repoPath,
			Name:        r.GetName(),
			Description: r.GetDescription(),
			Metadata: map[string]string{
				"stars":    fmt.Sprint(r.GetStargazersCount()),
				"language": r.GetLanguage(),
			},
		})
	}

	return allComponents, nil
}
