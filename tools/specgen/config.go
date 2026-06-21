package main

import "gopkg.in/yaml.v3"

// AppConfig holds the pinned version info for a single UniFi application.
type AppConfig struct {
	Default  string   `yaml:"default"`
	Versions []string `yaml:"versions"`
}

// Config is the top-level structure of specs/versions.yaml.
type Config struct {
	Mirror string               `yaml:"mirror"`
	Retain string               `yaml:"retain"`
	Apps   map[string]AppConfig `yaml:"apps"`
}

// LoadConfig parses a versions.yaml byte slice into a Config.
func LoadConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
