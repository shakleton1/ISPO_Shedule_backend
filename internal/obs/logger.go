package obs

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ispo-schedule/internal/config"
)

func InitLogger(cfg config.LogConfig) {
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var out io.Writer = os.Stdout
	if cfg.Pretty {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	}
	log.Logger = zerolog.New(out).With().Timestamp().Logger()
}

func parseLevel(s string) zerolog.Level {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info", "":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
