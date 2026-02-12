package config

import (
	"fmt"
	"time"
)

// Config — конфигурация приложения.
// Теги `yaml` используются для чтения из файла.
// Теги `env` используются для чтения переменных окружения.
// Теги `env-default` задают значения по умолчанию.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	GraphQL  GraphQLConfig  `yaml:"graphql"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig — конфигурация HTTP сервера.
type ServerConfig struct {
	Host           string        `yaml:"host" env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port           int           `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout    time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout   time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" env-default:"15s"`
	RequestTimeout time.Duration `yaml:"request_timeout" env:"SERVER_REQUEST_TIMEOUT" env-default:"30s"`
}

// Addr возвращает адрес сервера.
func (c ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DatabaseConfig — конфигурация базы данных.
type DatabaseConfig struct {
	Host            string        `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port            int           `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User            string        `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password        string        `yaml:"password" env:"DB_PASSWORD" env-default:"postgres"`
	Database        string        `yaml:"database" env:"DB_NAME" env-default:"my_english"`
	SSLMode         string        `yaml:"ssl_mode" env:"DB_SSLMODE" env-default:"disable"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" env-default:"5m"`
}

// DSN возвращает строку подключения к PostgreSQL в формате URL для pgx.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// GraphQLConfig — конфигурация GraphQL.
type GraphQLConfig struct {
	EnablePlayground    bool `yaml:"enable_playground" env:"GRAPHQL_PLAYGROUND" env-default:"true"`
	EnableIntrospection bool `yaml:"enable_introspection" env:"GRAPHQL_INTROSPECTION" env-default:"true"`
	QueryCacheSize      int  `yaml:"query_cache_size" env:"GRAPHQL_QUERY_CACHE_SIZE" env-default:"1000"`
}

// LogConfig — конфигурация логгера.
type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`   // debug, info, warn, error
	Format string `yaml:"format" env:"LOG_FORMAT" env-default:"text"` // json, text
}
