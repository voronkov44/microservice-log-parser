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
	edges := buildEdges(nodes, ports)

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

func buildEdges(nodes []Node, ports []Port) []TopologyEdge {
	nodesByID := make(map[int64]Node, len(nodes))
	nodesByKind := make(map[string][]Node)

	for _, node := range nodes {
		kind := normalizeKind(node.NodeKind)

		nodesByID[node.ID] = node
		nodesByKind[kind] = append(nodesByKind[kind], node)
	}

	hostPorts := activePortsForKinds(ports, nodesByID, map[string]bool{
		"host": true,
	})

	switchPorts := activePortsForKinds(ports, nodesByID, map[string]bool{
		"switch": true,
	})

	sortPorts(hostPorts)
	sortPorts(switchPorts)

	edges := make([]TopologyEdge, 0)
	usedSwitchPorts := make(map[int64]bool)

	for _, hostPort := range hostPorts {
		switchPort, ok := pickSwitchPort(hostPort, switchPorts, usedSwitchPorts)
		if !ok {
			continue
		}

		sourceNode := nodesByID[hostPort.NodeID]
		targetNode := nodesByID[switchPort.NodeID]

		edges = append(edges, TopologyEdge{
			SourceNodeID:    sourceNode.ID,
			SourceNodeGUID:  sourceNode.NodeGUID,
			SourcePortNum:   hostPort.PortNum,
			SourcePortGUID:  hostPort.PortGUID,
			TargetNodeID:    targetNode.ID,
			TargetNodeGUID:  targetNode.NodeGUID,
			TargetPortNum:   switchPort.PortNum,
			TargetPortGUID:  switchPort.PortGUID,
			Relation:        "inferred_host_switch",
			LinkWidthActive: hostPort.LinkWidthActive,
			LinkSpeedActive: hostPort.LinkSpeedActive,
			PortState:       hostPort.PortState,
		})

		usedSwitchPorts[switchPort.ID] = true
	}

	switches := nodesByKind["switch"]
	sort.Slice(switches, func(i, j int) bool {
		return switches[i].ID < switches[j].ID
	})

	portsByNodeID := groupPortsByNodeID(switchPorts)

	for i := 0; i+1 < len(switches); i++ {
		sourceNode := switches[i]
		targetNode := switches[i+1]

		sourcePort, ok := firstUnusedPort(portsByNodeID[sourceNode.ID], usedSwitchPorts)
		if !ok {
			continue
		}

		usedSwitchPorts[sourcePort.ID] = true

		targetPort, ok := firstUnusedPort(portsByNodeID[targetNode.ID], usedSwitchPorts)
		if !ok {
			continue
		}

		usedSwitchPorts[targetPort.ID] = true

		edges = append(edges, TopologyEdge{
			SourceNodeID:    sourceNode.ID,
			SourceNodeGUID:  sourceNode.NodeGUID,
			SourcePortNum:   sourcePort.PortNum,
			SourcePortGUID:  sourcePort.PortGUID,
			TargetNodeID:    targetNode.ID,
			TargetNodeGUID:  targetNode.NodeGUID,
			TargetPortNum:   targetPort.PortNum,
			TargetPortGUID:  targetPort.PortGUID,
			Relation:        "inferred_switch_backbone",
			LinkWidthActive: sourcePort.LinkWidthActive,
			LinkSpeedActive: sourcePort.LinkSpeedActive,
			PortState:       sourcePort.PortState,
		})
	}

	return edges
}

func activePortsForKinds(ports []Port, nodesByID map[int64]Node, allowedKinds map[string]bool) []Port {
	out := make([]Port, 0)

	for _, port := range ports {
		if !isActivePort(port) {
			continue
		}

		node, ok := nodesByID[port.NodeID]
		if !ok {
			continue
		}

		kind := normalizeKind(node.NodeKind)
		if !allowedKinds[kind] {
			continue
		}

		out = append(out, port)
	}

	return out
}

func isActivePort(port Port) bool {
	return port.ID > 0 &&
		port.NodeID > 0 &&
		port.PortNum > 0 &&
		port.PortState == 4
}

func sortPorts(ports []Port) {
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].NodeID != ports[j].NodeID {
			return ports[i].NodeID < ports[j].NodeID
		}

		if ports[i].PortNum != ports[j].PortNum {
			return ports[i].PortNum < ports[j].PortNum
		}

		return ports[i].ID < ports[j].ID
	})
}

func pickSwitchPort(hostPort Port, switchPorts []Port, used map[int64]bool) (Port, bool) {
	for _, switchPort := range switchPorts {
		if used[switchPort.ID] {
			continue
		}

		if sameLinkCharacteristics(hostPort, switchPort) {
			return switchPort, true
		}
	}

	for _, switchPort := range switchPorts {
		if !used[switchPort.ID] {
			return switchPort, true
		}
	}

	return Port{}, false
}

func sameLinkCharacteristics(a Port, b Port) bool {
	if a.LinkWidthActive == 0 || b.LinkWidthActive == 0 {
		return false
	}

	if a.LinkSpeedActive == 0 || b.LinkSpeedActive == 0 {
		return false
	}

	return a.LinkWidthActive == b.LinkWidthActive &&
		a.LinkSpeedActive == b.LinkSpeedActive
}

func groupPortsByNodeID(ports []Port) map[int64][]Port {
	out := make(map[int64][]Port)

	for _, port := range ports {
		out[port.NodeID] = append(out[port.NodeID], port)
	}

	for nodeID := range out {
		sortPorts(out[nodeID])
	}

	return out
}

func firstUnusedPort(ports []Port, used map[int64]bool) (Port, bool) {
	for _, port := range ports {
		if !used[port.ID] {
			return port, true
		}
	}

	return Port{}, false
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
		return title(kind)
	}
}

func title(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + value[1:]
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
