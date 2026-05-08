package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
		} `json:"issues"`
	} `json:"data"`
}

func (l *LinearProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	query := `query { issues(first: 50) { nodes { id identifier title description status { name } } } }`
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

	var tasks []core.Task
	for _, node := range linResp.Data.Issues.Nodes {
		tasks = append(tasks, core.Task{
			ID:          node.Identifier,
			Title:       node.Title,
			Description: node.Description,
			Status:      node.Status.Name,
			Provider:    "linear",
		})
	}

	return tasks, nil
}

func (l *LinearProvider) UpdateTask(ctx context.Context, taskID string, description string) error {
	query := fmt.Sprintf(`mutation { issueUpdate(id: "%s", input: { description: "%s" }) { success } }`, taskID, strings.ReplaceAll(description, "\n", "\\n"))
	reqBody, _ := json.Marshal(map[string]string{"query": query})

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
