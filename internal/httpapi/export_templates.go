package httpapi

const twoWeekScheduleExportHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4 landscape; margin: 8mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #1f2933; font-family: Arial, sans-serif; font-size: 8.5pt; }
    .page { page-break-after: always; }
    .page:last-child { page-break-after: auto; }
    .header { display: flex; justify-content: space-between; align-items: flex-start; gap: 8mm; margin-bottom: 5mm; border-bottom: 1.4pt solid #2f4f5f; padding-bottom: 3mm; }
    .title { font-size: 17pt; font-weight: 700; color: #193847; }
    .subtitle { margin-top: 1.5mm; font-size: 9.5pt; color: #52616b; }
    .meta { text-align: right; color: #52616b; line-height: 1.4; white-space: nowrap; }
    .week-title { font-size: 12pt; font-weight: 700; color: #193847; margin: 0 0 3mm; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th { background: #203f4c; color: #fff; border: 1px solid #203f4c; padding: 2mm 1mm; text-align: center; font-size: 8pt; }
    td { border: 1px solid #b7c3c9; vertical-align: top; height: 26mm; padding: 1.3mm; }
    .day-col { width: 25mm; background: #edf3f5; font-weight: 700; color: #203f4c; text-align: left; }
    .day-main { font-size: 9pt; }
    .day-date { font-size: 8pt; color: #52616b; margin-top: 1mm; }
    .day-note { margin-top: 1.5mm; padding: 1mm; border-left: 2px solid #b7791f; background: #fff8e6; color: #5f4700; font-size: 7.2pt; font-weight: 400; }
    .lesson { border-left: 2.2px solid #2f6f7e; background: #f8fbfc; padding: 1mm; margin-bottom: 1mm; min-height: 14mm; }
    .lesson.changed { border-left-color: #b45309; background: #fff7ed; }
    .lesson.added { border-left-color: #2563eb; background: #eff6ff; }
    .subject { font-weight: 700; font-size: 7.8pt; line-height: 1.15; }
    .line { margin-top: .7mm; color: #334e5c; line-height: 1.15; }
    .room { margin-top: .8mm; font-weight: 700; color: #193847; }
    .badge { display: inline-block; margin-top: .8mm; padding: .2mm 1mm; border-radius: 1.5mm; background: #e8eef1; color: #203f4c; font-size: 6.8pt; font-weight: 700; }
    .empty { color: #9aa6ac; text-align: center; padding-top: 9mm; }
  </style>
</head>
<body>
  {{ range .Weeks }}
  <section class="page">
    <div class="header">
      <div>
        <div class="title">{{ $.Title }}</div>
        <div class="subtitle">{{ $.Subtitle }}</div>
      </div>
      <div class="meta">
        <div>{{ .Title }}</div>
        <div>{{ $.GeneratedAt }}</div>
      </div>
    </div>
    <div class="week-title">{{ .RangeLabel }}</div>
    <table>
      <thead>
        <tr>
          <th class="day-col">День</th>
          {{ range $.Pairs }}<th>{{ . }} пара</th>{{ end }}
        </tr>
      </thead>
      <tbody>
        {{ range .Days }}
        <tr>
          <td class="day-col">
            <div class="day-main">{{ .DayName }}</div>
            <div class="day-date">{{ .DateLabel }}</div>
            {{ if .Note }}<div class="day-note">{{ .Note }}</div>{{ end }}
          </td>
          {{ range .Cells }}
          <td>
            {{ if .Lessons }}
              {{ range .Lessons }}
              <div class="lesson {{ if .IsChanged }}changed{{ end }} {{ if .IsAdded }}added{{ end }}">
                <div class="subject">{{ .Subject }}</div>
                {{ if .Primary }}<div class="line">{{ .Primary }}</div>{{ end }}
                {{ if .Secondary }}<div class="line">{{ .Secondary }}</div>{{ end }}
                {{ if .Location }}<div class="room">{{ .Location }}</div>{{ end }}
                {{ if .Badge }}<div class="badge">{{ .Badge }}</div>{{ end }}
                {{ if .Comment }}<div class="line">{{ .Comment }}</div>{{ end }}
              </div>
              {{ end }}
            {{ else }}
              <div class="empty">-</div>
            {{ end }}
          </td>
          {{ end }}
        </tr>
        {{ end }}
      </tbody>
    </table>
  </section>
  {{ end }}
</body>
</html>`

const scheduleOverridesExportHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4 landscape; margin: 8mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #1f2933; font-family: Arial, sans-serif; font-size: 8pt; }
    .header { display: flex; justify-content: space-between; gap: 8mm; margin-bottom: 5mm; border-bottom: 1.4pt solid #2f4f5f; padding-bottom: 3mm; }
    .title { font-size: 17pt; font-weight: 700; color: #193847; }
    .subtitle { margin-top: 1.5mm; font-size: 9.5pt; color: #52616b; }
    .meta { text-align: right; color: #52616b; line-height: 1.4; white-space: nowrap; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th { background: #203f4c; color: #fff; border: 1px solid #203f4c; padding: 1.6mm 1mm; text-align: left; font-size: 7.4pt; }
    td { border: 1px solid #b7c3c9; vertical-align: top; padding: 1.3mm; line-height: 1.25; }
    tr:nth-child(even) td { background: #f7fafb; }
    .date { width: 21mm; }
    .pair { width: 15mm; text-align: center; }
    .group { width: 24mm; }
    .action { width: 22mm; font-weight: 700; color: #193847; }
    .before, .after { width: 52mm; }
    .reason { width: 42mm; }
    .muted { color: #667985; }
    .empty { color: #9aa6ac; }
  </style>
</head>
<body>
  <div class="header">
    <div>
      <div class="title">{{ .Title }}</div>
      <div class="subtitle">{{ .Subtitle }}</div>
    </div>
    <div class="meta">
      <div>Записей: {{ .RowsCount }}</div>
      <div>{{ .GeneratedAt }}</div>
    </div>
  </div>
  <table>
    <thead>
      <tr>
        <th class="date">Дата</th>
        <th class="pair">Пара</th>
        <th class="group">Группа</th>
        <th class="action">Операция</th>
        <th class="before">Было</th>
        <th class="after">Стало</th>
        <th class="reason">Причина</th>
      </tr>
    </thead>
    <tbody>
      {{ range .Rows }}
      <tr>
        <td>{{ .Date }}</td>
        <td class="pair">{{ .Pair }}</td>
        <td>{{ .GroupName }}</td>
        <td class="action">{{ .ActionLabel }}</td>
        <td>{{ if .SourceText }}{{ .SourceText }}{{ else }}<span class="empty">-</span>{{ end }}</td>
        <td>{{ if .ReplacementText }}{{ .ReplacementText }}{{ else }}<span class="empty">-</span>{{ end }}</td>
        <td>{{ if .Reason }}{{ .Reason }}{{ else }}<span class="muted">Без причины</span>{{ end }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>
</body>
</html>`
