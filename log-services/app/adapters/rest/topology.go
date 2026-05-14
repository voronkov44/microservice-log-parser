package rest

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

func NewGetTopologyHandler(log *slog.Logger, service core.TopologyViewer, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		logID, err := strconv.ParseInt(r.PathValue("log_id"), 10, 64)
		if err != nil || logID <= 0 {
			res.Json(w, errorResponse{Error: "invalid log_id"}, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		topology, err := service.GetTopology(ctx, logID)
		if err != nil {
			statusCode := httpStatusFromError(err)

			log.Warn("get topology failed",
				"log_id", logID,
				"status_code", statusCode,
				"error", err,
			)

			res.Json(w, errorResponse{Error: err.Error()}, statusCode)
			return
		}

		res.Json(w, topologyResponseFromCore(topology), http.StatusOK)

		log.Info("get topology handled",
			"log_id", logID,
			"nodes", topology.Summary.NodesCount,
			"ports", topology.Summary.PortsCount,
			"edges", topology.Summary.EdgesCount,
			"duration", time.Since(start),
		)
	}
}

func topologyResponseFromCore(in core.Topology) topologyResponse {
	return topologyResponse{
		LogID:   in.LogID,
		Summary: topologySummaryResponseFromCore(in.Summary),
		Nodes:   topologyNodesResponseFromCore(in.Nodes),
		Groups:  topologyGroupsResponseFromCore(in.Groups),
		Edges:   topologyEdgesResponseFromCore(in.Edges),
	}
}

func topologySummaryResponseFromCore(in core.TopologySummary) topologySummaryResponse {
	return topologySummaryResponse{
		NodesCount:    in.NodesCount,
		PortsCount:    in.PortsCount,
		EdgesCount:    in.EdgesCount,
		HostsCount:    in.HostsCount,
		SwitchesCount: in.SwitchesCount,
	}
}

func topologyNodesResponseFromCore(in []core.TopologyNode) []topologyNodeResponse {
	out := make([]topologyNodeResponse, 0, len(in))

	for _, node := range in {
		out = append(out, topologyNodeResponse{
			ID:                 node.ID,
			LogID:              node.LogID,
			NodeGUID:           node.NodeGUID,
			NodeDesc:           node.NodeDesc,
			NodeType:           node.NodeType,
			NodeKind:           node.NodeKind,
			DeclaredPortsCount: node.DeclaredPortsCount,
			ParsedPortsCount:   node.ParsedPortsCount,
			SerialNumber:       node.SerialNumber,
			ProductName:        node.ProductName,
		})
	}

	return out
}

func topologyGroupsResponseFromCore(in []core.TopologyGroup) []topologyGroupResponse {
	out := make([]topologyGroupResponse, 0, len(in))

	for _, group := range in {
		out = append(out, topologyGroupResponse{
			Name:      group.Name,
			Kind:      group.Kind,
			NodeIDs:   group.NodeIDs,
			NodeGUIDs: group.NodeGUIDs,
		})
	}

	return out
}

func topologyEdgesResponseFromCore(in []core.TopologyEdge) []topologyEdgeResponse {
	out := make([]topologyEdgeResponse, 0, len(in))

	for _, edge := range in {
		out = append(out, topologyEdgeResponse{
			SourceNodeID:    edge.SourceNodeID,
			SourceNodeGUID:  edge.SourceNodeGUID,
			SourcePortNum:   edge.SourcePortNum,
			SourcePortGUID:  edge.SourcePortGUID,
			TargetNodeID:    edge.TargetNodeID,
			TargetNodeGUID:  edge.TargetNodeGUID,
			TargetPortNum:   edge.TargetPortNum,
			TargetPortGUID:  edge.TargetPortGUID,
			Relation:        edge.Relation,
			LinkWidthActive: edge.LinkWidthActive,
			LinkSpeedActive: edge.LinkSpeedActive,
			PortState:       edge.PortState,
		})
	}

	return out
}
