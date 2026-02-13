package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Load reads configuration from a YAML file and environment variables.
// Priority: ENV > YAML > defaults (via env-default tags).
// The YAML file path is determined by CONFIG_PATH env (fallback "./config.yaml").
// If the file does not exist and CONFIG_PATH was not set explicitly,
// configuration is loaded from ENV + defaults only.
func Load() (*Config, error) {
	var cfg Config

	path := os.Getenv("CONFIG_PATH")
	explicitPath := path != ""
	if !explicitPath {
		path = "./config.yaml"
	}

	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("config: read %s: %w", path, err)
		}
	} else if explicitPath {
		return nil, fmt.Errorf("config: file %s: %w", path, err)
	} else {
		// No file, load from ENV + defaults only.
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("config: read env: %w", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: validate: %w", err)
	}

	return &cfg, nil
}
