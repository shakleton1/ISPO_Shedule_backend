# ISPO Smart Schedule Backend

Backend для системы расписания ИСПО. Сервис хранит фактические занятия на конкретные даты, назначения преподавателей, кабинеты, учебные планы, учебные календари, ограничения дней, замены и печатные отчеты.

## Текущая архитектура

Актуальная модель расписания:

- `schedule_lessons` - единственный источник актуального расписания на конкретные даты.
- `room_assignments` - кабинет конкретной пары через `schedule_lesson_id`.
- `schedule_overrides` - журнал примененных операций `add`, `replace`, `cancel`, `restore` со snapshot "что было" и "что стало".
- `course_assignments` - назначение `группа + семестр + дисциплина -> преподаватель`, также хранит `campus_id` и флаг `is_flow`; кабинет отсюда не подставляется.
- `teacher_subjects` - какие дисциплины может вести преподаватель.
- `teacher_location_preferences`, `room_requests`, `location_week_availability` - предпочтения кабинетов, заявки на тип помещения и недельная доступность кабинетов.
- `academic_calendar_weeks`, `academic_calendar_day_overrides`, `study_calendar_weeks` - учебный календарь по учебному плану, дневные уточнения и календарь группы.
- `calendar_day_constraints` - глобальные ограничения конкретных дат для всех групп.
- `teacher_day_constraints` - ограничения преподавателей на конкретные даты.

Старые `schedule_templates` и `calendar_exceptions` не являются основой актуального расписания.

## Стек

- Go `1.25.10`
- Gin, GORM, PostgreSQL
- Goose migrations
- Viper config через YAML/env
- PDF через Chromium/chromedp
- XLSX через excelize
- Push через Firebase Cloud Messaging
- Prometheus `/metrics`
- OpenAPI: `docs/openapi.yaml`

## Быстрый старт

### 1. Конфиг

Перед `docker compose up` обязательно создайте файл `configs/config.yaml`. Если файла нет, Docker может создать директорию с таким именем, и контейнер не стартует.

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
```

Основные параметры:

- `env`: `dev`, `prod`, `test`
- `server.addr`
- `db.*`
- `schedule.semester_start_date`
- `auth.jwt_secret`
- `auth.bootstrap_admin_login`, `auth.bootstrap_admin_password`
- `admin.api_key`
- `push.*`
- `pdf.chrome_executable_path`

В production guardrails требуют сильный `auth.jwt_secret` и непустой `admin.api_key`.

### 2. Запуск через Docker Compose

```powershell
docker compose up -d --build
```

Контейнер API сам ждет PostgreSQL и применяет Goose migrations перед стартом.

### 3. Seed

```powershell
docker compose exec api /app/seed
```

Или локально:

```powershell
go run .\cmd\seed
```

Dev-аккаунты по умолчанию:

- `admin` / `admin`
- `dispatcher` / `dispatcher`
- `teacher.tuzova` / `teacher`
- `viewer` / `viewer`
- `student1` / `student1`

Пароли можно переопределить env-переменными:

- `ISPO_SEED_ADMIN_PASSWORD`
- `ISPO_SEED_DISPATCHER_PASSWORD`
- `ISPO_SEED_TEACHER_PASSWORD`
- `ISPO_SEED_VIEWER_PASSWORD`
- `ISPO_SEED_STUDENT_PASSWORD`

### 4. Локальный запуск без API-контейнера

```powershell
docker compose up -d postgres
go install github.com/pressly/goose/v3/cmd/goose@latest
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = "host=localhost port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable"
goose -dir .\db\migrations up
go run .\cmd\seed
go run .\cmd\api
```

### 5. Проверка

- Health: `GET http://127.0.0.1:8080/api/v1/health`
- DB health: `GET http://127.0.0.1:8080/api/v1/metrics/health`
- OpenAPI YAML: `GET http://127.0.0.1:8080/openapi.yaml`
- Swagger UI: `GET http://127.0.0.1:8080/docs`

## Роли и доступ

Auth:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`

Роли:

- `admin` - полный доступ.
- `dispatcher` - чтение админки, редактирование справочников, расписания, кабинетов, учебных планов, календарей и импортов.
- `teacher` - личные endpoints преподавателя по привязанному `teacher_id`.
- `viewer` - чтение админских списков.
- `student` - клиентская роль для расписания группы.

Admin gate принимает JWT или `X-Admin-Key`, если `admin.api_key` задан в конфиге.

## Основные endpoint-ы

Публичное расписание:

- `GET /api/v1/schedule/current?group_id=...&date=YYYY-MM-DD`
- `GET /api/v1/schedule/range?group_id=...&date_start=YYYY-MM-DD&date_end=YYYY-MM-DD`
- `GET /api/v1/schedule/version`
- `GET /api/v1/schedule/pdf?group_id=...&date_start=YYYY-MM-DD`
- `GET /api/v1/schedule/xlsx?group_id=...&date_start=YYYY-MM-DD`

Личный кабинет преподавателя:

- `GET /api/v1/teacher/schedule?date_start=YYYY-MM-DD&date_end=YYYY-MM-DD`
- `GET /api/v1/teacher/workload?date_start=YYYY-MM-DD&date_end=YYYY-MM-DD`

Публичные справочники:

- `GET /api/v1/groups`
- `GET /api/v1/subjects`
- `GET /api/v1/locations`
- `GET /api/v1/campuses`
- `GET /api/v1/location-types`

Администрирование:

- группы, дисциплины, преподаватели, корпуса, кабинеты, типы помещений
- связи преподавателей и дисциплин
- назначения преподавателей на группы
- учебные планы, элементы учебного плана, распределение часов
- академические календари, недели и дневные уточнения
- фактические занятия `schedule_lessons`
- замены через `POST /api/v1/admin/schedule-overrides/apply`
- ограничения дней и преподавателей
- заявки/назначения кабинетов и автозаполнение кабинетов
- отчеты PDF/XLSX по группе, преподавателям и заменам

Полный контракт находится в `docs/openapi.yaml`.

## Импорт

Поддерживаются admin endpoints:

- `POST /api/v1/admin/import/curriculum-items/csv`
- `POST /api/v1/admin/import/curriculum-items/xlsx`
- `POST /api/v1/admin/import/plx-curriculum/xlsx`
- `POST /api/v1/admin/import/study-calendar/csv`
- `POST /api/v1/admin/import/study-calendar/xlsx`

PLX/XLSX импорт учебного плана также разбирает график учебного процесса по курсам.

## Отчеты

PDF/XLSX:

- `GET /api/v1/admin/reports/group-schedule/{pdf,xlsx}`
- `GET /api/v1/admin/reports/teacher-schedule/{pdf,xlsx}`
- `GET /api/v1/admin/reports/teacher-schedules/{pdf,xlsx}`
- `GET /api/v1/admin/reports/schedule-overrides/{pdf,xlsx}`

Групповой отчет строится на две недели на одном листе. Общий отчет преподавателей рассчитан на печать по три преподавателя на лист.

## ER-диаграммы

Есть runtime ER-view:

- `GET /api/v1/admin/db/schema`

Физическая ER-документация:

- `docs/erd/schema_erd.json`
- `docs/erd/generate_erd.py`
- `docs/erd/2_5a_schedule_replacements.svg`
- `docs/erd/2_5b_rooms_constraints.svg`
- `docs/erd/2_5c_curricula_users.svg`

Для регенерации нужен Graphviz (`dot` в `PATH`):

```powershell
python .\docs\erd\generate_erd.py
```

## Полезные команды

PowerShell scripts:

- `powershell -File .\scripts\db-up.ps1`
- `powershell -File .\scripts\migrate-up.ps1`
- `powershell -File .\scripts\seed.ps1`
- `powershell -File .\scripts\run-api.ps1`
- `powershell -File .\scripts\test.ps1`
- `powershell -File .\scripts\lint.ps1`

Taskfile:

- `task db:up`
- `task migrate:up`
- `task seed`
- `task run`
- `task test`
- `task lint`

Проверки:

```powershell
go test ./...
python -m openapi_spec_validator docs/openapi.yaml
```

## Дополнительная документация

- Docker deploy: `docs/deploy_docker.md`
- Миграции: `docs/ops_migrations.md`
- Backup/restore: `docs/backup_restore.md`
- Производительность: `docs/performance.md`
