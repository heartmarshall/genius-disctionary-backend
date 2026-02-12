package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Load загружает конфигурацию приложения.
// Приоритет загрузки:
// 1. Переменные окружения (наивысший приоритет, перекрывают файл)
// 2. YAML файл конфигурации (если указан путь и файл существует)
// 3. Значения по умолчанию (указаны в struct tags)
func Load(configPath string) (*Config, error) {
	var cfg Config

	// Проверяем, передан ли путь к файлу и существует ли он
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			// Читаем из файла + ENV + Defaults
			if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
				return nil, fmt.Errorf("failed to read config from file %s: %w", configPath, err)
			}
			return &cfg, nil
		}
	}

	// Если файла нет, читаем только ENV + Defaults
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read env vars: %w", err)
	}

	return &cfg, nil
}
