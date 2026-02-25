package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsv2config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3AvatarStorage handles avatar uploads to S3-compatible object storage.
type S3AvatarStorage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
	usePathStyle  bool
}

func NewS3AvatarStorageFromEnv() (*S3AvatarStorage, error) {
	endpoint := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSEndpoint, ""))
	region := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSRegion, "auto"))
	accessKey := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSAccessKey, ""))
	secretKey := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSSecretKey, ""))
	bucket := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSBucket, ""))
	pathStyleMode := strings.ToLower(strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSPathStyle, "auto")))

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("OSS config incomplete: set OSS_ENDPOINT, OSS_ACCESS_KEY, OSS_SECRET_KEY, and OSS_BUCKET")
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	usePathStyle := pathStyleMode != "virtual"
	publicBaseURL := strings.TrimSpace(config.GetEnvOrDefault(config.EnvOSSPublicBaseURL, endpoint))
	publicBaseURL = strings.TrimRight(publicBaseURL, "/")

	cfg, err := awsv2config.LoadDefaultConfig(
		context.Background(),
		awsv2config.WithRegion(region),
		awsv2config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load OSS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = awsv2.String(endpoint)
		o.UsePathStyle = usePathStyle
	})

	return &S3AvatarStorage{
		client:        client,
		bucket:        bucket,
		publicBaseURL: publicBaseURL,
		usePathStyle:  usePathStyle,
	}, nil
}

func (s *S3AvatarStorage) UploadAvatar(
	ctx context.Context,
	userID string,
	file io.Reader,
	fileSize int64,
	contentType string,
	originalFilename string,
) (string, string, error) {
	objectKey := s.buildObjectKey(userID, contentType, originalFilename)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        awsv2.String(s.bucket),
		Key:           awsv2.String(objectKey),
		Body:          file,
		ContentLength: awsv2.Int64(fileSize),
		ContentType:   awsv2.String(contentType),
		CacheControl:  awsv2.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload avatar: %w", err)
	}

	return s.buildObjectURL(objectKey), objectKey, nil
}

func (s *S3AvatarStorage) buildObjectKey(userID, contentType, originalFilename string) string {
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".bin"
	}

	return fmt.Sprintf("avatars/%s/%d%s", userID, time.Now().UnixNano(), ext)
}

func (s *S3AvatarStorage) buildObjectURL(objectKey string) string {
	encodedKey := encodeObjectKey(objectKey)

	baseURL, err := url.Parse(s.publicBaseURL)
	if err != nil || baseURL.Host == "" {
		return fmt.Sprintf("%s/%s/%s", s.publicBaseURL, s.bucket, encodedKey)
	}

	if s.usePathStyle {
		baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + "/" + s.bucket + "/" + encodedKey
		return baseURL.String()
	}

	baseURL.Host = s.bucket + "." + baseURL.Host
	baseURL.Path = "/" + encodedKey
	return baseURL.String()
}

func encodeObjectKey(objectKey string) string {
	parts := strings.Split(objectKey, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
