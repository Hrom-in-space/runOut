package config

import (
	"github.com/caarlos0/env/v10"
)

//go:generate go run github.com/g4s8/envdoc@v0.0.10 --output ./../../env.md -field-names
type Config struct {
	// Порт приложения
	Port   string   `env:"PORT" envDefault:"8000"`
	PG     Postgres `envPrefix:"PG_"`
	OpenAI OpenAPI  `envPrefix:"OPENAI_"`
}

type OpenAPI struct {
	// API ключ
	APIKey string `env:"API_KEY"`
	// ID ассистента
	AssistantID string `env:"ASSISTANT_ID"`
}

type Postgres struct {
	// Пользователь
	Username string `env:"USER" envDefault:"app"`
	// Пароль пользователя
	Password string `env:"PWD" envDefault:"app"`
	// Хост
	Host string `envDefault:"localhost"`
	// Порт
	Port string `envDefault:"5432"`
	// Имя БД
	Database string `env:"DB" envDefault:"app"`
}

func New() (*Config, error) {
	cfg := &Config{}
	opts := env.Options{
		UseFieldNameByDefault: true,
	}

	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return nil, err
	}

	return cfg, nil
}
