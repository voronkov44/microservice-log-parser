package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

const (
	defaultPortsLimit = 100
	maxPortsLimit     = 500
)

func NewGetLogHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		logID, err := parseInt64PathValue(r, "log_id")
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		item, err := service.GetLog(ctx, logID)
		if err != nil {
			statusCode := httpStatusFromError(err)
			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		res.Json(w, storedLogToResponse(item), http.StatusOK)

		log.Info("get log handled",
			"log_id", logID,
			"duration", time.Since(start),
		)
	}
}

func NewGetNodesByLogHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		includeRaw := includeRaw(r)

		logID, err := parseInt64PathValue(r, "log_id")
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		nodes, err := service.GetNodesByLog(ctx, logID)
		if err != nil {
			statusCode := httpStatusFromError(err)
			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		response := storedNodesResponse{
			Count: len(nodes),
			Nodes: storedNodesToResponse(nodes, includeRaw),
		}

		res.Json(w, response, http.StatusOK)

		log.Info("get nodes by log handled",
			"log_id", logID,
			"count", len(nodes),
			"duration", time.Since(start),
		)
	}
}

func NewGetPortsByLogHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		includeRaw := includeRaw(r)

		limit, offset, err := parsePagination(r)
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		logID, err := parseInt64PathValue(r, "log_id")
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		ports, err := service.GetPortsByLog(ctx, logID)
		if err != nil {
			statusCode := httpStatusFromError(err)
			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		page := paginatePorts(ports, limit, offset)
		response := storedPortsResponse{
			Count:  len(page),
			Total:  len(ports),
			Limit:  limit,
			Offset: offset,
			Ports:  storedPortsToResponse(page, includeRaw),
		}

		res.Json(w, response, http.StatusOK)

		log.Info("get ports by log handled",
			"log_id", logID,
			"count", len(ports),
			"duration", time.Since(start),
		)
	}
}

func NewGetNodeHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		includeRaw := includeRaw(r)

		nodeID, err := parseInt64PathValue(r, "node_id")
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		node, err := service.GetNode(ctx, nodeID)
		if err != nil {
			statusCode := httpStatusFromError(err)
			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		res.Json(w, storedNodeToResponse(node, includeRaw), http.StatusOK)

		log.Info("get node handled",
			"node_id", nodeID,
			"duration", time.Since(start),
		)
	}
}

func NewGetPortsByNodeHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		includeRaw := includeRaw(r)

		limit, offset, err := parsePagination(r)
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		nodeID, err := parseInt64PathValue(r, "node_id")
		if err != nil {
			res.Json(w, errorResponse{Error: err.Error()}, httpStatusFromError(err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		ports, err := service.GetPortsByNode(ctx, nodeID)
		if err != nil {
			statusCode := httpStatusFromError(err)
			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		page := paginatePorts(ports, limit, offset)
		response := storedPortsResponse{
			Count:  len(page),
			Total:  len(ports),
			Limit:  limit,
			Offset: offset,
			Ports:  storedPortsToResponse(page, includeRaw),
		}

		res.Json(w, response, http.StatusOK)

		log.Info("get ports by node handled",
			"node_id", nodeID,
			"count", len(ports),
			"duration", time.Since(start),
		)
	}
}

func includeRaw(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_raw")), "true")
}

func parsePagination(r *http.Request) (int, int, error) {
	query := r.URL.Query()

	limit := defaultPortsLimit
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return 0, 0, fmt.Errorf("%w: limit must be positive integer", core.ErrBadArguments)
		}
		if value > maxPortsLimit {
			return 0, 0, fmt.Errorf("%w: limit must not exceed %d", core.ErrBadArguments, maxPortsLimit)
		}
		limit = value
	}

	offset := 0
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, fmt.Errorf("%w: offset must be non-negative integer", core.ErrBadArguments)
		}
		offset = value
	}

	return limit, offset, nil
}

func paginatePorts(ports []core.Port, limit int, offset int) []core.Port {
	if offset >= len(ports) {
		return []core.Port{}
	}

	end := offset + limit
	if end > len(ports) {
		end = len(ports)
	}

	return ports[offset:end]
}

func parseInt64PathValue(r *http.Request, name string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(name))
	if raw == "" {
		return 0, fmt.Errorf("%w: %s is required", core.ErrBadArguments, name)
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%w: %s must be positive integer", core.ErrBadArguments, name)
	}

	return value, nil
}

func storedLogToResponse(in core.Log) storedLogResponse {
	return storedLogResponse{
		ID:         in.ID,
		FilePath:   in.FilePath,
		Status:     string(in.Status),
		NodesCount: in.NodesCount,
		PortsCount: in.PortsCount,
		Error:      in.Error,
		UploadedAt: in.UploadedAt,
		ParsedAt:   in.ParsedAt,
	}
}

func storedNodesToResponse(in []core.Node, includeRaw bool) []storedNodeResponse {
	out := make([]storedNodeResponse, 0, len(in))

	for _, node := range in {
		out = append(out, storedNodeToResponse(node, includeRaw))
	}

	return out
}

func storedNodeToResponse(in core.Node, includeRaw bool) storedNodeResponse {
	rawJSON := ""
	if includeRaw {
		rawJSON = in.RawJSON
	}

	return storedNodeResponse{
		ID:              in.ID,
		LogID:           in.LogID,
		NodeGUID:        in.NodeGUID,
		NodeDesc:        in.NodeDesc,
		NodeType:        in.NodeType,
		NodeKind:        in.NodeKind,
		NumPorts:        in.NumPorts,
		ClassVersion:    in.ClassVersion,
		BaseVersion:     in.BaseVersion,
		SystemImageGUID: in.SystemImageGUID,
		PortGUID:        in.PortGUID,
		Info:            storedNodeInfoToResponse(in.Info, includeRaw),
		RawJSON:         rawJSON,
	}
}

func storedNodeInfoToResponse(in *core.NodeInfo, includeRaw bool) *storedNodeInfoResponse {
	if in == nil {
		return nil
	}

	rawJSON := ""
	if includeRaw {
		rawJSON = in.RawJSON
	}

	return &storedNodeInfoResponse{
		ID:           in.ID,
		NodeID:       in.NodeID,
		NodeGUID:     in.NodeGUID,
		SerialNumber: in.SerialNumber,
		PartNumber:   in.PartNumber,
		Revision:     in.Revision,
		ProductName:  in.ProductName,
		RawJSON:      rawJSON,
	}
}

func storedPortsToResponse(in []core.Port, includeRaw bool) []storedPortResponse {
	out := make([]storedPortResponse, 0, len(in))

	for _, port := range in {
		out = append(out, storedPortToResponse(port, includeRaw))
	}

	return out
}

func storedPortToResponse(in core.Port, includeRaw bool) storedPortResponse {
	rawJSON := ""
	if includeRaw {
		rawJSON = in.RawJSON
	}

	return storedPortResponse{
		ID:              in.ID,
		LogID:           in.LogID,
		NodeID:          in.NodeID,
		NodeGUID:        in.NodeGUID,
		PortGUID:        in.PortGUID,
		PortNum:         in.PortNum,
		LID:             in.LID,
		LocalPortNum:    in.LocalPortNum,
		PortState:       in.PortState,
		PortPhyState:    in.PortPhyState,
		LinkWidthActive: in.LinkWidthActive,
		LinkSpeedActive: in.LinkSpeedActive,
		RawJSON:         rawJSON,
	}
}
