package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type Config struct {
	LogLevel          string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address           string `yaml:"topology_address" env:"TOPOLOGY_ADDRESS" env-default:"localhost:8083"`
	RepositoryAddress string `yaml:"repository_address" env:"REPOSITORY_GRPC_ADDRESS" env-default:"localhost:8082"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
