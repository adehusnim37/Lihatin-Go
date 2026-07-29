package notification

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/gin-gonic/gin"
)

const maxCampaignImageSizeBytes int64 = 5 * 1024 * 1024

var allowedCampaignImageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

func (c *Controller) UploadCampaignImage(ctx *gin.Context) {
	adminID := ctx.GetString("user_id")
	if c.campaignImageStore == nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "CAMPAIGN_IMAGE_STORAGE_NOT_CONFIGURED", "Campaign image storage is not configured on server", "image")
		return
	}

	fileHeader, err := ctx.FormFile("image")
	if err != nil || fileHeader == nil {
		httputil.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{"image": "Image file is required"})
		return
	}
	if fileHeader.Size <= 0 {
		httputil.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{"image": "Image file is empty"})
		return
	}
	if fileHeader.Size > maxCampaignImageSizeBytes {
		httputil.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{"image": "Image file must be less than or equal to 5MB"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_IMAGE_READ_FAILED", "Failed to read campaign image", "image")
		return
	}
	defer file.Close()

	contentType, err := detectCampaignImageContentType(file)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "CAMPAIGN_IMAGE_INVALID", "Invalid campaign image", "image")
		return
	}
	if _, ok := allowedCampaignImageContentTypes[contentType]; !ok {
		httputil.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{"image": "Only JPG, PNG, WEBP, or GIF images are allowed"})
		return
	}

	imageURL, objectKey, err := c.campaignImageStore.UploadImage(
		ctx.Request.Context(),
		adminID,
		file,
		fileHeader.Size,
		contentType,
		fileHeader.Filename,
	)
	if err != nil {
		var responseErr *smithyhttp.ResponseError
		if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == http.StatusRequestEntityTooLarge {
			httputil.SendErrorResponse(ctx, http.StatusRequestEntityTooLarge, "CAMPAIGN_IMAGE_TOO_LARGE_FOR_STORAGE", "Image size exceeds the upstream storage gateway limit", "image")
			return
		}
		logger.Logger.Error("Failed uploading campaign image", "user_id", adminID, "error", err.Error())
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_IMAGE_UPLOAD_FAILED", "Failed to upload campaign image", "image")
		return
	}

	httputil.SendCreatedResponse(ctx, gin.H{
		"image_url":  imageURL,
		"object_key": objectKey,
	}, "Campaign image uploaded successfully")
}

func detectCampaignImageContentType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buffer[:n]), nil
}
