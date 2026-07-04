package main

import (
	"flag"
	"log"
	"os"

	"url-shortener/internal/bootstrap"
	"url-shortener/internal/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.MustLoad(fetchConfigPath())

	if cfg.RunMigrations {
		if err := bootstrap.RunMigrations("file://./migrations", cfg); err != nil {
			log.Fatal(err)
		}
	}

	application, cleanup, err := bootstrap.InitializeApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := application.Run(cleanup); err != nil {
		log.Fatal(err)
	}
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
		res = os.Getenv("CONFIG_PATH")
	}
	return res
}
