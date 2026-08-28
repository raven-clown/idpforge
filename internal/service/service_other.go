//go:build !windows

package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// Run executes fn until SIGTERM/SIGINT, then calls stop and waits for fn to
// return. On Linux this is what systemd's Type=simple expects.
func Run(fn func(ctx context.Context) error, stop func()) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- fn(ctx) }()

	<-ctx.Done()
	stop()
	return <-errCh
}
