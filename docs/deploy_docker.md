# Docker deploy

Этот проект деплоится через Docker. Ниже — минимальный production-friendly flow: билд образа, миграции отдельным одноразовым контейнером, запуск API с примонтированным `config.yaml`.

## 1) Собрать образ

```bash
docker build -t ispo-schedule-api:latest .
```

## 2) Конфиг (config.yaml)

Контейнер ожидает конфиг-файл по пути `/app/config.yaml` (путь задаётся через `ISPO_CONFIG_PATH`, он уже выставлен в `Dockerfile`).

Рекомендуемый подход:

- хранить **не-секретные** значения в файле `configs/config.yaml`
- секреты (пароли/ключи) прокидывать через env (Viper: префикс `ISPO_`, ключи вида `ISPO_DB_PASSWORD`, `ISPO_AUTH_JWT_SECRET` и т.п.)

Минимальный пример (адаптируйте под окружение):

```yaml
server:
  addr: 0.0.0.0:8080

db:
  host: postgres
  port: 5432
  user: postgres
  password: postgres
  name: ispo_schedule
  sslmode: disable

auth:
  jwt_secret: "change-me"

pdf:
  # В Docker по умолчанию выставлено в /usr/bin/chromium через ENV,
  # но можно продублировать и здесь.
  chrome_executable_path: "/usr/bin/chromium"
```

## 3) Миграции (одноразовый контейнер)

Миграции лучше запускать отдельно (job/one-off container) **до** старта API.

Образ содержит `goose` и папку миграций `/app/db/migrations`, поэтому миграции можно применить так:

```bash
# Пример строки подключения (подставьте свои значения)
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING='host=postgres port=5432 user=postgres password=postgres dbname=ispo_schedule sslmode=disable'

docker run --rm \
  --network <your_network> \
  -e GOOSE_DRIVER \
  -e GOOSE_DBSTRING \
  ispo-schedule-api:latest \
  goose -dir /app/db/migrations up
```

Где `<your_network>` — сеть, в которой доступен Postgres (например, сеть docker-compose).

## 4) Запуск API

```bash
docker run --rm \
  --name ispo-api \
  --network <your_network> \
  -p 8080:8080 \
  -v $(pwd)/configs/config.yaml:/app/config.yaml:ro \
  -e ISPO_DB_PASSWORD=postgres \
  -e ISPO_AUTH_JWT_SECRET=change-me \
  ispo-schedule-api:latest
```

## 5) Health / readiness

- Liveness: `GET /api/v1/health` (используется `HEALTHCHECK` в образе)
- Readiness (DB): `GET /api/v1/metrics/health` — вернёт `503` если Postgres недоступен
