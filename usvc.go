package usvc

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FlagRegisterer interface {
	RegisterFlags(*flag.FlagSet)
}

// Conf holds configs for creating a http.Server
type Conf struct {
	HTTPAddr    string
	GRPCAddr    string
	TLSCertFile string
	TLSKeyFile  string
	LogLevel    string
	LogFormat   string
}

// DefaultConf uses a new flagset and os.Args,
// adding all flags
func DefaultConf(frs ...FlagRegisterer) Conf {
	var c Conf

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	c.RegisterFlags(fs)
	for _, fr := range frs {
		fr.RegisterFlags(fs)
	}

	fs.Parse(os.Args[1:])
	return c
}

// RegisterFlags adds flags to flagset
func (c *Conf) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.HTTPAddr, "http.addr", ":8080", "listen addr for http")
	fs.StringVar(&c.GRPCAddr, "grpc.addr", ":8080", "listen addr for grpc")
	fs.StringVar(&c.TLSCertFile, "tls.crt", "", "tls cert file")
	fs.StringVar(&c.TLSKeyFile, "tls.key", "", "tls key file")
	fs.StringVar(&c.LogLevel, "log.level", "", "logging level: debug, info, warn, error")
	fs.StringVar(&c.LogFormat, "log.format", "", "format: logfmt, json")
}

// Logger returns a configured logger
func (c Conf) Logger() zerolog.Logger {
	lvl, _ := zerolog.ParseLevel(c.LogLevel)
	var out io.Writer
	switch c.LogFormat {
	case "logfmt":
		out = zerolog.ConsoleWriter{
			Out: os.Stdout,
		}
	case "json":
		fallthrough
	default:
		out = os.Stdout
	}
	return zerolog.New(out).Level(lvl).With().Timestamp().Logger()
}

type Runner func(context.Context) error

func (c Conf) Server(m *http.ServeMux) (*http.Server, *grpc.Server, Runner, error) {
	latency := metric.Must(global.Meter(os.Args[0])).NewInt64ValueRecorder(
		"request_latency_ms",
		metric.WithDescription("http request serve latency"),
		metric.WithUnit(unit.Milliseconds),
	)

	if m == nil {
		m = http.NewServeMux()
	}

	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)
	m.Handle("/health", healthOK)
	promExporter, _ := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{1, 5, 10, 50, 100},
	})
	m.Handle("/metrics", promExporter)

	// http
	h := httpMid(m, c.Logger(), latency)
	h = corsAllowAll(h)

	// grpc
	var grpctls bool
	var opts []grpc.ServerOption
	if c.TLSKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			return nil, nil, nil, err
		}
		opts = append(opts, grpc.Creds(creds))
		grpctls = true
	}
	opts = append(opts, grpcMid(c.Logger(), latency))
	grpcServer := grpc.NewServer(opts...)

	// share
	if c.GRPCAddr == c.HTTPAddr {
		httpServer, run, err := c.sharedServer(h, grpcServer)
		return httpServer, grpcServer, run, err
	}

	// separate
	grpcRun := func(ctx context.Context) error {
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-c:
			case <-ctx.Done():
			}
			grpcServer.GracefulStop()
		}()

		lis, err := net.Listen("tcp", c.GRPCAddr)
		if err != nil {
			return err
		}

		lg := c.Logger()
		lg.Info().Str("grpc-addr", c.GRPCAddr).Bool("tls", grpctls).Msg("started grpc server")
		return grpcServer.Serve(lis)
	}
	httpServer, httpRun, err := c.sharedServer(h, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	run := func(ctx context.Context) error {
		group, ctx := errgroup.WithContext(ctx)
		group.Go(func() error { return httpRun(ctx) })
		group.Go(func() error { return grpcRun(ctx) })
		return group.Wait()
	}
	return httpServer, grpcServer, run, nil

}

func (c Conf) sharedServer(h http.Handler, grpcServer *grpc.Server) (*http.Server, Runner, error) {
	var dispatch http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
	if grpcServer == nil {
		dispatch = h
	}

	srv := &http.Server{
		Addr:              c.HTTPAddr,
		Handler:           dispatch,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}

	run := func(ctx context.Context) error {
		se := make(chan error)

		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-c:
			case <-ctx.Done():
			}

			// call shutdown and wait for both
			gc := make(chan struct{})
			if grpcServer != nil {
				go func() {
					grpcServer.GracefulStop()
					close(gc)
				}()
			}
			err := srv.Shutdown(context.Background())
			if grpcServer != nil {
				<-gc
			}

			se <- err
		}()

		lg := c.Logger()
		var err error
		if c.TLSKeyFile != "" {
			lg.Info().Str("http-addr", c.HTTPAddr).Bool("tls", true).Msg("started http server")
			err = srv.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
		} else {
			lg.Info().Str("http-addr", c.HTTPAddr).Bool("tls", false).Msg("started http server")
			err = srv.ListenAndServe()
		}
		if errors.Is(err, http.ErrServerClosed) {
			return <-se
		}
		return err
	}

	return srv, run, nil
}
