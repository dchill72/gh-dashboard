package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// L is the package-level logger. It discards everything unless Init is called.
var L *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

// Init sets up file-based logging when the LOGGING=1 env var is set.
// Logs are written to ./logs/<timestamp>.log in the current working directory.
// Returns a no-op if LOGGING is not set.
func Init() error {
	if os.Getenv("LOGGING") != "1" {
		return nil
	}

	if err := os.MkdirAll("logs", 0o755); err != nil {
		return err
	}

	name := filepath.Join("logs", time.Now().Format("20060102-150405")+".log")
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	L = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	L.Info("logger initialized", "file", name)
	return nil
}
