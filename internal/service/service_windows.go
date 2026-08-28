//go:build windows

package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "idpforge"

// Run executes fn under the Windows Service Control Manager when installed
// as a service (sc.exe create / New-Service), or directly with Ctrl+C
// handling when run interactively from a console.
func Run(fn func(ctx context.Context) error, stop func()) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isService {
		return runInteractive(fn, stop)
	}
	return svc.Run(serviceName, &handler{fn: fn, stop: stop})
}

func runInteractive(fn func(ctx context.Context) error, stop func()) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- fn(ctx) }()

	<-ctx.Done()
	stop()
	return <-errCh
}

type handler struct {
	fn   func(ctx context.Context) error
	stop func()
}

func (h *handler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- h.fn(ctx) }()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case err := <-errCh:
			_ = err
			s <- svc.Status{State: svc.StopPending}
			cancel()
			return false, 0
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				h.stop()
				cancel()
				<-errCh
				return false, 0
			}
		}
	}
}
