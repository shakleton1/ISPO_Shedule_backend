# Производительность

Расписание строится из фактических занятий на конкретные даты. Основной read-path:

1. `schedule_lessons`
2. `room_assignments`
3. `subjects`, `teachers`, `locations`
4. учебный календарь и дневные ограничения
5. in-memory cache для канонических недель Пн-Сб

`schedule_overrides` не участвует в merge при отображении расписания: замена уже применена к `schedule_lessons`, а `schedule_overrides` используется для истории и отчетов.

## Week cache

Сервис кеширует ответы для канонической недели:

- группа
- понедельник недели
- `system_state.schedule_version`

Кеш используется для:

- `GET /api/v1/schedule/current`
- `GET /api/v1/schedule/range`, если диапазон ровно Пн-Сб
- PDF/XLSX экспорта, когда внутри запрашиваются недельные диапазоны

Любое опубликованное изменение расписания bump-ает `system_state.schedule_version`, поэтому старый cache key перестает использоваться.

## Индексы, важные для расписания

Критичные запросы опираются на:

- `schedule_lessons(group_id, lesson_date)`
- `schedule_lessons(teacher_id, lesson_date)`
- unique active slot `(group_id, lesson_date, pair_number, COALESCE(subgroup, 0)) WHERE status <> 'cancelled'`
- `room_assignments(schedule_lesson_id)`
- `schedule_overrides(group_id, lesson_date)`
- `teacher_day_constraints(teacher_id, target_date)`
- `calendar_day_constraints(target_date)`
- календарные индексы по `calendar_id`, `course_number`, `week_number`

Новые индексы добавлять миграциями.

## Как смотреть SQL в dev

```powershell
$env:ISPO_DB_LOG_LEVEL = "info"
go run .\cmd\api
```

Допустимые значения:

- `silent`
- `error`
- `warn`
- `info`

`info` помогает найти повторяющиеся запросы в одном HTTP-request.

## EXPLAIN для расписания группы

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT sl.id,
       sl.group_id,
       sl.lesson_date,
       sl.pair_number,
       sl.subgroup,
       sl.subject_id,
       s.name AS subject_name,
       sl.teacher_id,
       t.name AS teacher_name,
       ra.location_id,
       l.name AS location_name,
       sl.lesson_format,
       sl.status,
       sl.source,
       sl.flow_key,
       sl.version
FROM schedule_lessons sl
LEFT JOIN subjects s ON s.id = sl.subject_id
LEFT JOIN teachers t ON t.id = sl.teacher_id
LEFT JOIN room_assignments ra
  ON ra.schedule_lesson_id = sl.id
 AND ra.status = 'published'
LEFT JOIN locations l ON l.id = ra.location_id
WHERE sl.group_id = 1
  AND sl.lesson_date BETWEEN DATE '2026-03-23' AND DATE '2026-04-05'
  AND sl.status <> 'cancelled'
ORDER BY sl.lesson_date,
         sl.pair_number,
         COALESCE(sl.subgroup, 0),
         sl.id;
```

## EXPLAIN для кабинета

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT sl.lesson_date,
       sl.pair_number,
       sl.group_id,
       sl.subject_id,
       sl.teacher_id,
       ra.location_id
FROM schedule_lessons sl
JOIN room_assignments ra ON ra.schedule_lesson_id = sl.id
WHERE ra.location_id = 10
  AND ra.status = 'published'
  AND sl.status <> 'cancelled'
  AND sl.lesson_date BETWEEN DATE '2026-03-23' AND DATE '2026-04-05'
ORDER BY sl.lesson_date, sl.pair_number, sl.group_id;
```

## EXPLAIN для журнала замен

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT so.lesson_date,
       so.pair_number,
       so.action_type,
       so.source_subject_id,
       so.replacement_subject_id,
       so.source_teacher_id,
       so.replacement_teacher_id,
       so.source_location_id,
       so.replacement_location_id,
       so.status,
       so.applied_at
FROM schedule_overrides so
WHERE so.group_id = 1
  AND so.lesson_date BETWEEN DATE '2026-03-23' AND DATE '2026-04-05'
ORDER BY so.lesson_date,
         so.pair_number,
         COALESCE(so.subgroup, 0),
         so.id;
```

## Практические правила

- Не читать `schedule_overrides` для построения текущего расписания.
- Для массовых отчетов преподавателей использовать `scope=teacher` и ограничивать диапазон дат.
- Для потоков использовать `flow_key`; кабинетный конфликт допустим только при общем потоке.
- Не строить расписание через `course_assignments`: это справочник назначений, а не фактическая сетка занятий.
- При проблемах с PDF сначала проверять HTML/XLSX-данные, затем Chromium/chromedp.
- Для больших импортов держать `server.admin_import_max_body_bytes` и rate limit в конфиге согласованными с ожидаемым размером файлов.
