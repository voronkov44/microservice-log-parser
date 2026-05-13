package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func New(logLevel string, logFilePath string) (*slog.Logger, func(), error) {
	level := parseLevel(logLevel)

	var writers []io.Writer
	writers = append(writers, os.Stdout)

	var file *os.File
	if logFilePath != "" {
		dir := filepath.Dir(logFilePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, nil, err
			}
		}

		var err error
		file, err = os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return nil, nil, err
		}

		writers = append(writers, file)
	}

	multiWriter := io.MultiWriter(writers...)

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	log := slog.New(handler)

	cleanup := func() {
		if file != nil {
			_ = file.Close()
		}
	}

	return log, cleanup, nil
}

func parseLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
