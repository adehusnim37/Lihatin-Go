package jobs

import (
    "context"
    "time"

    "github.com/adehusnim37/lihatin-go/repositories"
    "github.com/adehusnim37/lihatin-go/internal/pkg/logger"
)

// RunCleanupScheduler runs cleanup periodically and waits for each run to finish.
// interval: how often to run (e.g. 24*time.Hour)
// olderThanDays: argument passed to CleanupOldAttempts
func RunCleanupScheduler(ctx context.Context, repo *repositories.LoginAttemptRepository, interval time.Duration, olderThanDays int) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    logger.Logger.Info("LoginAttempt cleanup scheduler started", "interval", interval.String(), "older_than_days", olderThanDays)

    for {
        // run immediately on start (optional)
        if err := repo.CleanupOldAttempts(olderThanDays); err != nil {
            logger.Logger.Error("CleanupOldAttempts failed", "error", err.Error())
        } else {
            logger.Logger.Info("CleanupOldAttempts finished")
        }

        select {
        case <-ctx.Done():
            logger.Logger.Info("Cleanup scheduler stopped by context")
            return
        case <-ticker.C:
            // synchronous: wait for cleanup to finish before next loop iteration
            if err := repo.CleanupOldAttempts(olderThanDays); err != nil {
                logger.Logger.Error("CleanupOldAttempts failed", "error", err.Error())
            } else {
                logger.Logger.Info("CleanupOldAttempts finished")
            }
        }
    }
}