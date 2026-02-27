# Playbook бэкап/восстановление (Postgres)

Ниже — практический минимум для бэкапа и восстановления базы `ispo_schedule`.

## Бэкап (pg_dump)

Рекомендуемый формат — custom (`-Fc`), чтобы можно было выборочно восстанавливать.

```powershell
$ts = Get-Date -Format "yyyyMMdd_HHmmss"
$env:PGPASSWORD = "<password>"
pg_dump -h <host> -p 5432 -U <user> -d ispo_schedule -Fc -f "backup_ispo_schedule_$ts.dump"
```

Проверка, что файл читается:

```powershell
pg_restore -l "backup_ispo_schedule_$ts.dump" | Select-Object -First 20
```

## Восстановление (pg_restore)

Вариант A (полное восстановление в новую БД):

```powershell
$env:PGPASSWORD = "<password>"
createdb -h <host> -p 5432 -U <user> ispo_schedule_restore
pg_restore -h <host> -p 5432 -U <user> -d ispo_schedule_restore -Fc "backup_ispo_schedule_$ts.dump"
```

Вариант B (в существующую базу, перезаливка):

1. Остановить сервис.
2. Убедиться, что нет активных коннектов.
3. Восстановить:

```powershell
$env:PGPASSWORD = "<password>"
pg_restore -h <host> -p 5432 -U <user> -d ispo_schedule --clean --if-exists -Fc "backup_ispo_schedule_$ts.dump"
```

## Частота и хранение

- Минимум: ежедневный бэкап + хранение 7–14 дней.
- Перед каждым деплоем миграций — отдельный бэкап.

## Smoke-проверка после восстановления

- Прогнать миграции `goose up` (если restore старее схемы).
- Проверить:
  - `GET /api/v1/health`
  - `GET /api/v1/metrics/health`
  - `GET /api/v1/schedule/version`
