package httpapi

const scheduleHTMLTemplate = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4; margin: 0; }
    body { margin: 0; padding: 0; }
    .a4-page { width: 210mm; height: 297mm; padding: 8mm; box-sizing: border-box; font-family: Roboto, Arial, sans-serif; }
    .header { font-size: 16pt; font-weight: 700; text-align: center; margin-bottom: 6mm; }

    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4mm; }
    .week { border: 2px solid #000; }
    .week-title { font-size: 12pt; font-weight: 700; text-align: center; padding: 2mm 0; border-bottom: 2px solid #000; }

    .day-row { position: relative; display: grid; grid-template-columns: 10mm 1fr; min-height: 36mm; border-bottom: 2px solid #000; }
    .day-row:last-child { border-bottom: 0; }
    .day-title { writing-mode: vertical-rl; transform: rotate(180deg); font-weight: 700; border-right: 2px solid #000; display:flex; align-items:center; justify-content:center; font-size: 12pt; }

    .lessons { display: grid; grid-template-columns: repeat(8, 1fr); }
    .cell { position: relative; border-right: 1px solid #000; padding: 1mm; box-sizing: border-box; min-height: 36mm; }
    .cell:last-child { border-right: 0; }
    .subject { font-size: 8pt; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .teacher { font-size: 7pt; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .room { font-size: 9pt; font-weight: 700; margin-top: 1mm; }
    .handwritten { font-family: Caveat, "Comic Sans MS", cursive; font-size: 11pt; }

    .split { display: grid; grid-template-columns: 1fr 2px 1fr; height: 100%; }
    .split .sep { background: #000; }
    .split .part { padding: 1mm; box-sizing: border-box; }

    .overlay-layer { position:absolute; inset: 0; display:flex; align-items:center; justify-content:center; pointer-events:none; }
    .overlay-text { font-family: Caveat, "Comic Sans MS", cursive; font-size: 26pt; color: #2c3e50; transform: rotate(-5deg); opacity: 0.9; }
  </style>
</head>
<body>
  <div class="a4-page">
    <div class="header">Расписание занятий группы {{ .GroupName }}</div>
    <div class="grid">

      <div class="week">
        <div class="week-title">Неделя 1</div>
        {{ range .DaysWeek1 }}
          <div class="day-row">
            <div class="day-title">{{ .DayName }}</div>
            <div class="lessons">
              {{ range .Lessons }}
                <div class="cell">
                  {{ if .IsSplit }}
                    <div class="split">
                      <div class="part">
                        {{ if .Sub1 }}
                          <div class="subject">{{ .Sub1.Subject }}</div>
                          <div class="teacher">{{ .Sub1.Teacher }}</div>
                          <div class="room {{ if .Sub1.IsChanged }}handwritten{{ end }}">{{ .Sub1.Location }}</div>
                        {{ end }}
                      </div>
                      <div class="sep"></div>
                      <div class="part">
                        {{ if .Sub2 }}
                          <div class="subject">{{ .Sub2.Subject }}</div>
                          <div class="teacher">{{ .Sub2.Teacher }}</div>
                          <div class="room {{ if .Sub2.IsChanged }}handwritten{{ end }}">{{ .Sub2.Location }}</div>
                        {{ end }}
                      </div>
                    </div>
                  {{ else }}
                    {{ if .Single }}
                      <div class="subject">{{ .Single.Subject }}</div>
                      <div class="teacher">{{ .Single.Teacher }}</div>
                      <div class="room {{ if .Single.IsChanged }}handwritten{{ end }}">{{ .Single.Location }}</div>
                    {{ end }}
                  {{ end }}
                </div>
              {{ end }}
            </div>
            {{ if .OverlayText }}
              <div class="overlay-layer"><div class="overlay-text">{{ .OverlayText }}</div></div>
            {{ end }}
          </div>
        {{ end }}
      </div>

      <div class="week">
        <div class="week-title">Неделя 2</div>
        {{ range .DaysWeek2 }}
          <div class="day-row">
            <div class="day-title">{{ .DayName }}</div>
            <div class="lessons">
              {{ range .Lessons }}
                <div class="cell">
                  {{ if .IsSplit }}
                    <div class="split">
                      <div class="part">
                        {{ if .Sub1 }}
                          <div class="subject">{{ .Sub1.Subject }}</div>
                          <div class="teacher">{{ .Sub1.Teacher }}</div>
                          <div class="room {{ if .Sub1.IsChanged }}handwritten{{ end }}">{{ .Sub1.Location }}</div>
                        {{ end }}
                      </div>
                      <div class="sep"></div>
                      <div class="part">
                        {{ if .Sub2 }}
                          <div class="subject">{{ .Sub2.Subject }}</div>
                          <div class="teacher">{{ .Sub2.Teacher }}</div>
                          <div class="room {{ if .Sub2.IsChanged }}handwritten{{ end }}">{{ .Sub2.Location }}</div>
                        {{ end }}
                      </div>
                    </div>
                  {{ else }}
                    {{ if .Single }}
                      <div class="subject">{{ .Single.Subject }}</div>
                      <div class="teacher">{{ .Single.Teacher }}</div>
                      <div class="room {{ if .Single.IsChanged }}handwritten{{ end }}">{{ .Single.Location }}</div>
                    {{ end }}
                  {{ end }}
                </div>
              {{ end }}
            </div>
            {{ if .OverlayText }}
              <div class="overlay-layer"><div class="overlay-text">{{ .OverlayText }}</div></div>
            {{ end }}
          </div>
        {{ end }}
      </div>

    </div>
  </div>
</body>
</html>`
