package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Action string `yaml:"action"`
}

type Config struct {
	Services      map[string]string `yaml:"services"`
	DefaultAction string            `yaml:"default_action"`
	Policies      []Policy          `yaml:"policies"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.DefaultAction == "" {
		cfg.DefaultAction = "deny"
	}
	return &cfg, nil
}
