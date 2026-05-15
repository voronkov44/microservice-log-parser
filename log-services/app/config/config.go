package config

import (
	"log"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Address           string        `yaml:"address" env:"APP_ADDRESS" env-default:"localhost:8080"`
	Port              string        `env:"PORT"`
	Timeout           time.Duration `yaml:"timeout" env:"APP_TIMEOUT" env-default:"5s"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"APP_READ_HEADER_TIMEOUT" env-default:"5s"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" env:"APP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type Config struct {
	LogLevel    string     `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	LogFilePath string     `yaml:"log_file_path" env:"LOG_FILE_PATH" env-default:"logs/app.log"`
	HTTPConfig  HTTPConfig `yaml:"api_server"`

	ParserAddress     string `yaml:"parser_address" env:"PARSER_GRPC_ADDRESS" env-default:"localhost:8081"`
	RepositoryAddress string `yaml:"repository_address" env:"REPOSITORY_GRPC_ADDRESS" env-default:"localhost:8082"`
	TopologyAddress   string `yaml:"topology_address" env:"TOPOLOGY_GRPC_ADDRESS" env-default:"localhost:8083"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}

	if port := strings.TrimSpace(cfg.HTTPConfig.Port); port != "" {
		cfg.HTTPConfig.Address = ":" + port
	}

	return cfg
}
