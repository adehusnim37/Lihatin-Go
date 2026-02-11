package logger

import (
	"log/slog"
	"os"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

var Logger *slog.Logger

func init() {
	// Development: Text format
	if config.GetEnvOrDefault(config.Env, "development") == "development" {
		Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		// Production: JSON format
		Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
}
