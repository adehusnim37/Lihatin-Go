package email

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const resendStageTTL = 30 * 24 * time.Hour

func resendStageKey(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return fmt.Sprintf("email_resend_stage:%s", normalized)
}

func getResendStage(ctx *gin.Context, email string) int {
	sessionManager := middleware.GetSessionManager()
	if sessionManager == nil || sessionManager.GetRedisClient() == nil {
		return 0
	}

	value, err := sessionManager.GetRedisClient().Get(ctx.Request.Context(), resendStageKey(email)).Result()
	if err != nil {
		if err != redis.Nil {
			logger.Logger.Warn("Failed to read resend stage",
				"email", email,
				"error", err.Error(),
			)
		}
		return 0
	}

	stage, err := strconv.Atoi(value)
	if err != nil || stage < 0 {
		return 0
	}

	return stage
}

func setResendStage(ctx *gin.Context, email string, stage int) {
	if stage < 0 {
		stage = 0
	}

	sessionManager := middleware.GetSessionManager()
	if sessionManager == nil || sessionManager.GetRedisClient() == nil {
		return
	}

	err := sessionManager.GetRedisClient().Set(
		ctx.Request.Context(),
		resendStageKey(email),
		stage,
		resendStageTTL,
	).Err()
	if err != nil {
		logger.Logger.Warn("Failed to write resend stage",
			"email", email,
			"stage", stage,
			"error", err.Error(),
		)
	}
}

func clearResendStage(ctx *gin.Context, email string) {
	sessionManager := middleware.GetSessionManager()
	if sessionManager == nil || sessionManager.GetRedisClient() == nil {
		return
	}

	err := sessionManager.GetRedisClient().Del(
		ctx.Request.Context(),
		resendStageKey(email),
	).Err()
	if err != nil {
		logger.Logger.Warn("Failed to clear resend stage",
			"email", email,
			"error", err.Error(),
		)
	}
}
