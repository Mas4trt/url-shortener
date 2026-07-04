# URL Shortener Service

High-performance URL shortening service built with Clean Architecture principles.  
Provides low latency through multi-level caching and high fault tolerance.

---

## 🏗 Архитектурные особенности

- **Dependency Injection**  
Использование `google/wire` для сборки зависимостей на этапе компиляции.

- **Smart Caching**  
Слой кэширования на базе Redis с использованием `singleflight` для защиты БД от эффекта "cache stampede".

- **Crypto-secure Generation**  
Генерация уникальных алиасов с использованием `crypto/rand`, обеспечивающая высокую энтропию и непредсказуемость.

- **Robust Storage**  
Послойная работа с данными: PostgreSQL как основное хранилище и Redis как горячий кэш.

- **Resilience**  
Защита от коллизий алиасов через настраиваемый механизм `max_retries`.

- **Observability**  
Структурированное логирование через `slog`.

---

## 🛠 Технологический стек

- **Язык:** Go 1.25.7  
- **База данных:** PostgreSQL 18.4  
- **Кэширование:** Redis 7  
- **Валидация:** `go-playground/validator`

---

## 🚀 Начало работы

### Требования

- Docker & Docker Compose  
- Go 1.25+  
- Make  

### Установка

Клонируйте репозиторий:

```bash
git clone <url>
cd url-shortener
```
Подготовьте переменные окружения:
```bash
cp .env.example .env
```

📦 Available Make Commands
```bash
make up            # Start Docker Compose services
make down          # Stop services
make migrate-up    # Run migrations
make migrate-down  # Rollback migrations
make test          # Run all tests
make build         # Build the application
```

🧪 Development initialization
Brings up full local environment:
PostgreSQL
Redis
Migrations
App
```bash
make init
```

📡 API Endpoints
```bash
| Метод  | Путь     | Описание                                                             |
| ------ | -------- | -------------------------------------------------------------------- |
| POST   | /url     | Сокращение ссылки. Принимает JSON `{ "url": "...", "alias": "..." }` |
| GET    | /{alias} | Редирект на оригинальную ссылку                                      |
| DELETE | /{alias} | Удаление записи из системы                                           |
| GET    | /healthz | Проверка готовности (Liveness/Readiness probe)                       |
```

🧪 Тестирование
Проект покрыт интеграционными тестами с использованием `testcontainers-go`.
Тесты автоматически запускают изолированные контейнеры с `PostgreSQL` и `Redis`.

```bash
go test -v ./internal/tests/integration/...
```

📦 Структура проекта
```bash
.
├── cmd/               # Точка входа приложения
├── internal/
│   ├── app/           # Жизненный цикл (Graceful Shutdown)
│   ├── bootstrap/     # DI (wire.go) и инициализация зависимостей
│   ├── service/      # Бизнес-логика (включая генератор алиасов)
│   ├── storage/      # Репозитории (Postgres + Redis)
│   └── transport/    # HTTP-хэндлеры и middleware
├── migrations/       # SQL миграции
└── pkg/              # Утилиты (random, logger)
```

⚙️ Конфигурация
Сервис настраивается через `config/local.yaml` или переменные окружения:
- ``DATABASE_URL`` — строка подключения к PostgreSQL
- ``REDIS_URL`` — адрес `Redis
- ``ALIAS_LENGTH`` — длина генерируемого алиаса (по умолчанию 6)
- ``TTL`` — время жизни кэша в Redis

📄 License
Developed with ❤️ using Go & Clean Architecture principles.

Инфраструктура
- Миграции: golang-migrate
- Оркестрация: Docker Compose
- Автоматизация: Makefile
