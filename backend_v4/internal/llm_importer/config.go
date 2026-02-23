package llm_importer

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds llm-import settings.
type Config struct {
	LLMOutputDir string `yaml:"llm_output_dir" env:"LLM_IMPORT_OUTPUT_DIR" env-default:"./llm-output"`
	BatchSize    int    `yaml:"batch_size"      env:"LLM_IMPORT_BATCH_SIZE" env-default:"500"`
	DryRun       bool   `yaml:"dry_run"         env:"LLM_IMPORT_DRY_RUN"`
	SourceSlug   string `yaml:"source_slug"     env:"LLM_IMPORT_SOURCE_SLUG" env-default:"llm"`
}

// LoadConfig reads config from YAML file or environment variables.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				return nil, fmt.Errorf("llm-import config: %w", err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("llm-import config: file %s not found", path)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("llm-import config: read env: %w", err)
	}
	return &cfg, nil
}
