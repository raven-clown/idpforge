// Package storage saves user-uploaded files (profile pictures) to local
// disk or an S3-compatible bucket (MinIO, AWS S3, Cloudflare R2, ...).
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/raven-clown/idpforge/internal/config"
)

type Store interface {
	// Put saves data under key and returns a URL the client can fetch it from.
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) (url string, err error)
	Delete(ctx context.Context, key string) error
}

func New(cfg config.StorageConfig) (Store, error) {
	switch cfg.Backend {
	case "local":
		return newLocalStore(cfg.LocalDir)
	case "s3":
		return newS3Store(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage backend %q", cfg.Backend)
	}
}
