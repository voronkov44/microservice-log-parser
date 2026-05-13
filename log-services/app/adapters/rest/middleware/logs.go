package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(log *slog.Logger) Middleware {
	if log == nil {
		log = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapper := &WrapperWriter{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapper, r)

			duration := time.Since(start)

			attrs := []any{
				"status", wrapper.StatusCode,
				"method", r.Method,
				"path", r.URL.Path,
				"bytes", wrapper.Bytes,
				"duration", duration.String(),
			}

			if query := r.URL.RawQuery; query != "" {
				attrs = append(attrs, "query", query)
			}

			switch {
			case wrapper.StatusCode >= 500:
				log.Error("http request failed", attrs...)
			case wrapper.StatusCode >= 400:
				log.Warn("http request failed", attrs...)
			default:
				log.Info("http request succeeded", attrs...)
			}
		})
	}
}
