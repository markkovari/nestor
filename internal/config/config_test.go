package config

import (
	"testing"
)

func TestValidate_EmptyConfig(t *testing.T) {
	c := &Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("expected no error for empty config, got: %v", err)
	}
}

func TestValidate_GeminiMissingKey(t *testing.T) {
	c := &Config{
		LLM: LLMConfig{Provider: "gemini", APIKey: ""},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for gemini with missing api_key, got nil")
	}
}

func TestValidate_GeminiMissingModel(t *testing.T) {
	c := &Config{
		LLM: LLMConfig{Provider: "gemini", APIKey: "key", Model: ""},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for gemini with missing model, got nil")
	}
}

func TestValidate_GeminiValid(t *testing.T) {
	c := &Config{
		LLM: LLMConfig{Provider: "gemini", APIKey: "k", Model: "m"},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected no error for valid gemini config, got: %v", err)
	}
}

func TestValidate_GitHubTokenNoRepos(t *testing.T) {
	c := &Config{
		Adapters: AdaptersConfig{
			GitHub: GitHubConfig{Token: "tok", Repos: nil},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for github token with no repos, got nil")
	}
}

func TestValidate_JiraDomainNoToken(t *testing.T) {
	c := &Config{
		Adapters: AdaptersConfig{
			Jira: JiraConfig{Domain: "x.atlassian.net", Token: ""},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for jira domain with no token, got nil")
	}
}

func TestValidate_JiraDomainNoUser(t *testing.T) {
	c := &Config{
		Adapters: AdaptersConfig{
			Jira: JiraConfig{Domain: "x.atlassian.net", Token: "tok", User: ""},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for jira domain with no user, got nil")
	}
}
