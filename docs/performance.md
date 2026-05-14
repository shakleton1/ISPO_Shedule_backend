# Производительность

Этот проект собирает ответы расписания из фактических занятий на конкретные даты:

- занятий (`schedule_lessons`)
- кабинетов занятий (`room_assignments`)
- applied-журнала замен (`schedule_overrides`) для отчетов, не для merge рендера
- дневных оверлеев/событий

## 1) Кэш недели (на стороне сервера)

Backend держит in-memory кэш для **канонических недель Пн..Сб**, ключ:

- `group_id`
- `week_start` (дата понедельника, `YYYY-MM-DD`)
- `data_version` (из `system_state.schedule_version`)

Если `data_version` не менялся, повторные запросы:

- `GET /api/v1/schedule/current`
- `GET /api/v1/schedule/range` (когда `date_start` = понедельник, а `date_end` = суббота)
- `GET /api/v1/schedule/pdf` (внутри строит 2 недельных диапазона)

будут переиспользовать закэшированный ответ недели.

## 2) Как ловить N+1 в dev

Можно включить SQL-логи GORM через env:

```powershell
$env:ISPO_DB_LOG_LEVEL = "info"
go run .\cmd\api
```

Это помогает увидеть повторяющиеся похожие запросы в рамках одного HTTP-запроса (типичный симптом N+1).

Допустимые значения:

- `silent`, `error`, `warn` (по умолчанию), `info`

## 3) EXPLAIN / EXPLAIN ANALYZE (Postgres)

Для критичных schedule-эндпоинтов самые “горячие” запросы обычно — `schedule_lessons` + `room_assignments`.

### 3.1) Запрос фактических занятий

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT sl.lesson_date, sl.pair_number, sl.subject_id, s.name AS subject_name,
       ra.location_id, l.name AS location_name, COALESCE(t.name, '') AS teacher_name,
       sl.subgroup, sl.lesson_format, sl.status, sl.version
FROM schedule_lessons sl
LEFT JOIN subjects s ON s.id = sl.subject_id
LEFT JOIN room_assignments ra ON ra.schedule_lesson_id = sl.id AND ra.status = 'published'
LEFT JOIN locations l ON l.id = ra.location_id
LEFT JOIN teachers t ON t.id = sl.teacher_id
WHERE sl.group_id = 1
  AND sl.lesson_date BETWEEN '2026-02-23' AND '2026-02-28'
  AND sl.status <> 'cancelled'
ORDER BY sl.lesson_date, sl.pair_number, COALESCE(sl.subgroup, 0), sl.id;
```

### 3.2) Журнал замен

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT so.lesson_date, so.id, so.pair_number, so.action_type,
       so.source_subject_id, so.replacement_subject_id,
       so.source_location_id, so.replacement_location_id,
       so.source_teacher_id, so.replacement_teacher_id,
       so.reason, so.status, so.applied_at
FROM schedule_overrides so
WHERE so.group_id = 1
  AND so.lesson_date BETWEEN '2026-02-23' AND '2026-02-28'
ORDER BY so.lesson_date, so.pair_number, COALESCE(so.subgroup, 0), so.id;
```

Если в этих EXPLAIN видно последовательное сканирование (seq scan) по большим таблицам — добавляйте/правьте индексы (лучше через миграции).
