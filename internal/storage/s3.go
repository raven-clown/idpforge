package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/raven-clown/idpforge/internal/config"
)

// s3Store works against any S3-compatible endpoint: MinIO, AWS S3,
// Cloudflare R2, Backblaze B2, etc.
type s3Store struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func newS3Store(cfg config.StorageConfig) (*s3Store, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	publicURL := cfg.S3PublicBaseURL
	if publicURL == "" {
		scheme := "http"
		if cfg.S3UseSSL {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s/%s", scheme, cfg.S3Endpoint, cfg.S3Bucket)
	}

	return &s3Store{client: client, bucket: cfg.S3Bucket, publicURL: publicURL}, nil
}

func (s *s3Store) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return s.publicURL + "/" + key, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
