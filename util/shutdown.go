package util

import (
	"os"
	"os/signal"
	"syscall"
)

// MakeShutdownCh creates a channel which will emit whenever a SIGTERM/SIGINT
// is received by the application - this is used to close any active sessions.
func MakeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}

// NotShutdown returns an interator which will be closed when a shutdown request
// is received by the application.
func NotShutdown() <-chan struct{} {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	ch := make(chan struct{})

	go func() {
		for {
			select {
			case <-signalCh:
				close(ch)
				return
			case ch <- struct{}{}:
			}
		}
	}()

	return ch
}
