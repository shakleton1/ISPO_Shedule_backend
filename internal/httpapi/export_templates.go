package httpapi

const twoWeekScheduleExportHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4 landscape; margin: 6mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #111; font-family: Arial, sans-serif; font-size: 6.2pt; }
    .sheet { width: 285mm; min-height: 198mm; }
    .header { display: flex; justify-content: space-between; align-items: flex-start; gap: 6mm; margin-bottom: 3mm; border-bottom: 1px solid #111; padding-bottom: 2mm; }
    .title { font-size: 13pt; font-weight: 700; color: #111; }
    .subtitle { margin-top: 1mm; font-size: 8pt; color: #333; }
    .meta { text-align: right; color: #333; line-height: 1.25; white-space: nowrap; font-size: 7pt; }
    .weeks { display: grid; grid-template-columns: 1fr 1fr; gap: 4mm; }
    .week-title { font-size: 9pt; font-weight: 700; color: #111; margin: 0 0 1.5mm; display: flex; justify-content: space-between; border-bottom: 1px solid #777; padding-bottom: .8mm; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th { border: 1px solid #222; padding: 1mm .5mm; text-align: center; font-size: 5.8pt; font-weight: 700; background: #fff; color: #111; }
    td { border: 1px solid #777; vertical-align: top; height: 25mm; padding: .7mm; }
    .day-col { width: 15mm; font-weight: 700; color: #111; text-align: left; }
    .day-main { font-size: 7pt; }
    .day-date { font-size: 5.8pt; color: #333; margin-top: .5mm; }
    .day-note { margin-top: .8mm; padding-top: .6mm; border-top: 1px dotted #777; color: #111; font-size: 5.4pt; font-weight: 400; }
    .lesson { border-left: 1.5px solid #111; padding-left: .7mm; margin-bottom: .8mm; min-height: 9mm; }
    .lesson.changed, .lesson.added { border-left-style: dashed; }
    .subject { font-weight: 700; font-size: 5.7pt; line-height: 1.12; }
    .line { margin-top: .4mm; color: #222; line-height: 1.1; }
    .room { margin-top: .5mm; font-weight: 700; color: #111; }
    .badge { display: inline-block; margin-top: .4mm; padding: 0 .8mm; border: 1px solid #777; color: #111; font-size: 5.2pt; font-weight: 700; }
    .empty { color: #999; text-align: center; padding-top: 8mm; }
  </style>
</head>
<body>
  <section class="sheet">
    <div class="header">
      <div>
        <div class="title">{{ $.Title }}</div>
        <div class="subtitle">{{ $.Subtitle }}</div>
      </div>
      <div class="meta">
        <div>2 недели на одном листе</div>
        <div>{{ $.GeneratedAt }}</div>
      </div>
    </div>
    <div class="weeks">
      {{ range .Weeks }}
      <div class="week">
        <div class="week-title"><span>{{ .Title }}</span><span>{{ .RangeLabel }}</span></div>
        <table>
          <thead>
            <tr>
              <th class="day-col">День</th>
              {{ range $.Pairs }}<th>{{ . }}</th>{{ end }}
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
      </div>
      {{ end }}
    </div>
  </section>
</body>
</html>`

const scheduleOverridesExportHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4 landscape; margin: 8mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #111; font-family: Arial, sans-serif; font-size: 8pt; }
    .header { display: flex; justify-content: space-between; gap: 8mm; margin-bottom: 5mm; border-bottom: 1px solid #111; padding-bottom: 3mm; }
    .title { font-size: 16pt; font-weight: 700; color: #111; }
    .subtitle { margin-top: 1.5mm; font-size: 9pt; color: #333; }
    .meta { text-align: right; color: #333; line-height: 1.4; white-space: nowrap; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th { color: #111; border: 1px solid #222; padding: 1.4mm 1mm; text-align: left; font-size: 7.4pt; background: #fff; }
    td { border: 1px solid #777; vertical-align: top; padding: 1.3mm; line-height: 1.25; }
    .date { width: 21mm; }
    .pair { width: 15mm; text-align: center; }
    .group { width: 24mm; }
    .action { width: 22mm; font-weight: 700; color: #111; }
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

const teacherBoardExportHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4 portrait; margin: 7mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #111; font-family: Arial, sans-serif; font-size: 6.6pt; }
    .page { page-break-after: always; min-height: 282mm; }
    .page:last-child { page-break-after: auto; }
    .header { display: flex; justify-content: space-between; align-items: flex-start; gap: 4mm; border-bottom: 1px solid #111; padding-bottom: 2mm; margin-bottom: 3mm; }
    .title { font-size: 12pt; font-weight: 700; }
    .subtitle { margin-top: 1mm; font-size: 7.4pt; color: #333; }
    .meta { text-align: right; white-space: nowrap; font-size: 6.4pt; color: #333; }
    .teachers { display: grid; grid-template-columns: repeat(3, 1fr); gap: 3mm; align-items: start; }
    .teacher { min-height: 260mm; border-left: 1px solid #111; padding-left: 1.8mm; }
    .teacher-name { font-size: 8.2pt; font-weight: 700; border-bottom: 1px solid #555; padding-bottom: 1mm; margin-bottom: 1.5mm; min-height: 8mm; }
    .week-title { font-weight: 700; margin: 1.5mm 0 1mm; border-bottom: 1px dotted #777; padding-bottom: .7mm; }
    .day { break-inside: avoid; border-bottom: 1px solid #bbb; padding: 1mm 0; min-height: 9mm; }
    .day-head { font-weight: 700; color: #111; margin-bottom: .7mm; }
    .lesson { margin: .7mm 0 0; line-height: 1.14; }
    .pair { font-weight: 700; }
    .subject { font-weight: 700; }
    .details { color: #222; }
    .empty { color: #888; }
  </style>
</head>
<body>
  {{ range $pageIndex, $page := .Pages }}
  <section class="page">
    <div class="header">
      <div>
        <div class="title">{{ $.Title }}</div>
        <div class="subtitle">{{ $.Subtitle }}</div>
      </div>
      <div class="meta">
        <div>3 преподавателя на лист</div>
        <div>{{ $.GeneratedAt }}</div>
      </div>
    </div>
    <div class="teachers">
      {{ range $page.Teachers }}
      <div class="teacher">
        <div class="teacher-name">{{ .Name }}</div>
        {{ range .Weeks }}
          <div class="week-title">{{ .Title }} | {{ .RangeLabel }}</div>
          {{ range .Days }}
          <div class="day">
            <div class="day-head">{{ .DayName }} {{ .DateLabel }}</div>
            {{ $has := false }}
            {{ range .Cells }}
              {{ $pair := .PairNumber }}
              {{ range .Lessons }}
                {{ $has = true }}
                <div class="lesson">
                  <span class="pair">{{ $pair }}.</span>
                  <span class="subject">{{ .Subject }}</span>
                  <div class="details">
                    {{ if .Primary }}{{ .Primary }}{{ end }}
                    {{ if .Secondary }}; {{ .Secondary }}{{ end }}
                    {{ if .Location }}; {{ .Location }}{{ end }}
                    {{ if .Badge }}; {{ .Badge }}{{ end }}
                    {{ if .Comment }}; {{ .Comment }}{{ end }}
                  </div>
                </div>
              {{ end }}
            {{ end }}
            {{ if not $has }}<div class="empty">-</div>{{ end }}
          </div>
          {{ end }}
        {{ end }}
      </div>
      {{ end }}
    </div>
  </section>
  {{ end }}
</body>
</html>`
