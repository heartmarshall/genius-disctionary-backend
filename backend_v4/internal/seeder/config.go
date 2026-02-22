package seeder

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds seeder pipeline settings.
type Config struct {
	WiktionaryPath     string `env:"SEEDER_WIKTIONARY_PATH"`
	NGSLPath           string `env:"SEEDER_NGSL_PATH"`
	NAWLPath           string `env:"SEEDER_NAWL_PATH"`
	CMUPath            string `env:"SEEDER_CMU_PATH"`
	WordNetPath        string `env:"SEEDER_WORDNET_PATH"`
	TatoebaPath        string `env:"SEEDER_TATOEBA_PATH"`
	TopN               int    `env:"SEEDER_TOP_N"          env-default:"20000"`
	BatchSize          int    `env:"SEEDER_BATCH_SIZE"      env-default:"500"`
	MaxExamplesPerWord int    `env:"SEEDER_MAX_EXAMPLES"    env-default:"5"`
	DryRun             bool   `env:"SEEDER_DRY_RUN"`
}

// LoadConfig reads seeder configuration from a YAML file and environment variables.
// Priority: ENV > YAML > defaults (via env-default tags).
func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				return nil, fmt.Errorf("seeder config: read %s: %w", path, err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("seeder config: file %s not found", path)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("seeder config: read env: %w", err)
	}

	return &cfg, nil
}
