package usvc

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	// Signals is the default signals used to cancel SignalContext.
	// SIGINT: Ctrl-c,
	// SIGTERM: k8s kill,
	Signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
)

// SignalContext provides a context cancelled by the default Signals
// as well as any additional ones passed in
func SignalContext(sigs ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, append(Signals, sigs...)...)
		<-c
		cancel()
	}()
	return ctx
}
