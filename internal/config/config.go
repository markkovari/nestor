package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Adapters AdaptersConfig `mapstructure:"adapters"`
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

func LoadConfig() (*Config, error) {
	viper.SetConfigName("nestor")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.nestor")

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

	return &config, nil
}
