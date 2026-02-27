# Performance notes

This project builds schedule responses by combining:

- templates (`schedule_templates`)
- overrides (`schedule_overrides`)
- calendar exceptions (`calendar_exceptions`)
- day overlays/events

## 1) Week cache (server-side)

The backend keeps an in-memory cache for **canonical Mon..Sat weeks**, keyed by:

- `group_id`
- `week_start` (Monday date, `YYYY-MM-DD`)
- `data_version` (from `system_state.schedule_version`)

So if `data_version` is unchanged, repeated calls to:

- `GET /api/v1/schedule/current`
- `GET /api/v1/schedule/range` (when `date_start` is Monday and `date_end` is Saturday)
- `GET /api/v1/schedule/pdf` (internally builds two week ranges)

will reuse the cached week response.

## 2) Detecting N+1 in dev

You can enable SQL logs from GORM using an env var:

```powershell
$env:ISPO_DB_LOG_LEVEL = "info"
go run .\cmd\api
```

This helps you spot repeated similar queries per request (a typical N+1 symptom).

Valid values:

- `silent`, `error`, `warn` (default), `info`

## 3) EXPLAIN / EXPLAIN ANALYZE (Postgres)

For critical schedule endpoints, the hottest DB queries are usually templates/overrides.

### 3.1) Templates query

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

### 3.2) Overrides query

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

If these EXPLAINs show sequential scans on big tables, add or adjust indexes (preferably via migrations).
