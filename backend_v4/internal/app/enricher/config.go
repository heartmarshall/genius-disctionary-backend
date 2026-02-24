package enricher

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds enricher pipeline settings.
type Config struct {
	WordListPath    string `yaml:"word_list_path"    env:"ENRICH_WORD_LIST_PATH"`
	WiktionaryPath  string `yaml:"wiktionary_path"   env:"ENRICH_WIKTIONARY_PATH"`
	WordNetPath     string `yaml:"wordnet_path"      env:"ENRICH_WORDNET_PATH"`
	CMUPath         string `yaml:"cmu_path"          env:"ENRICH_CMU_PATH"`
	EnrichOutputDir string `yaml:"enrich_output_dir" env:"ENRICH_OUTPUT_DIR"      env-default:"./enrich-output"`
	LLMOutputDir    string `yaml:"llm_output_dir"    env:"ENRICH_LLM_OUTPUT_DIR"  env-default:"./llm-output"`
	Mode            string `yaml:"mode"              env:"ENRICH_MODE"             env-default:"manual"`
	Source          string `yaml:"source"            env:"ENRICH_SOURCE"           env-default:"file"`
	BatchSize       int    `yaml:"batch_size"        env:"ENRICH_BATCH_SIZE"       env-default:"50"`
	LLMAPIKey       string `yaml:"llm_api_key"       env:"ENRICH_LLM_API_KEY"`
	LLMModel        string `yaml:"llm_model"         env:"ENRICH_LLM_MODEL"        env-default:"claude-opus-4-6"`
	DatabaseDSN     string `yaml:"database_dsn"      env:"DATABASE_DSN"`
}

// LoadConfig reads enricher config from YAML or environment variables.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				return nil, fmt.Errorf("enrich config: %w", err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("enrich config: file %s not found", path)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("enrich config: read env: %w", err)
	}
	return &cfg, nil
}
