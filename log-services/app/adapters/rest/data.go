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
			Nodes: storedNodesToResponse(nodes),
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

		response := storedPortsResponse{
			Count: len(ports),
			Ports: storedPortsToResponse(ports),
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

		res.Json(w, storedNodeToResponse(node), http.StatusOK)

		log.Info("get node handled",
			"node_id", nodeID,
			"duration", time.Since(start),
		)
	}
}

func NewGetPortsByNodeHandler(log *slog.Logger, service core.LogReader, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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

		response := storedPortsResponse{
			Count: len(ports),
			Ports: storedPortsToResponse(ports),
		}

		res.Json(w, response, http.StatusOK)

		log.Info("get ports by node handled",
			"node_id", nodeID,
			"count", len(ports),
			"duration", time.Since(start),
		)
	}
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
		ID:             in.ID,
		FilePath:       in.FilePath,
		Status:         string(in.Status),
		NodesCount:     in.NodesCount,
		PortsCount:     in.PortsCount,
		Error:          in.Error,
		UploadedAtUnix: in.UploadedAtUnix,
		ParsedAtUnix:   in.ParsedAtUnix,
	}
}

func storedNodesToResponse(in []core.Node) []storedNodeResponse {
	out := make([]storedNodeResponse, 0, len(in))

	for _, node := range in {
		out = append(out, storedNodeToResponse(node))
	}

	return out
}

func storedNodeToResponse(in core.Node) storedNodeResponse {
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
		Info:            storedNodeInfoToResponse(in.Info),
		RawJSON:         in.RawJSON,
	}
}

func storedNodeInfoToResponse(in *core.NodeInfo) *storedNodeInfoResponse {
	if in == nil {
		return nil
	}

	return &storedNodeInfoResponse{
		ID:           in.ID,
		NodeID:       in.NodeID,
		NodeGUID:     in.NodeGUID,
		SerialNumber: in.SerialNumber,
		PartNumber:   in.PartNumber,
		Revision:     in.Revision,
		ProductName:  in.ProductName,
		RawJSON:      in.RawJSON,
	}
}

func storedPortsToResponse(in []core.Port) []storedPortResponse {
	out := make([]storedPortResponse, 0, len(in))

	for _, port := range in {
		out = append(out, storedPortToResponse(port))
	}

	return out
}

func storedPortToResponse(in core.Port) storedPortResponse {
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
		RawJSON:         in.RawJSON,
	}
}
