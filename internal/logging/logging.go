package logging

import (
	"log/slog"
	"os"
	"strings"
)

var logger *slog.Logger

func Setup(level, file string) {
	var h slog.Handler
	lvl := parseLevel(level)
	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err == nil {
			h = slog.NewJSONHandler(f, &slog.HandlerOptions{Level: lvl})
		}
	}
	if h == nil {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}
	logger = slog.New(h)
}

func parseLevel(l string) slog.Level {
	switch strings.ToUpper(l) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func L() *slog.Logger { return logger }
