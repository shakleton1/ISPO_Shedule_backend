# Производительность

Этот проект собирает ответы расписания, комбинируя данные из:

- шаблонов (`schedule_templates`)
- overrides/оверрайдов (`schedule_overrides`)
- переносов дней (`calendar_exceptions`)
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

Для критичных schedule-эндпоинтов самые “горячие” запросы обычно — templates/overrides.

### 3.1) Запрос шаблонов

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT st.day_of_week, st.pair_number, st.subject_id, s.name AS subject_name,
       st.location_id, l.name AS location_name, COALESCE(t.name, '') AS teacher_name,
       st.teacher_manual, st.location_manual, st.subgroup
FROM schedule_templates st
JOIN subjects s ON s.id = st.subject_id
JOIN locations l ON l.id = st.location_id
LEFT JOIN teachers t ON t.id = st.teacher_id
WHERE st.group_id = 1
  AND st.day_of_week BETWEEN 0 AND 5
  AND st.week_parity IN ('numerator', 'both')
  AND st.status = 'published'
ORDER BY st.day_of_week, st.pair_number, COALESCE(st.subgroup, 0), st.id;
```

### 3.2) Запрос overrides

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT so.target_date, so.id, so.pair_number, so.action_type,
       so.new_subject_id, COALESCE(s.name, '') AS new_subject_name,
       so.new_location_id, COALESCE(l.name, '') AS new_location_name,
       so.new_teacher_manual, t.name AS new_teacher_name,
       so.comment, so.subgroup, so.updated_at
FROM schedule_overrides so
LEFT JOIN subjects s ON s.id = so.new_subject_id
LEFT JOIN locations l ON l.id = so.new_location_id
LEFT JOIN teachers t ON t.id = so.new_teacher_id
WHERE so.group_id = 1
  AND so.target_date BETWEEN '2026-02-23' AND '2026-02-28'
ORDER BY so.target_date, so.pair_number, COALESCE(so.subgroup, 0), so.id;
```

Если в этих EXPLAIN видно последовательное сканирование (seq scan) по большим таблицам — добавляйте/правьте индексы (лучше через миграции).
