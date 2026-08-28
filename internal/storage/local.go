package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type localStore struct {
	dir string
}

func newLocalStore(dir string) (*localStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &localStore{dir: dir}, nil
}

func (s *localStore) Put(_ context.Context, key string, data io.Reader, _ int64, _ string) (string, error) {
	dest := filepath.Join(s.dir, filepath.Base(key))
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", err
	}
	return "/uploads/" + filepath.Base(key), nil
}

func (s *localStore) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(s.dir, filepath.Base(key)))
}
