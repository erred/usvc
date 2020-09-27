package usvc

import (
	"flag"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
)

type TracerOpts struct {
	CollectorEndpoint string
}

func (o *TracerOpts) Flags(fs *flag.FlagSet) {
	fs.StringVar(&o.CollectorEndpoint, "trace.collector", "http://jaeger:14268/api/traces?format=jaeger.thrift", "jaeger collector endpoint")
}

// Tracer installs a global tracer
func (o TracerOpts) Tracer() (shutdown func() error, err error) {
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint(
			o.CollectorEndpoint,
			jaeger.WithCollectorEndpointOptionFromEnv(),
		),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: name,
		}),
		jaeger.WithProcessFromEnv(),
		// jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.NeverSample()}),
		// jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	shutdown = func() error {
		flush()
		return nil
	}

	b3 := b3.B3{}
	global.SetPropagators(propagation.New(
		propagation.WithExtractors(b3),
		propagation.WithInjectors(b3),
	))
	return
}
