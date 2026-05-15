package rest

import (
	"context"
	"errors"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/req"
	"log/slog"
	"net/http"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

func NewParseLogHandler(log *slog.Logger, service core.LogParser, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		payload, err := req.HandleBody[parseLogRequest](w, r)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		result, err := service.ParseLog(ctx, payload.Path)
		if err != nil {
			statusCode := httpStatusFromError(err)

			log.Warn("parse log failed",
				"path", payload.Path,
				"status_code", statusCode,
				"error", err,
			)

			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		response := parseLogResponse{
			LogID:          result.LogID,
			Status:         string(result.Status),
			NodesCount:     result.NodesCount,
			PortsCount:     result.PortsCount,
			NodesInfoCount: result.NodesInfoCount,
		}

		res.Json(w, response, http.StatusOK)

		log.Info("parse log handled",
			"log_id", result.LogID,
			"path", payload.Path,
			"nodes", result.NodesCount,
			"ports", result.PortsCount,
			"nodes_info", result.NodesInfoCount,
			"duration", time.Since(start),
		)
	}
}

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, core.ErrBadArguments):
		return http.StatusBadRequest
	case errors.Is(err, core.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, core.ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
