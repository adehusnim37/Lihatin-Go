package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3SupportAttachmentStorage handles support file uploads to S3-compatible storage (R2/OSS).
type S3SupportAttachmentStorage struct {
	base *S3AvatarStorage
}

func NewS3SupportAttachmentStorageFromEnv() (*S3SupportAttachmentStorage, error) {
	base, err := NewS3AvatarStorageFromEnv()
	if err != nil {
		return nil, err
	}
	return &S3SupportAttachmentStorage{base: base}, nil
}

func (s *S3SupportAttachmentStorage) UploadAttachment(
	ctx context.Context,
	ticketID string,
	messageID string,
	fileName string,
	contentType string,
	data []byte,
) (objectURL string, objectKey string, err error) {
	if s == nil || s.base == nil || s.base.client == nil {
		return "", "", fmt.Errorf("support attachment storage not configured")
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("empty attachment payload")
	}

	objectKey = s.buildObjectKey(ticketID, messageID, fileName, contentType)
	reader := bytes.NewReader(data)

	_, err = s.base.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        awsv2.String(s.base.bucket),
		Key:           awsv2.String(objectKey),
		Body:          reader,
		ContentLength: awsv2.Int64(int64(len(data))),
		ContentType:   awsv2.String(contentType),
		CacheControl:  awsv2.String("private, max-age=0, no-cache"),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload support attachment: %w", err)
	}

	return s.base.buildObjectURL(objectKey), objectKey, nil
}

func (s *S3SupportAttachmentStorage) buildObjectKey(ticketID, messageID, fileName, contentType string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if ext == "" {
		if exts, err := mime.ExtensionsByType(strings.TrimSpace(contentType)); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".bin"
	}

	randomSuffix := randomHex(8)
	return fmt.Sprintf(
		"support/%s/%s/%d-%s%s",
		strings.TrimSpace(ticketID),
		strings.TrimSpace(messageID),
		time.Now().UnixNano(),
		randomSuffix,
		ext,
	)
}

func randomHex(byteLength int) string {
	if byteLength <= 0 {
		return "00"
	}
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
