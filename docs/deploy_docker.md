# Docker deploy

Документ описывает актуальный Docker-flow для backend. В репозитории есть `Dockerfile`, `docker-compose.yml` и entrypoint, который перед стартом API ждет PostgreSQL и применяет Goose migrations.

## 1. Требования

- Docker Engine + Docker Compose plugin
- файл `configs/config.yaml`
- значения секретов через env или `.env`

Важно: перед `docker compose up` файл `configs/config.yaml` должен существовать именно как файл. Если его нет, Docker может создать директорию `configs/config.yaml`, и контейнер упадет с ошибкой монтирования или чтения `/app/config.yaml`.

```bash
cp configs/config.example.yaml configs/config.yaml
```

На Windows:

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
```

## 2. Переменные окружения

`docker-compose.yml` прокидывает:

- `ISPO_CONFIG_PATH=/app/config.yaml`
- `ISPO_DB_PASSWORD`
- `ISPO_AUTH_JWT_SECRET`
- `ISPO_ADMIN_API_KEY`

Пример `.env` рядом с `docker-compose.yml`:

```dotenv
ISPO_DB_PASSWORD=change-me
ISPO_AUTH_JWT_SECRET=replace-with-long-random-secret
ISPO_ADMIN_API_KEY=replace-with-admin-key
```

В `prod` приложение проверяет, что JWT secret достаточно сильный и `admin.api_key` не пустой.

## 3. Запуск полного стека

```bash
docker compose up -d --build
```

Сервисы:

- `postgres` - PostgreSQL, volume `ispo_schedule_pgdata`
- `api` - backend, порт `8080`

Проверка:

```bash
docker compose ps
curl -fsS http://127.0.0.1:8080/api/v1/health
curl -fsS http://127.0.0.1:8080/api/v1/metrics/health
```

## 4. Миграции

В штатном compose-flow миграции применяются entrypoint-ом контейнера API:

```text
goose -dir /app/db/migrations up
```

Для ручного запуска внутри контейнера:

```bash
docker compose exec api goose -dir /app/db/migrations status
docker compose exec api goose -dir /app/db/migrations up
```

Для one-off контейнера:

```bash
docker compose run --rm api goose -dir /app/db/migrations status
docker compose run --rm api goose -dir /app/db/migrations up
```

Перед production-миграциями делайте backup, см. `docs/backup_restore.md`.

## 5. Seed

```bash
docker compose exec api /app/seed
```

Seed создает тестовые группы, преподавателей, кабинеты, учебные планы, календарь, расписание, замены и пользователей.

Dev-аккаунты:

- `admin` / `admin`
- `dispatcher` / `dispatcher`
- `teacher.tuzova` / `teacher`
- `viewer` / `viewer`
- `student1` / `student1`

Пароли можно переопределить через `ISPO_SEED_*`.

## 6. Обновление релиза

Базовый порядок:

1. Получить новую версию кода.
2. Проверить, что `configs/config.yaml` остался файлом.
3. Сделать backup БД.
4. Выполнить:

```bash
docker compose build api
docker compose up -d
```

5. Проверить health endpoints и логи:

```bash
docker compose logs --tail=200 api
curl -fsS http://127.0.0.1:8080/api/v1/health
```

## 7. Частая ошибка с config.yaml

Если контейнер пишет:

```text
read config: read /app/config.yaml: is a directory
```

или Docker сообщает, что нельзя смонтировать директорию в файл, исправьте host path:

```bash
docker compose rm -sf api
rm -rf configs/config.yaml
cp configs/config.example.yaml configs/config.yaml
docker compose up -d api
```

На production вместо example-файла верните реальный `configs/config.yaml`.

## 8. Остановка

Остановить сервисы без удаления данных:

```bash
docker compose down
```

Остановить и удалить volume БД:

```bash
docker compose down --volumes
```

Команду с `--volumes` не используйте на production без явного backup.
