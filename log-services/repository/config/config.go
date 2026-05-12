package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type Config struct {
	LogLevel  string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address   string `yaml:"repository_address" env:"REPOSITORY_ADDRESS" env-default:"localhost:8082"`
	DBAddress string `yaml:"db_address" env:"DB_ADDRESS" env-default:"postgres://postgres:password@localhost:5432/postgres?sslmode=disable"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
