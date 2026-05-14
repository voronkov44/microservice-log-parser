package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address  string `yaml:"parser_address" env:"PARSER_ADDRESS" env-default:"localhost:8081"`
	DataDir  string `yaml:"data_dir" env:"DATA_DIR" env-default:"../data"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
