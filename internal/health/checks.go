// Package health runs named diagnostic checks (database, cache, disk space)
// so /healthz reports specifically what is wrong (config, connectivity,
// disk full) instead of a bare "ok"/"not ok".
package health

import (
	"context"
	"fmt"
)

type CheckResult struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type Check func(ctx context.Context) error

type Checker struct {
	checks map[string]Check
}

func NewChecker() *Checker {
	return &Checker{checks: map[string]Check{}}
}

func (c *Checker) Register(name string, check Check) {
	c.checks[name] = check
}

// Run executes every registered check and returns per-check results plus
// whether all of them passed.
func (c *Checker) Run(ctx context.Context) (results []CheckResult, allOK bool) {
	allOK = true
	for name, check := range c.checks {
		if err := check(ctx); err != nil {
			allOK = false
			results = append(results, CheckResult{Name: name, OK: false, Error: err.Error()})
			continue
		}
		results = append(results, CheckResult{Name: name, OK: true})
	}
	return results, allOK
}

// DiskSpaceCheck fails when free space on the filesystem containing path
// drops below minFreeBytes, so an operator sees "disk full" as a specific
// health failure instead of a generic write error later.
func DiskSpaceCheck(path string, minFreeBytes uint64) Check {
	return func(_ context.Context) error {
		free, err := freeBytes(path)
		if err != nil {
			return fmt.Errorf("check disk space at %s: %w", path, err)
		}
		if free < minFreeBytes {
			return fmt.Errorf("low disk space at %s: %d bytes free, want at least %d", path, free, minFreeBytes)
		}
		return nil
	}
}
