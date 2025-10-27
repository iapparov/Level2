# Calendar

Calendar — микросервис HTTP-сервер для работы с небольшим календарем событий. 

---

## Состав репозитория

- **cmd/main.go** — точка входа, запуск через Fx DI.
- **internal/**
  - **app/** — модели данных (Calendar, Calendar req).
  - **config/** — загрузка конфигурации из YAML.
  - **repository/** — работа с in-memory хранилищем.
  - **di/** — DI-компоненты для Fx.
  - **web/** — HTTP-обработчики и роутер.
- **config/local.yaml** — пример конфигурации.
- **docs/** — Swagger-документация.

---

## Быстрый старт

### 1. Настройка конфигурации

Проверьте файл [`config/local.yaml`](config/local.yaml):

```yaml
env: local
http_port: 8080
```

При выборе env: prod – logs пишутся в json


Укажите путь к конфигу через переменную окружения:

```sh
export CONFIG_PATH=config/local.yaml
```

### 4. Запуск сервиса

```sh
go mod tidy
go run ./cmd/main.go
```

Сервис стартует на порту 8080.

---

## API

- **POST /create_event** — создание нового события;
- **POST /update_event** — обновление существующего;
- **POST /delete_event** — удаление;
- **GET /events_for_day** — получить все события на день;
- **GET /events_for_week** — события на неделю;
- **GET /events_for_month** — события на месяц.
- **Swagger**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)


---

## Тесты

- Юнит-тесты: `go test ./...`
---


## Зависимости

- Go 1.25+
- **Chi** (`github.com/go-chi/chi/v5`) — лёгкий HTTP-роутер
- **Zap** (`go.uber.org/zap`) — структурированный логгер
- **Fx** (`go.uber.org/fx`) — DI-фреймворк для зависимостей
- Swagger (для документации)

---

## Swagger

- Swagger: [docs/swagger.yaml](docs/swagger.yaml)
- Документация генерируется автоматически и доступна по `/swagger/*`.