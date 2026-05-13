package rest

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

func NewHealthHandler(log *slog.Logger, pingers map[string]core.Pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		replies := make(map[string]string, len(pingers))

		statusCode := http.StatusOK

		for name, pinger := range pingers {
			if err := pinger.Ping(ctx); err != nil {
				replies[name] = "unavailable"
				statusCode = http.StatusServiceUnavailable

				log.Warn("ping failed",
					"service", name,
					"error", err,
				)

				continue
			}

			replies[name] = "ok"
		}

		res.Json(w, healthResponse{Replies: replies}, statusCode)

		log.Info("healthz handled",
			"replies", replies,
			"duration", time.Since(start),
		)
	}
}
