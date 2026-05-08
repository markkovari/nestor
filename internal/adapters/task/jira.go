package task

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/markkovari/nestor/internal/core"
)

type JiraProvider struct {
	domain string
	auth   string // base64(user:token)
}

func NewJiraProvider(domain, user, token string) *JiraProvider {
	creds := base64.StdEncoding.EncodeToString([]byte(user + ":" + token))
	return &JiraProvider{domain: domain, auth: creds}
}

func (j *JiraProvider) Name() string { return "jira" }

type jiraSearchResponse struct {
	StartAt    int `json:"startAt"`
	MaxResults int `json:"maxResults"`
	Total      int `json:"total"`
	Issues     []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string   `json:"summary"`
			Description *jiraDoc `json:"description"`
			Status      struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	} `json:"issues"`
}

type jiraDoc struct {
	Type    string      `json:"type"`
	Content []jiraBlock `json:"content"`
}

type jiraBlock struct {
	Type    string      `json:"type"`
	Text    string      `json:"text"`
	Content []jiraBlock `json:"content"`
}

func extractText(doc *jiraDoc) string {
	if doc == nil {
		return ""
	}
	var parts []string
	var walk func(blocks []jiraBlock)
	walk = func(blocks []jiraBlock) {
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
			walk(b.Content)
		}
	}
	walk(doc.Content)
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

func (j *JiraProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	var allTasks []core.Task
	startAt := 0
	maxResults := 100

	for {
		apiURL := fmt.Sprintf("https://%s/rest/api/3/search?jql=%s&startAt=%d&maxResults=%d",
			j.domain,
			url.QueryEscape("project is not EMPTY AND statusCategory != Done ORDER BY updated DESC"),
			startAt, maxResults)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Basic "+j.auth)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("jira api returned status %d", resp.StatusCode)
		}

		var result jiraSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			allTasks = append(allTasks, core.Task{
				ID:          issue.Key,
				Title:       issue.Fields.Summary,
				Description: extractText(issue.Fields.Description),
				Status:      issue.Fields.Status.Name,
				Provider:    "jira",
			})
		}

		if startAt+maxResults >= result.Total {
			break
		}
		startAt += maxResults
	}

	return allTasks, nil
}

func (j *JiraProvider) UpdateTask(ctx context.Context, taskID string, description string) error {
	apiURL := fmt.Sprintf("https://%s/rest/api/3/issue/%s", j.domain, taskID)

	body := map[string]any{
		"fields": map[string]any{
			"description": map[string]any{
				"type":    "doc",
				"version": 1,
				"content": []map[string]any{
					{
						"type": "paragraph",
						"content": []map[string]any{
							{"type": "text", "text": description},
						},
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Basic "+j.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("jira update returned status %d", resp.StatusCode)
	}
	return nil
}
