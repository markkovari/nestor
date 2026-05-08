package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Adapters AdaptersConfig `mapstructure:"adapters"`
	CacheTTL int            `mapstructure:"cache_ttl"` // TTL in minutes
}

type DatabaseConfig struct {
	URL      string `mapstructure:"url"`
	NS       string `mapstructure:"ns"`
	DB       string `mapstructure:"db"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"` // "gemini", "openai", "ollama"
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	BaseURL  string `mapstructure:"base_url"`
}

type AdaptersConfig struct {
	GitHub GitHubConfig `mapstructure:"github"`
	Jira   JiraConfig   `mapstructure:"jira"`
	Linear LinearConfig `mapstructure:"linear"`
}

type GitHubConfig struct {
	Token string   `mapstructure:"token"`
	Repos []string `mapstructure:"repos"`
}

type JiraConfig struct {
	Domain string `mapstructure:"domain"`
	User   string `mapstructure:"user"`
	Token  string `mapstructure:"token"`
}

type LinearConfig struct {
	APIKey string `mapstructure:"api_key"`
}

func (c *Config) Validate() error {
	if c.LLM.Provider == "gemini" && c.LLM.APIKey == "" {
		return errors.New("llm.api_key is required when llm.provider is \"gemini\"")
	}
	if c.LLM.Provider == "gemini" && c.LLM.Model == "" {
		return errors.New("llm.model is required when llm.provider is \"gemini\"")
	}
	if c.Adapters.GitHub.Token != "" && len(c.Adapters.GitHub.Repos) == 0 {
		return errors.New("adapters.github.repos must not be empty when adapters.github.token is set")
	}
	if c.Adapters.Jira.Domain != "" && c.Adapters.Jira.Token == "" {
		return errors.New("adapters.jira.token is required when adapters.jira.domain is set")
	}
	if c.Adapters.Jira.Domain != "" && c.Adapters.Jira.User == "" {
		return errors.New("adapters.jira.user is required when adapters.jira.domain is set")
	}
	return nil
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("nestor")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.nestor")

	// Map nested keys like database.url to NESTOR_DATABASE_URL
	viper.SetEnvPrefix("NESTOR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
