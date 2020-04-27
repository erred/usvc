package usvc

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
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

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.Srv.TLSConfig = c
	}
}

func WithTLS(cert tls.Certificate) ServerOption {
	return func(s *Server) {
		s.Srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
			Certificates:             []tls.Certificate{cert},
		}
	}
}

func WithCORS(allowedMethods []string, allowedSuffix []string) ServerOption {
	allowedMeths := strings.Join(allowedMethods, ", ")
	meth := map[string]struct{}{}
	for _, m := range allowedMethods {
		meth[m] = struct{}{}
	}

	var allowAllOrigin bool
	if (len(allowedSuffix) == 1) && (allowedSuffix[0] == "*") {
		allowAllOrigin = true
	}
	as := func(o string) string {
		if allowAllOrigin {
			return "*"
		}
		for _, s := range allowedSuffix {
			if strings.HasSuffix(o, s) {
				return o
			}
		}
		return ""
	}

	return func(s *Server) {
		h := s.Srv.Handler
		s.Srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				o := as(r.Header.Get("origin"))
				if o == "" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", o)
				w.Header().Set("Access-Control-Allow-Methods", allowedMeths)
				w.Header().Set("Access-Control-Max-Age", "86400")
				if o == "*" {
					w.Header().Add("Vary", "Origin")
				}
			} else if _, ok := meth[r.Method]; !ok {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
