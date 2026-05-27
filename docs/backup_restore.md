# Backup и restore PostgreSQL

База проекта по умолчанию называется `ispo_schedule`. Перед production-миграциями и деплоем делайте отдельный backup.

## Backup через pg_dump

Рекомендуемый формат - custom (`-Fc`), потому что он удобен для `pg_restore`.

PowerShell:

```powershell
$ts = Get-Date -Format "yyyyMMdd_HHmmss"
$env:PGPASSWORD = "<password>"
pg_dump -h <host> -p 5432 -U <user> -d ispo_schedule -Fc -f "backup_ispo_schedule_$ts.dump"
```

Bash:

```bash
ts="$(date +%Y%m%d_%H%M%S)"
export PGPASSWORD='<password>'
pg_dump -h <host> -p 5432 -U <user> -d ispo_schedule -Fc -f "backup_ispo_schedule_${ts}.dump"
```

Проверить, что dump читается:

```bash
pg_restore -l backup_ispo_schedule_<timestamp>.dump | head -20
```

## Backup из docker-compose

Если Postgres запущен сервисом `postgres`:

```bash
ts="$(date +%Y%m%d_%H%M%S)"
docker compose exec -T postgres pg_dump -U postgres -d ispo_schedule -Fc > "backup_ispo_schedule_${ts}.dump"
```

На Windows PowerShell:

```powershell
$ts = Get-Date -Format "yyyyMMdd_HHmmss"
docker compose exec -T postgres pg_dump -U postgres -d ispo_schedule -Fc | Set-Content -Encoding Byte "backup_ispo_schedule_$ts.dump"
```

Если PowerShell портит бинарный поток, используйте `cmd.exe`:

```cmd
docker compose exec -T postgres pg_dump -U postgres -d ispo_schedule -Fc > backup_ispo_schedule.dump
```

## Restore в новую БД

Безопасный способ проверки backup:

```bash
export PGPASSWORD='<password>'
createdb -h <host> -p 5432 -U <user> ispo_schedule_restore
pg_restore -h <host> -p 5432 -U <user> -d ispo_schedule_restore -Fc backup_ispo_schedule.dump
```

После restore можно запустить приложение на копии БД и проверить health/smoke.

## Restore поверх существующей БД

Используйте только при осознанном откате.

1. Остановить API.
2. Убедиться, что нет активных подключений.
3. Восстановить dump:

```bash
export PGPASSWORD='<password>'
pg_restore -h <host> -p 5432 -U <user> -d ispo_schedule --clean --if-exists -Fc backup_ispo_schedule.dump
```

Для docker-compose:

```bash
docker compose stop api
docker compose exec -T postgres pg_restore -U postgres -d ispo_schedule --clean --if-exists -Fc < backup_ispo_schedule.dump
docker compose up -d api
```

## Smoke после restore

```bash
curl -fsS http://127.0.0.1:8080/api/v1/health
curl -fsS http://127.0.0.1:8080/api/v1/metrics/health
curl -fsS http://127.0.0.1:8080/api/v1/schedule/version
```

Проверить миграции:

```bash
docker compose exec api goose -dir /app/db/migrations status
```

## Хранение backup

- Минимум: ежедневный backup и хранение 7-14 дней.
- Перед каждой миграцией: отдельный backup с пометкой версии.
- Хранить хотя бы одну копию вне VDS.
- Периодически проверять restore на отдельной базе.
