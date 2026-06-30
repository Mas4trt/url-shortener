package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит все настройки приложения
type Config struct {
	Env          string     `yaml:"env" env-required:"true"`
	StoragePath  string     `yaml:"storage_path" env-required:"true"`
	ServerConfig HTTPServer `yaml:"http_server"`
}

// HTTPServer содержит настройки для запуска HTTP-сервера
type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

// MustLoad загружает конфигурацию и паникует в случае ошибки.
func MustLoad(configPath string) *Config {
	if configPath == "" {
		panic("config path is empty")
	}

	_, err := os.Stat(configPath)

	if errors.Is(err, os.ErrNotExist) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	// Читаем конфиг и оборачиваем ошибку, если что-то пошло не так
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(fmt.Sprintf("failed to read config: %v", err))
	}

	return &cfg
}
