package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/markkovari/nestor/internal/core"
)

type LinearProvider struct {
	apiKey string
}

func NewLinearProvider(apiKey string) *LinearProvider {
	return &LinearProvider{apiKey: apiKey}
}

func (l *LinearProvider) Name() string {
	return "linear"
}

type linearResponse struct {
	Data struct {
		Issues struct {
			Nodes []struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Status      struct {
					Name string `json:"name"`
				} `json:"status"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	} `json:"data"`
}

func (l *LinearProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	var allTasks []core.Task
	cursor := ""

	for {
		var query string
		if cursor == "" {
			query = `query { issues(first: 50, filter: { state: { type: { nin: ["completed", "cancelled"] } } }) { nodes { id identifier title description status { name } } pageInfo { hasNextPage endCursor } } }`
		} else {
			query = fmt.Sprintf(`query { issues(first: 50, after: "%s", filter: { state: { type: { nin: ["completed", "cancelled"] } } }) { nodes { id identifier title description status { name } } pageInfo { hasNextPage endCursor } } }`, cursor)
		}

		reqBody, _ := json.Marshal(map[string]string{"query": query})
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linear.app/graphql", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", l.apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var linResp linearResponse
		if err := json.NewDecoder(resp.Body).Decode(&linResp); err != nil {
			return nil, err
		}

		for _, node := range linResp.Data.Issues.Nodes {
			allTasks = append(allTasks, core.Task{
				ID:          node.Identifier,
				Title:       node.Title,
				Description: node.Description,
				Status:      node.Status.Name,
				Provider:    "linear",
			})
		}

		if !linResp.Data.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = linResp.Data.Issues.PageInfo.EndCursor
	}

	return allTasks, nil
}

func (l *LinearProvider) UpdateTask(ctx context.Context, taskID string, description string) error {
	body := map[string]any{
		"query": `mutation IssueUpdate($id: String!, $description: String!) { issueUpdate(id: $id, input: { description: $description }) { success } }`,
		"variables": map[string]string{
			"id":          taskID,
			"description": description,
		},
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linear.app/graphql", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", l.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("linear api returned status %d", resp.StatusCode)
	}

	return nil
}
