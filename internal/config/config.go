package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env string `yaml:"env" env-default:"prod"`
	DB
	Cache       `yaml:"cache"`
	HTTPServer  `yaml:"http_server"`
	FileStorage `yaml:"file_storage"`
	AdminToken  string `yaml:"admin_token"`
}

type DB struct {
	Addr     string `env:"POSTGRES_ADDR" env-default:"localhost"`
	Port     uint16 `env:"POSTGRES_PORT" env-default:"5432"`
	User     string `env:"POSTGRES_USER" env-required:"true"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DB       string `env:"POSTGRES_DB" env-required:"true"`
}

type Cache struct {
	Addr         string        `env:"REDIS_ADDR" env-default:"localhost"`
	Password     string        `env:"REDIS_PASSWORD" env-required:"true"`
	DB           int           `env:"REDIS_DB" env-required:"true"`
	SessionTTL   time.Duration `yaml:"session_ttl" env-default:"1h"`
	DocumentsTTL time.Duration `yaml:"documents_ttl" env-default:"1m"`
}

type FileStorage struct {
	Path string `yaml:"path" env-default:"./storage/files/"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"0.0.0.0:8082"`
	Timeout     time.Duration `yaml:"timeout" env-defalut:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-defalut:"60s"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
