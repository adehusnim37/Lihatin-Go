package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3CampaignImageStorage stores public images used by promotional emails.
type S3CampaignImageStorage struct {
	base *S3AvatarStorage
}

func NewS3CampaignImageStorageFromEnv() (*S3CampaignImageStorage, error) {
	base, err := NewS3AvatarStorageFromEnv()
	if err != nil {
		return nil, err
	}
	return &S3CampaignImageStorage{base: base}, nil
}

func (s *S3CampaignImageStorage) UploadImage(
	ctx context.Context,
	adminID string,
	file io.Reader,
	fileSize int64,
	contentType string,
	_ string,
) (objectURL string, objectKey string, err error) {
	if s == nil || s.base == nil || s.base.client == nil {
		return "", "", fmt.Errorf("campaign image storage not configured")
	}

	ext := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}[contentType]
	if ext == "" {
		return "", "", fmt.Errorf("unsupported campaign image content type %q", contentType)
	}

	objectKey = fmt.Sprintf(
		"campaigns/%s/%d-%s%s",
		strings.TrimSpace(adminID),
		time.Now().UnixNano(),
		randomHex(8),
		ext,
	)
	_, err = s.base.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        awsv2.String(s.base.bucket),
		Key:           awsv2.String(objectKey),
		Body:          file,
		ContentLength: awsv2.Int64(fileSize),
		ContentType:   awsv2.String(contentType),
		CacheControl:  awsv2.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload campaign image: %w", err)
	}

	return s.base.buildObjectURL(objectKey), objectKey, nil
}
