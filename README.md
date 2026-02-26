# ISPO Smart Schedule (Backend)

Backend для системы "Умное расписание" (колледж ИСПО СПбПУ).

Сервис хранит недельные шаблоны расписания, применяет оверрайды на конкретные даты, учитывает переносы дней (calendar exceptions) и отдает клиенту JSON/PDF. Для админки есть CRUD + импорт шаблонов из CSV/XLSX.

## Стек

- Go 1.21+
- Gin
- PostgreSQL 15+
- GORM
- Config: Viper (yaml/env)
- Миграции: Goose
- PDF: chromedp (Headless Chrome/Chromium)
- Push: Firebase Cloud Messaging (FCM)
- Observability: Prometheus `/metrics`, логирование

## Быстрый старт (Windows / dev)

### 1) Поднять PostgreSQL

Проще всего через Docker:

```powershell
docker compose up -d
```

По умолчанию Postgres будет доступен на `localhost:5432`.

### 2) Конфиг

Скопируйте пример и отредактируйте при необходимости:

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
```

Ключевые параметры (см. `configs/config.example.yaml`):

- `server.addr`
- `db.*` (host/port/user/password/name/sslmode)
- `schedule.semester_start_date` (важно для четности недель)
- `auth.jwt_secret`, `auth.bootstrap_admin_login/password`
- `admin.api_key` (опционально)
- `push.*` (FCM)
- `pdf.chrome_executable_path` (если chromedp не находит браузер)

### 3) Миграции (goose)

Установите goose и примените миграции:

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = "host=localhost port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable"
goose -dir .\db\migrations up
```

### 4) Запуск API

```powershell
go run .\cmd\api
```

По умолчанию сервис стартует на `127.0.0.1:8080`.

### 5) Проверка

- Health: `GET http://127.0.0.1:8080/api/v1/health`
- OpenAPI: `GET http://127.0.0.1:8080/openapi.yaml`
- Swagger UI: `GET http://127.0.0.1:8080/docs`

## Эндпоинты (кратко)

### Публичные (без auth)

- `GET /api/v1/health`
- `GET /api/v1/metrics/health`
- `GET /metrics` (Prometheus)
- `GET /openapi.yaml`
- `GET /docs` (Swagger UI)

Расписание:

- `GET /api/v1/schedule/current?group_id=1&date=2026-02-26`
- `GET /api/v1/schedule/range?group_id=1&date_start=2026-02-24&date_end=2026-03-07`
- `GET /api/v1/schedule/version`
- `GET /api/v1/schedule/pdf?group_id=1&date_start=2026-02-24` (опционально `date_end`)

Справочники для клиента:

- `GET /api/v1/groups`
- `GET /api/v1/subjects`
- `GET /api/v1/locations`

### Auth

- `POST /api/v1/auth/login` → `{ "access_token": "..." }`
- `GET /api/v1/auth/me` (нужен `Authorization: Bearer <token>`)

### Push (FCM)

- `POST /api/v1/push/register`
- `POST /api/v1/push/unregister`

### Admin (`/api/v1/admin/*`)

Админ защищен одним из способов:

1. JWT: `Authorization: Bearer <token>`
2. (Опционально) `X-Admin-Key` если задан `admin.api_key` (аварийный обход)

Роли:

- `admin` — полный доступ
- `dispatcher` — изменение расписания (шаблоны/оверрайды/оверлеи/исключения) без редактирования справочников
- `viewer` — только чтение админских списков (шаблоны/оверрайды/исключения/справочники)

Операции (основное):

- Импорт шаблонов: `POST /api/v1/admin/import/templates/csv|xlsx`
- Шаблоны: `GET/POST/PUT/DELETE /api/v1/admin/templates`
- Оверрайды: `GET /api/v1/admin/overrides`, `POST /api/v1/admin/override`, `PUT/DELETE /api/v1/admin/overrides/:id`
- Оверлей дня: `POST /api/v1/admin/overlay`
- Calendar exceptions: `GET/POST /api/v1/admin/calendar-exceptions`, `DELETE /api/v1/admin/calendar-exceptions/:date`

Полный список и схемы — в `docs/openapi.yaml`.

## Импорт шаблонов CSV/XLSX

Импорт работает в режиме "заменить всё для группы":

- сначала удаляются все `schedule_templates` для `group_id`
- затем вставляются строки из файла
- после успешного импорта увеличивается `system_state.schedule_version` и (если включено) отправляется push

Обязательные колонки (CSV):

- `day_of_week` (0..5, где 0=Пн)
- `week_parity` (`numerator`|`denominator`|`both`)
- `pair_number` (1..8)
- `subject`
- `location`
- `teacher_name`

Опционально:

- `subgroup` (1 или 2)

## Важные правила и ограничения БД

### Нормализация преподавателей

Преподаватели вынесены в таблицу `teachers`:

- `schedule_templates.teacher_id` → `teachers(id)`
- `schedule_overrides.new_teacher_id` → `teachers(id)`

API при этом по-прежнему использует поле `teacher_name`/`new_teacher_name` (строка): репозиторий автоматически создаёт/находит запись в `teachers` при записи.

### Уникальность и политика конфликтов

- Для `schedule_overrides` действует уникальность: один оверрайд на `(group_id, target_date, pair_number, subgroup)`.
- Для `schedule_templates` действует уникальность: один шаблон на `(group_id, day_of_week, week_parity, pair_number, subgroup)`.

Если в базе исторически были дубликаты, миграции детерминированно оставляют "лучший" вариант (как правило, самый новый по `updated_at`, при равенстве — по `id`).

## PDF

PDF строится через chromedp и требует доступного Chrome/Chromium.

Если браузер не находится автоматически, укажите путь в `pdf.chrome_executable_path` (например, `C:\Program Files\Google\Chrome\Application\chrome.exe`).

## Observability

- Логи: zerolog (`log.level`, `log.pretty`)
- Prometheus метрики: `GET /metrics`
- Health DB: `GET /api/v1/metrics/health`

## Разработка

Прогон тестов:

```powershell
go test ./...
```
