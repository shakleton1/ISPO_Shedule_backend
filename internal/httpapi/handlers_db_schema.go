package httpapi

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dbTableColumn struct {
	TableName  string `gorm:"column:table_name"`
	Column     string `gorm:"column:column_name"`
	DataType   string `gorm:"column:data_type"`
	IsNullable string `gorm:"column:is_nullable"`
}

type dbFK struct {
	TableName     string `gorm:"column:table_name"`
	ColumnName    string `gorm:"column:column_name"`
	RefTableName  string `gorm:"column:ref_table_name"`
	RefColumnName string `gorm:"column:ref_column_name"`
}

func handleAdminDBSchema(repoDB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Keep it read-only and safe: only introspect schema.
		cols, err := listPublicColumns(repoDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fks, err := listPublicFKs(repoDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		mermaid := buildMermaidER(cols, fks)

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, renderMermaidPage("DB Schema", mermaid))
	}
}

func listPublicColumns(db *gorm.DB) ([]dbTableColumn, error) {
	var rows []dbTableColumn
	// Exclude system schemas.
	err := db.Raw(`
SELECT
  table_name,
  column_name,
  data_type,
  is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
ORDER BY table_name, ordinal_position;
`).Scan(&rows).Error
	return rows, err
}

func listPublicFKs(db *gorm.DB) ([]dbFK, error) {
	var rows []dbFK
	err := db.Raw(`
WITH fk AS (
  SELECT
    c.conrelid,
    c.confrelid,
    unnest(c.conkey) WITH ORDINALITY AS conkey(attnum, ord),
    unnest(c.confkey) WITH ORDINALITY AS confkey(attnum, ord2)
  FROM pg_constraint c
  WHERE c.contype = 'f'
    AND c.connamespace = 'public'::regnamespace
)
SELECT
  cr.relname AS table_name,
  a.attname AS column_name,
  pr.relname AS ref_table_name,
  af.attname AS ref_column_name
FROM fk
JOIN pg_class cr ON cr.oid = fk.conrelid
JOIN pg_class pr ON pr.oid = fk.confrelid
JOIN pg_attribute a ON a.attrelid = fk.conrelid AND a.attnum = fk.conkey.attnum
JOIN pg_attribute af ON af.attrelid = fk.confrelid AND af.attnum = fk.confkey.attnum
WHERE fk.conkey.ord = fk.confkey.ord2
ORDER BY table_name, column_name;
`).Scan(&rows).Error
	return rows, err
}

func buildMermaidER(cols []dbTableColumn, fks []dbFK) string {
	// Group columns by table.
	tables := map[string][]dbTableColumn{}
	for _, c := range cols {
		tables[c.TableName] = append(tables[c.TableName], c)
	}

	tableNames := make([]string, 0, len(tables))
	for name := range tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	var sb strings.Builder
	sb.WriteString("erDiagram\n")

	for _, t := range tableNames {
		sb.WriteString("  ")
		sb.WriteString(sanitizeMermaidIdent(t))
		sb.WriteString(" {\n")
		for _, col := range tables[t] {
			dt := mermaidType(col.DataType)
			// Mermaid ER expects: TYPE name
			sb.WriteString("    ")
			sb.WriteString(dt)
			sb.WriteString(" ")
			sb.WriteString(sanitizeMermaidIdent(col.Column))
			if strings.EqualFold(col.IsNullable, "NO") {
				sb.WriteString(" \"NOT NULL\"")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("  }\n\n")
	}

	// Relations
	for _, fk := range fks {
		child := sanitizeMermaidIdent(fk.TableName)
		parent := sanitizeMermaidIdent(fk.RefTableName)
		label := sanitizeMermaidLabel(fk.ColumnName)
		// Many (child) to one (parent)
		sb.WriteString("  ")
		sb.WriteString(child)
		sb.WriteString(" }o--|| ")
		sb.WriteString(parent)
		sb.WriteString(" : \"")
		sb.WriteString(label)
		sb.WriteString("\"\n")
	}

	return sb.String()
}

func mermaidType(pgDataType string) string {
	// Mermaid ER attribute type must be a single token (no spaces).
	// information_schema.columns.data_type often contains multi-word types.
	dt := strings.ToLower(strings.TrimSpace(pgDataType))
	switch dt {
	case "bigint":
		return "BIGINT"
	case "integer":
		return "INT"
	case "smallint":
		return "SMALLINT"
	case "boolean":
		return "BOOL"
	case "text":
		return "TEXT"
	case "character varying", "character", "varchar", "char":
		return "TEXT"
	case "date":
		return "DATE"
	case "time without time zone", "time with time zone":
		return "TIME"
	case "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamp with time zone":
		return "TIMESTAMPTZ"
	case "uuid":
		return "UUID"
	case "json", "jsonb":
		return strings.ToUpper(dt)
	case "numeric", "decimal":
		return "NUMERIC"
	case "real":
		return "REAL"
	case "double precision":
		return "DOUBLE"
	default:
		// Fall back to first word uppercased.
		fields := strings.Fields(dt)
		if len(fields) == 0 {
			return "TYPE"
		}
		return strings.ToUpper(fields[0])
	}
}

func sanitizeMermaidIdent(s string) string {
	// Mermaid identifiers cannot contain dashes/spaces reliably.
	// Our table/column names use snake_case; keep alnum + underscore.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	return out
}

func sanitizeMermaidLabel(s string) string {
	// Escape quotes in labels.
	return strings.ReplaceAll(s, "\"", "'")
}

func renderMermaidPage(title string, mermaid string) string {
	// Mermaid is fetched from CDN; this is for dev/admin usage.
	return `<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<title>` + htmlEscape(title) + `</title>
		<style>
			body { margin: 0; font-family: system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif; }
			header { padding: 12px 16px; }
			main { padding: 16px; overflow: auto; }
			.hint { font-size: 12px; opacity: 0.7; }
		</style>
		<script type="module">
			import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
			mermaid.initialize({ startOnLoad: true, theme: 'default' });
		</script>
	</head>
	<body>
		<header>
			<div><strong>` + htmlEscape(title) + `</strong></div>
			<div class="hint">Read-only schema view (tables + foreign keys). Use browser zoom if needed.</div>
		</header>
		<main>
			<pre class="mermaid">` + htmlEscape(mermaid) + `</pre>
		</main>
	</body>
</html>`
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
