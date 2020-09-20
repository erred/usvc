package usvc

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/api/metric"
)

// RunHTTP is a convenience function to run the server without modification
func (c Conf) RunHTTP(ctx context.Context, m *http.ServeMux) error {
	// run in shared mode
	if c.GRPCAddr != c.HTTPAddr {
		c.GRPCAddr = c.HTTPAddr
	}
	_, _, run, err := c.Server(m)
	if err != nil {
		return err
	}
	return run(ctx)
}

func httpMid(h http.Handler, log zerolog.Logger, latency metric.Int64ValueRecorder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metrics", "/health":
			h.ServeHTTP(w, r)
			return
		}

		t := time.Now()
		remote := r.Header.Get("x-forwarded-for")
		if remote == "" {
			remote = r.RemoteAddr
		}
		ua := r.Header.Get("user-agent")

		defer func() {
			d := time.Since(t)
			latency.Record(r.Context(), d.Milliseconds())
			log.Debug().
				Str("src", remote).
				Str("url", r.URL.String()).
				Str("user-agent", ua).
				Dur("dur", d).
				Msg("served")
		}()

		h.ServeHTTP(w, r)
	})
}

var healthOK = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func corsAllowAll(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodGet, http.MethodPost:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Max-Age", "86400")
			h.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}
