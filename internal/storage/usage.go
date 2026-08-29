package storage

import (
	"os"
	"path/filepath"
)

// LocalDirSize sums file sizes under dir, for reporting disk usage of the
// local storage backend on the admin UI's usage graph. Returns 0, nil for
// a directory that doesn't exist yet (fresh install, no uploads so far).
func LocalDirSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}
