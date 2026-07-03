package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	PublicBaseURL string `json:"public_base_url"`
	UploadDir     string `json:"upload_dir"`
	DataFile      string `json:"data_file"`
	MaxUploadMB   int64  `json:"max_upload_mb"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		Host:          "0.0.0.0",
		Port:          8080,
		PublicBaseURL: "http://localhost:8080",
		UploadDir:     "data/uploads",
		DataFile:      "data/apks.json",
		MaxUploadMB:   2048,
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := json.Unmarshal(content, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %q: %w", path, err)
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if cfg.MaxUploadMB <= 0 {
		return cfg, fmt.Errorf("max_upload_mb must be greater than 0")
	}
	if cfg.UploadDir == "" {
		return cfg, fmt.Errorf("upload_dir is required")
	}
	if cfg.DataFile == "" {
		return cfg, fmt.Errorf("data_file is required")
	}
	return cfg, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
