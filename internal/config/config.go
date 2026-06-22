package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	GrpcPort int `yaml:"grpc_port"`
	HttpPort int `yaml:"http_port"`
}

type AnthropicConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type ProvidersConfig struct {
	Anthropic AnthropicConfig `yaml:"anthropic"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
}

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Providers ProvidersConfig `yaml:"providers"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Server.HttpPort == 0 {
		cfg.Server.HttpPort = 8080
	}
	return &cfg, nil
}
