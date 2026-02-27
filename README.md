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
- `dispatcher` — изменение расписания (шаблоны/оверрайды/оверлеи/исключения) без редактирования справочников
- `viewer` — только чтение админских списков (шаблоны/оверрайды/исключения/справочники)

Операции (основное):

- Импорт шаблонов: `POST /api/v1/admin/import/templates/csv|xlsx`
- Шаблоны: `GET/POST/PUT/DELETE /api/v1/admin/templates`
- Оверрайды: `GET /api/v1/admin/overrides`, `POST /api/v1/admin/override`, `PUT/DELETE /api/v1/admin/overrides/:id`
- Массовые оверрайды (на период): `POST /api/v1/admin/overrides/bulk`
- Перенос пары (атомарно `CANCEL` + `ADD`): `POST /api/v1/admin/override/move`
- Оверлей дня: `POST /api/v1/admin/overlay`
- События дня (структурированные, не только текст): `GET/POST /api/v1/admin/day-events`, `PUT/DELETE /api/v1/admin/day-events/:id`
- Calendar exceptions: `GET/POST /api/v1/admin/calendar-exceptions`, `DELETE /api/v1/admin/calendar-exceptions/:date`

Жизненный цикл и “источник истины” (policy):

- `schedule_templates` (status=`published`) — базовый источник расписания по неделе.
- `course_assignments` (status=`published`) — вспомогательный источник: используется для автозаполнения преподавателя (и далее аудитории) если в шаблоне поле пустое.
- `schedule_overrides` — самый высокий приоритет на конкретную дату (CANCEL/REPLACE/ADD) и перекрывает шаблоны/автозаполнение.
- Импорт — это способ записать шаблоны (в published или draft), но не отдельный “источник истины”.

Draft/publish:

- Для `schedule_templates` и `course_assignments` есть `status: draft|published`.
- Любые изменения со статусом `draft` не bump’ают `system_state.schedule_version` и не отправляют push.
- Публикация черновиков делает изменения “видимыми” клиентам: bump’ает `schedule_version` и (если включено) шлёт push.

Полезно для дебага:

- `GET /api/v1/admin/schedule/explain?group_id=1&date=2026-02-26&pair_number=2&subgroup=1` — показывает, какие шаблоны/оверрайды/авторезолв участвовали в результате.

Учебные планы и календарный учебный график:

- Специальности: `GET/POST/PUT/DELETE /api/v1/admin/specialties`
- Учебные планы (варианты по году набора): `GET/POST/PUT/DELETE /api/v1/admin/curricula`
- Академ. календари (на учебный год): `GET/POST /api/v1/admin/curricula/:id/calendars`, `DELETE /api/v1/admin/calendars/:id`
- Недели календаря: `GET /api/v1/admin/calendars/:id/weeks`, `PUT /api/v1/admin/calendars/:id/weeks`
- Строки учебного плана: `GET/POST /api/v1/admin/curricula/:id/items`, `PUT/DELETE /api/v1/admin/curriculum-items/:id`
- Распределение по семестрам: `GET /api/v1/admin/curriculum-items/:id/allocations`, `PUT /api/v1/admin/curriculum-items/:id/allocations`

Важно: если группа привязана к `curriculum_id` и в академ. календаре неделя помечена как `is_teaching=false`, то шаблоны на эти даты не применяются (каникулы/практики и т.п.), но оверрайды по датам всё равно можно задавать вручную.

Просмотр структуры БД “Atlas-style” (локально):

- `GET /api/v1/admin/db/schema` — HTML страница с ER-диаграммой (Mermaid) по текущей схеме Postgres. Это read-only интроспекция, доступна только через admin gate.

Полный список и схемы — в `docs/openapi.yaml`.

## Docker deploy

См. пошаговый гайд: [docs/deploy_docker.md](docs/deploy_docker.md).

## Импорт шаблонов CSV/XLSX

Импорт работает в режиме "заменить всё для группы" в рамках выбранного статуса:

- сначала удаляются все `schedule_templates` для `group_id` и `status`
- затем вставляются строки из файла
- если импорт в `status=published`: после успеха увеличивается `system_state.schedule_version` и (если включено) отправляется push
- если импорт в `status=draft`: версия не меняется и push не отправляется

Параметры:

- `group_id` — обязателен
- `status` — опционально, `published` (по умолчанию) или `draft`

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

## Ops / эксплуатация

- Prod-стратегия миграций: [docs/ops_migrations.md](docs/ops_migrations.md)
- Backup/restore playbook: [docs/backup_restore.md](docs/backup_restore.md)
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
