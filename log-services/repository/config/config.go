package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"strings"
)

type Config struct {
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address     string `yaml:"repository_address" env:"REPOSITORY_ADDRESS" env-default:"localhost:8082"`
	DBAddress   string `yaml:"db_address" env:"DB_ADDRESS" env-default:"postgres://postgres:password@localhost:5432/postgres?sslmode=disable"`
	DatabaseURL string `env:"DATABASE_URL"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}

	if databaseURL := strings.TrimSpace(cfg.DatabaseURL); databaseURL != "" {
		cfg.DBAddress = databaseURL
	}

	return cfg
}
