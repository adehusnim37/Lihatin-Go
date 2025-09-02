package utils

import (
    "log/slog"
    "os"
)

var Logger *slog.Logger

func init() {
    // Development: Text format
    if os.Getenv("ENV") == "development" {
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