package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит все настройки приложения
type Config struct {
	Env         string     `yaml:"env" env-required:"true"`
	StoragePath string     `yaml:"storage_path" env-required:"true"`
	HTTPServer  HTTPServer `yaml:"http_server"`
}

// HTTPServer содержит настройки для запуска HTTP-сервера
type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

// MustLoad загружает конфигурацию и паникует в случае ошибки.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("critical error loading config: %v", err))
	}

	return cfg
}

func Load() (*Config, error) {
	configPath := fetchConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("config path is empty(set GONFIG_PATH or --config flag)")
	}

	// Проверяем, существует ли файл физически
	_, err := os.Stat(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("config file missing: %s", configPath)
	} else if err != nil {
		return nil, fmt.Errorf("error checking config file: %w", err)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return &cfg, err
}

// fetchConfigPath выбирает откуда взять путь к конфигу
// Приоритет: флаг командной строки -> переменная окружения
func fetchConfigPath() string {
	var res string

	// flag.StringVar позволяет передавать --config="path/to/config.yaml"
	// Проверка flag.Parsed() нужна, чтобы не вызывать Parse повторно в тестах
	if !flag.Parsed() {
		flag.StringVar(&res, "config", "", "path to configuration file")
		flag.Parse()
	}

	// Если флаг пустой, смотрим в окружение
	if res == "" {
		res = os.Getenv("GONFIG_PATH")
	}

	return res
}
