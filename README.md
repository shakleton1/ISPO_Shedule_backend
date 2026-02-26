# ISPO Smart Schedule (Backend)

Backend для системы "Умное расписание" (колледж ИСПО СПбПУ).

## Стек

- Go 1.21+
- Gin
- PostgreSQL 15+
- GORM
- Config: Viper (yaml/env)
- Migrations: Goose
- PDF: chromedp (Headless Chrome)

## Быстрый старт (dev)

1. Создайте БД PostgreSQL и пользователя.

2. Скопируйте конфиг:

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
```

3. Примените миграции (goose):

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = "host=localhost port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable"
goose -dir .\db\migrations up
```

4. Запуск API:

```powershell
go run .\cmd\api
```

По умолчанию сервис стартует на `127.0.0.1:8080`.

## Основные эндпоинты

- `GET /api/v1/health`
- `GET /api/v1/schedule/current?group_id=1&date=2026-02-26`
- `GET /api/v1/schedule/version`
- `GET /api/v1/schedule/range?group_id=1&date_start=2026-02-24&date_end=2026-03-07`
- `GET /api/v1/schedule/pdf?group_id=1&date_start=2026-02-24`

## Аутентификация

Auth endpoints:

- `POST /api/v1/auth/login` → `{ "access_token": "..." }`
- `GET /api/v1/auth/me` (нужен заголовок `Authorization: Bearer <token>`)

Bootstrap admin (dev): в `configs/config.example.yaml` есть `auth.bootstrap_admin_login/password`.

## Admin API

Admin защищен одним из способов:

1. JWT (рекомендуется): `Authorization: Bearer <token>` и роль `admin` или `dispatcher`.
2. (Опционально) Если задан `admin.api_key`, можно передать `X-Admin-Key` как аварийный обход.

- `POST /api/v1/admin/override`
- `POST /api/v1/admin/overlay`

Полный CRUD админ-справочников и расписания — в роутере (см. `internal/http`).
