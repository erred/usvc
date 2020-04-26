package usvc

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Server struct {
	Log *zerolog.Logger
	Mux *http.ServeMux
	Srv *http.Server
}

func NewServer(cfg *Config) *Server {
	var lg zerolog.Logger
	switch cfg.LogFormat {
	case "console":
		lg = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    true,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	case "json":
		fallthrough
	default:
		lg = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
	lvl, _ := zerolog.ParseLevel(cfg.LogLevel)
	lg = lg.Level(lvl)

	mx := http.NewServeMux()

	return &Server{
		Log: &lg,
		Mux: mx,
		Srv: &http.Server{
			Addr:              net.JoinHostPort(cfg.Addr, cfg.Port),
			Handler:           mx,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
			ErrorLog:          log.New(lg, "", 0),
		},
	}
}

func (s *Server) Run() error {
	s.Log.Info().Str("addr", s.Srv.Addr).Msg("starting server")
	return s.Srv.ListenAndServe()
}
func (s *Server) Shutdown() error {
	s.Log.Info().Msg("stopping server")
	return s.Srv.Shutdown(SignalContext())
}

type ServerOption func(*Server)

func WithLiveliness(p string) ServerOption {
	return func(s *Server) {
		s.Mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
}
