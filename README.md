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

### 3.5) Seed (минимальные данные для dev)

Создаёт минимальный набор: специальность/учебный план/академ. календарь + несколько недель, 2–3 предмета, 2 локации, 1 группа, базовые пользователи.

```powershell
go run .\cmd\seed
```

По умолчанию создаются пользователи (dev):

- `admin` / `admin`
- `dispatcher` / `dispatcher`
- `viewer` / `viewer`
- `student1` / `student1`

Пароли можно переопределить через env:

- `ISPO_SEED_ADMIN_PASSWORD`
- `ISPO_SEED_DISPATCHER_PASSWORD`
- `ISPO_SEED_VIEWER_PASSWORD`
- `ISPO_SEED_STUDENT_PASSWORD`

### 4) Запуск API

```powershell
go run .\cmd\api
```

По умолчанию сервис стартует на `127.0.0.1:8080`.

### 5) Проверка

- Health: `GET http://127.0.0.1:8080/api/v1/health`
- OpenAPI: `GET http://127.0.0.1:8080/openapi.yaml`
- Swagger UI: `GET http://127.0.0.1:8080/docs`

## DX (удобные команды)

Для Windows есть простые скрипты в `scripts/`:

- Postgres: `powershell -File .\scripts\db-up.ps1`
- Миграции: `powershell -File .\scripts\migrate-up.ps1`
- Seed: `powershell -File .\scripts\seed.ps1`
- Run API: `powershell -File .\scripts\run-api.ps1`
- Test: `powershell -File .\scripts\test.ps1`
- Lint: `powershell -File .\scripts\lint.ps1`

Также добавлен `Taskfile.yml` (если используете task runner):

- `task db:up`
- `task migrate:up`
- `task seed`
- `task run`
- `task test`
- `task lint`

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
- `dispatcher` — изменение фактических занятий, замен, оверлеев и событий без редактирования справочников
- `viewer` — только чтение админских списков

Операции (основное):

- Фактические занятия: `GET/POST/PATCH/DELETE /api/v1/admin/schedule-lessons`, `POST /api/v1/admin/schedule-lessons/:id/cancel`
- Замены/добавления/отмены: `POST /api/v1/admin/schedule-overrides/apply`
- Журнал замен: `GET /api/v1/admin/overrides`, `GET /api/v1/admin/reports/schedule-overrides`
- Оверлей дня: `POST /api/v1/admin/overlay`
- События дня (структурированные, не только текст): `GET/POST /api/v1/admin/day-events`, `PUT/DELETE /api/v1/admin/day-events/:id`

Жизненный цикл и “источник истины” (policy):

- `schedule_lessons` — единственный источник актуального расписания на конкретные даты.
- `room_assignments` — фактический кабинет конкретной пары через `schedule_lesson_id`.
- `schedule_overrides` — примененная операция `add|replace|cancel|restore` со snapshot “что было” и “что стало” для отчетов.
- `course_assignments` — связь группа + семестр + предмет -> преподаватель; кабинет отсюда не подставляется.

Draft/publish:

- Для `schedule_lessons` есть `status: draft|published|cancelled`, для `course_assignments` — `draft|published`.
- Любые изменения со статусом `draft` не bump’ают `system_state.schedule_version` и не отправляют push.
- Публикация черновиков делает изменения “видимыми” клиентам: bump’ает `schedule_version` и (если включено) шлёт push.

Полезно для дебага:

- `GET /api/v1/admin/schedule/explain?group_id=1&date=2026-02-26&pair_number=2&subgroup=1` — показывает итоговые фактические занятия в слоте.

Учебные планы и календарный учебный график:

- Специальности: `GET/POST/PUT/DELETE /api/v1/admin/specialties`
- Учебные планы (варианты по году набора): `GET/POST/PUT/DELETE /api/v1/admin/curricula`
- Академ. календари (на учебный год): `GET/POST /api/v1/admin/curricula/:id/calendars`, `DELETE /api/v1/admin/calendars/:id`
- Недели календаря: `GET /api/v1/admin/calendars/:id/weeks`, `PUT /api/v1/admin/calendars/:id/weeks`
- Строки учебного плана: `GET/POST /api/v1/admin/curricula/:id/items`, `PUT/DELETE /api/v1/admin/curriculum-items/:id`
- Распределение по семестрам: `GET /api/v1/admin/curriculum-items/:id/allocations`, `PUT /api/v1/admin/curriculum-items/:id/allocations`

Важно: учебный календарь отображает занятость недели группы (практика, экзамены и т.п.) и используется в проверках/подсказках; актуальные пары всё равно хранятся конкретными датами в `schedule_lessons`.

Просмотр структуры БД “Atlas-style” (локально):

- `GET /api/v1/admin/db/schema` — HTML страница с ER-диаграммой (Mermaid) по текущей схеме Postgres. Это read-only интроспекция, доступна только через admin gate.

Полный список и схемы — в `docs/openapi.yaml`.

## Docker deploy

См. пошаговый гайд: [docs/deploy_docker.md](docs/deploy_docker.md).

## Важные правила и ограничения БД

### Нормализация преподавателей

Преподаватели вынесены в таблицу `teachers`:

- `schedule_lessons.teacher_id` → `teachers(id)`

## Ops / эксплуатация

- Prod-стратегия миграций: [docs/ops_migrations.md](docs/ops_migrations.md)
- Backup/restore playbook: [docs/backup_restore.md](docs/backup_restore.md)
- `schedule_overrides.source_teacher_id` / `replacement_teacher_id` → `teachers(id)`

API расписания использует `teacher_id`; создание/поиск преподавателей остается в справочниках и импортах.

### Уникальность и политика конфликтов

- Для `schedule_lessons` действует уникальность активного слота `(group_id, lesson_date, pair_number, subgroup)` при `status <> cancelled`.
- Для `room_assignments` действует уникальность `schedule_lesson_id`.

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
