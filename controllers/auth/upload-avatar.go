package auth

import (
	"errors"
	"io"
	"mime/multipart"
	stdhttp "net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	apphttp "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/gin-gonic/gin"
)

const maxAvatarSizeBytes int64 = 5 * 1024 * 1024 // 5 MB

var allowedAvatarContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

// UploadAvatar uploads profile photo to OSS and updates user's avatar URL.
func (c *Controller) UploadAvatar(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	if c.avatarStore == nil {
		apphttp.SendErrorResponse(
			ctx,
			stdhttp.StatusServiceUnavailable,
			"AVATAR_STORAGE_NOT_CONFIGURED",
			"Avatar storage is not configured on server",
			"avatar",
			userID,
		)
		return
	}

	fileHeader, err := ctx.FormFile("avatar")
	if err != nil || fileHeader == nil {
		apphttp.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{
			"avatar": "Avatar file is required",
		})
		return
	}

	if fileHeader.Size <= 0 {
		apphttp.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{
			"avatar": "Avatar file is empty",
		})
		return
	}

	if fileHeader.Size > maxAvatarSizeBytes {
		apphttp.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{
			"avatar": "Avatar file must be less than or equal to 5MB",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		apphttp.SendErrorResponse(
			ctx,
			stdhttp.StatusInternalServerError,
			"AVATAR_FILE_READ_FAILED",
			"Failed to read avatar file",
			"avatar",
			userID,
		)
		return
	}
	defer file.Close()

	contentType, err := detectImageContentType(file)
	if err != nil {
		apphttp.SendErrorResponse(
			ctx,
			stdhttp.StatusBadRequest,
			"AVATAR_FILE_INVALID",
			"Invalid avatar file",
			"avatar",
			userID,
		)
		return
	}

	if _, ok := allowedAvatarContentTypes[contentType]; !ok {
		apphttp.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{
			"avatar": "Only JPG, PNG, WEBP, or GIF images are allowed",
		})
		return
	}

	avatarURL, objectKey, err := c.avatarStore.UploadAvatar(
		ctx.Request.Context(),
		userID,
		file,
		fileHeader.Size,
		contentType,
		fileHeader.Filename,
	)
	if err != nil {
		var responseErr *smithyhttp.ResponseError
		if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == stdhttp.StatusRequestEntityTooLarge {
			apphttp.SendErrorResponse(
				ctx,
				stdhttp.StatusRequestEntityTooLarge,
				"AVATAR_UPLOAD_TOO_LARGE_FOR_STORAGE",
				"Avatar size exceeds upstream storage gateway limit. Reduce image size or increase proxy limit.",
				"avatar",
				userID,
			)
			return
		}

		logger.Logger.Error("Failed uploading avatar to object storage", "user_id", userID, "error", err.Error())
		apphttp.SendErrorResponse(
			ctx,
			stdhttp.StatusInternalServerError,
			"AVATAR_UPLOAD_FAILED",
			"Failed to upload avatar",
			"avatar",
			userID,
		)
		return
	}

	updatePayload := dto.UpdateProfileRequest{
		Avatar: &avatarURL,
	}
	if err := c.repo.GetUserRepository().UpdateUser(userID, updatePayload); err != nil {
		apphttp.HandleError(ctx, err, userID)
		return
	}

	apphttp.SendOKResponse(ctx, gin.H{
		"avatar_url": avatarURL,
		"object_key": objectKey,
	}, "Avatar uploaded successfully")
}

func detectImageContentType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return stdhttp.DetectContentType(buffer[:n]), nil
}
