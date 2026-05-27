# Миграции: production strategy

Проект использует Goose и PostgreSQL. Миграции лежат в `db/migrations` и применяются в порядке timestamp-префиксов.

## Где запускаются миграции

В Docker-образ входит бинарь `goose` и каталог `/app/db/migrations`.

В compose-flow миграции запускает entrypoint API перед стартом приложения:

```text
goose -dir /app/db/migrations up
```

Локально:

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = "host=localhost port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable"
goose -dir .\db\migrations status
goose -dir .\db\migrations up
```

## Правила для production

1. Сначала backup, потом миграции.
2. Не править уже примененные migration-файлы.
3. Новое изменение схемы - новый файл миграции.
4. Для рискованных изменений сначала добавлять новые поля/таблицы, потом переводить код, потом удалять старое.
5. Удаление колонок и таблиц делать только после проверки, что код их больше не читает.
6. Constraints и unique indexes добавлять после нормализации данных.
7. Большие индексы на production планировать отдельно; для `CONCURRENTLY` нужен Goose `NO TRANSACTION`.

## Рекомендуемый порядок деплоя

1. Проверить CI.
2. Сделать backup, см. `docs/backup_restore.md`.
3. Собрать новую версию:

```bash
docker compose build api
```

4. Поднять сервис:

```bash
docker compose up -d
```

5. Проверить логи миграций:

```bash
docker compose logs --tail=200 api
```

6. Проверить smoke endpoints:

```bash
curl -fsS http://127.0.0.1:8080/api/v1/health
curl -fsS http://127.0.0.1:8080/api/v1/metrics/health
curl -fsS http://127.0.0.1:8080/api/v1/schedule/version
```

## Проверка миграций на чистой БД

```bash
docker compose down --volumes
docker compose up -d postgres
docker compose run --rm api goose -dir /app/db/migrations up
docker compose run --rm api /app/seed
```

После этого можно поднять API и проверить Swagger/health.

## Откат

Goose поддерживает `down`, но на production откат миграции может быть опасен, если `Down` удаляет данные. Практический план отката:

1. Остановить API.
2. Восстановить БД из backup.
3. Вернуть предыдущий Docker image/code revision.
4. Поднять API.
5. Проверить health и smoke endpoints.

`goose down` допустим только для проверенных миграций, где потеря данных исключена или явно принята.

## Текущие важные таблицы

- `schedule_lessons` - актуальные пары.
- `schedule_overrides` - журнал примененных замен.
- `room_assignments` - кабинет конкретной пары.
- `course_assignments` - назначение преподавателя на дисциплину группы, включая `campus_id` и `is_flow`.
- `calendar_day_constraints` - глобальные ограничения дней.
- `teacher_day_constraints` - ограничения преподавателей.
- `academic_calendar_weeks` и `academic_calendar_day_overrides` - календарь учебного плана.
- `study_calendar_weeks` - календарь группы.
- `students` - историческое имя таблицы пользователей; роли: `student`, `dispatcher`, `admin`, `viewer`, `teacher`.
