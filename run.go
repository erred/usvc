package usvc

import "context"

// Service is the minimal interface to be implemented
// to be passed to Run
type Service interface {
	Run() error
	Shutdown() error
}

// Run runs the service and calls Shutdown when the context is cancelled
func Run(ctx context.Context, svc Service) error {
	errc := make(chan error)
	go func() {
		errc <- svc.Run()
	}()

	var err error
	select {
	case <-ctx.Done():
		err = svc.Shutdown()
	case err = <-errc:
	}
	return err
}
