package core

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

type Service struct {
	log        *slog.Logger
	repository Repository
}

func NewService(log *slog.Logger, repository Repository) *Service {
	return &Service{
		log:        log,
		repository: repository,
	}
}

func (s *Service) Ping(ctx context.Context) error {
	return s.repository.Ping(ctx)
}

func (s *Service) GetTopology(ctx context.Context, logID int64) (TopologyResult, error) {
	if logID <= 0 {
		return TopologyResult{}, ErrBadArguments
	}

	if _, err := s.repository.GetLog(ctx, logID); err != nil {
		return TopologyResult{}, err
	}

	nodes, err := s.repository.GetNodesByLog(ctx, logID)
	if err != nil {
		return TopologyResult{}, err
	}

	ports, err := s.repository.GetPortsByLog(ctx, logID)
	if err != nil {
		return TopologyResult{}, err
	}

	parsedPortsByNodeID := make(map[int64]int32)
	parsedPortsByNodeGUID := make(map[string]int32)

	for _, port := range ports {
		if port.NodeID > 0 {
			parsedPortsByNodeID[port.NodeID]++
		}
		if port.NodeGUID != "" {
			parsedPortsByNodeGUID[port.NodeGUID]++
		}
	}

	topologyNodes := make([]TopologyNode, 0, len(nodes))
	groupsByKind := make(map[string]*TopologyGroup)

	var hostsCount int32
	var switchesCount int32

	for _, node := range nodes {
		kind := normalizeKind(node.NodeKind)

		switch kind {
		case "host":
			hostsCount++
		case "switch":
			switchesCount++
		}

		parsedPortsCount := parsedPortsByNodeID[node.ID]
		if parsedPortsCount == 0 {
			parsedPortsCount = parsedPortsByNodeGUID[node.NodeGUID]
		}

		topologyNode := TopologyNode{
			ID:                 node.ID,
			LogID:              node.LogID,
			NodeGUID:           node.NodeGUID,
			NodeDesc:           node.NodeDesc,
			NodeType:           node.NodeType,
			NodeKind:           kind,
			DeclaredPortsCount: node.NumPorts,
			ParsedPortsCount:   parsedPortsCount,
		}

		if node.Info != nil {
			topologyNode.SerialNumber = node.Info.SerialNumber
			topologyNode.ProductName = node.Info.ProductName
		}

		topologyNodes = append(topologyNodes, topologyNode)

		group := groupsByKind[kind]
		if group == nil {
			group = &TopologyGroup{
				Name: groupName(kind),
				Kind: kind,
			}
			groupsByKind[kind] = group
		}

		group.NodeIDs = append(group.NodeIDs, node.ID)
		group.NodeGUIDs = append(group.NodeGUIDs, node.NodeGUID)
	}

	groups := groupsFromMap(groupsByKind)

	// пока что пустой
	edges := make([]TopologyEdge, 0)

	result := TopologyResult{
		LogID: logID,
		Summary: TopologySummary{
			NodesCount:    int32(len(nodes)),
			PortsCount:    int32(len(ports)),
			EdgesCount:    int32(len(edges)),
			HostsCount:    hostsCount,
			SwitchesCount: switchesCount,
		},
		Nodes:  topologyNodes,
		Groups: groups,
		Edges:  edges,
	}

	s.log.Info("topology built",
		"log_id", logID,
		"nodes", result.Summary.NodesCount,
		"ports", result.Summary.PortsCount,
		"edges", result.Summary.EdgesCount,
		"hosts", result.Summary.HostsCount,
		"switches", result.Summary.SwitchesCount,
	)

	return result, nil
}

func normalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return "unknown"
	}

	return kind
}

func groupName(kind string) string {
	switch kind {
	case "host":
		return "Hosts"
	case "switch":
		return "Switches"
	case "unknown":
		return "Unknown"
	default:
		return strings.Title(kind)
	}
}

func groupsFromMap(in map[string]*TopologyGroup) []TopologyGroup {
	order := []string{"host", "switch", "unknown"}

	used := make(map[string]bool, len(in))
	out := make([]TopologyGroup, 0, len(in))

	for _, kind := range order {
		group, ok := in[kind]
		if !ok {
			continue
		}

		out = append(out, *group)
		used[kind] = true
	}

	var rest []string
	for kind := range in {
		if !used[kind] {
			rest = append(rest, kind)
		}
	}

	sort.Strings(rest)

	for _, kind := range rest {
		out = append(out, *in[kind])
	}

	return out
}
