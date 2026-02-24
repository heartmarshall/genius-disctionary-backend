package seeder

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds seeder pipeline settings.
type Config struct {
	WiktionaryPath     string `yaml:"wiktionary_path"      env:"SEEDER_WIKTIONARY_PATH"`
	NGSLPath           string `yaml:"ngsl_path"            env:"SEEDER_NGSL_PATH"`
	NAWLPath           string `yaml:"nawl_path"            env:"SEEDER_NAWL_PATH"`
	CMUPath            string `yaml:"cmu_path"             env:"SEEDER_CMU_PATH"`
	WordNetPath        string `yaml:"wordnet_path"         env:"SEEDER_WORDNET_PATH"`
	TatoebaPath        string `yaml:"tatoeba_path"         env:"SEEDER_TATOEBA_PATH"`
	TopN               int    `yaml:"top_n"                env:"SEEDER_TOP_N"          env-default:"20000"`
	BatchSize          int    `yaml:"batch_size"           env:"SEEDER_BATCH_SIZE"      env-default:"500"`
	MaxExamplesPerWord int    `yaml:"max_examples_per_word" env:"SEEDER_MAX_EXAMPLES"   env-default:"5"`
	DryRun             bool   `yaml:"dry_run"              env:"SEEDER_DRY_RUN"`
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
