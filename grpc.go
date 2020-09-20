package usvc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/api/metric"
	"google.golang.org/grpc"
)

func grpcMid(log zerolog.Logger, latency metric.Int64ValueRecorder) grpc.ServerOption {
	return grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		t := time.Now()
		defer func() {
			d := time.Since(t)
			latency.Record(ctx, d.Milliseconds())

			log.Debug().
				Str("method", info.FullMethod).
				Dur("dur", d).
				Msg("served")
		}()

		return handler(ctx, req)
	})
}
