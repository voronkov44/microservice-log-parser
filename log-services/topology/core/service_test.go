package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func TestServiceGetTopologyRejectsBadLogID(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeRepository{})

	_, err := service.GetTopology(context.Background(), 0)
	if !errors.Is(err, ErrBadArguments) {
		t.Fatalf("GetTopology() error = %v, want ErrBadArguments", err)
	}
}

func TestServiceGetTopologyPropagatesGetLogError(t *testing.T) {
	wantErr := errors.New("repository failed")
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeRepository{getLogErr: wantErr})

	_, err := service.GetTopology(context.Background(), 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetTopology() error = %v, want %v", err, wantErr)
	}
}

func TestServiceGetTopologyBuildsGroupsSummaryAndEdges(t *testing.T) {
	repo := &fakeRepository{
		log: Log{ID: 1, Status: LogStatusParsed},
		nodes: []Node{
			{ID: 10, LogID: 1, NodeGUID: "host-1", NodeDesc: "Host A", NodeKind: "HOST", NumPorts: 2, Info: &NodeInfo{SerialNumber: "SN-host", ProductName: "Host Product"}},
			{ID: 20, LogID: 1, NodeGUID: "switch-1", NodeDesc: "Switch A", NodeKind: "switch", NumPorts: 2},
			{ID: 30, LogID: 1, NodeGUID: "switch-2", NodeDesc: "Switch B", NodeKind: "switch", NumPorts: 1},
			{ID: 40, LogID: 1, NodeGUID: "unknown-1", NodeDesc: "Unknown", NodeKind: "", NumPorts: 0},
			{ID: 50, LogID: 1, NodeGUID: "router-1", NodeDesc: "Router", NodeKind: "router", NumPorts: 0},
			{ID: 60, LogID: 1, NodeGUID: "accel-1", NodeDesc: "Accelerator", NodeKind: "accelerator", NumPorts: 0},
		},
		ports: []Port{
			{ID: 100, LogID: 1, NodeID: 10, NodeGUID: "host-1", PortGUID: "host-p1", PortNum: 1, PortState: 4, LinkWidthActive: 4, LinkSpeedActive: 100},
			{ID: 101, LogID: 1, NodeID: 10, NodeGUID: "host-1", PortGUID: "host-down", PortNum: 2, PortState: 1, LinkWidthActive: 4, LinkSpeedActive: 100},
			{ID: 200, LogID: 1, NodeID: 20, NodeGUID: "switch-1", PortGUID: "sw1-p1", PortNum: 1, PortState: 4, LinkWidthActive: 4, LinkSpeedActive: 100},
			{ID: 201, LogID: 1, NodeID: 20, NodeGUID: "switch-1", PortGUID: "sw1-p2", PortNum: 2, PortState: 4, LinkWidthActive: 8, LinkSpeedActive: 200},
			{ID: 300, LogID: 1, NodeID: 30, NodeGUID: "switch-2", PortGUID: "sw2-p1", PortNum: 1, PortState: 4, LinkWidthActive: 8, LinkSpeedActive: 200},
			{ID: 301, LogID: 1, NodeID: 30, NodeGUID: "switch-2", PortGUID: "sw2-down", PortNum: 2, PortState: 2, LinkWidthActive: 8, LinkSpeedActive: 200},
		},
	}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), repo)

	got, err := service.GetTopology(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTopology() error = %v", err)
	}

	wantSummary := TopologySummary{
		NodesCount:    6,
		PortsCount:    6,
		EdgesCount:    2,
		HostsCount:    1,
		SwitchesCount: 2,
	}
	if got.Summary != wantSummary {
		t.Fatalf("summary = %+v, want %+v", got.Summary, wantSummary)
	}

	if got.Nodes[0].NodeKind != "host" {
		t.Fatalf("node kind = %q, want normalized host", got.Nodes[0].NodeKind)
	}
	if got.Nodes[0].ParsedPortsCount != 2 {
		t.Fatalf("host parsed ports = %d, want 2", got.Nodes[0].ParsedPortsCount)
	}
	if got.Nodes[0].SerialNumber != "SN-host" || got.Nodes[0].ProductName != "Host Product" {
		t.Fatalf("host info not propagated: %+v", got.Nodes[0])
	}

	gotGroupNames := make([]string, 0, len(got.Groups))
	for _, group := range got.Groups {
		gotGroupNames = append(gotGroupNames, group.Name)
	}
	wantGroupNames := []string{"Hosts", "Switches", "Unknown", "Accelerator", "Router"}
	if !reflect.DeepEqual(gotGroupNames, wantGroupNames) {
		t.Fatalf("group order = %v, want %v", gotGroupNames, wantGroupNames)
	}

	if len(got.Edges) != 2 {
		t.Fatalf("edges count = %d, want 2", len(got.Edges))
	}

	hostEdge := got.Edges[0]
	if hostEdge.SourceNodeID != 10 || hostEdge.TargetNodeID != 20 || hostEdge.Relation != "inferred_host_switch" {
		t.Fatalf("host edge = %+v, want host-1 -> switch-1 inferred_host_switch", hostEdge)
	}
	if hostEdge.PortState != 4 || hostEdge.LinkWidthActive != 4 || hostEdge.LinkSpeedActive != 100 {
		t.Fatalf("host edge link fields = %+v", hostEdge)
	}

	switchEdge := got.Edges[1]
	if switchEdge.SourceNodeID != 20 || switchEdge.TargetNodeID != 30 || switchEdge.Relation != "inferred_switch_backbone" {
		t.Fatalf("switch edge = %+v, want switch-1 -> switch-2 inferred_switch_backbone", switchEdge)
	}
}

func TestServiceGetTopologyNoActivePortsProducesNoEdges(t *testing.T) {
	repo := &fakeRepository{
		log: Log{ID: 1, Status: LogStatusParsed},
		nodes: []Node{
			{ID: 10, LogID: 1, NodeGUID: "host-1", NodeKind: "host"},
			{ID: 20, LogID: 1, NodeGUID: "switch-1", NodeKind: "switch"},
		},
		ports: []Port{
			{ID: 100, LogID: 1, NodeID: 10, NodeGUID: "host-1", PortNum: 1, PortState: 1},
			{ID: 200, LogID: 1, NodeID: 20, NodeGUID: "switch-1", PortNum: 1, PortState: 2},
		},
	}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), repo)

	got, err := service.GetTopology(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTopology() error = %v", err)
	}

	if len(got.Edges) != 0 {
		t.Fatalf("edges = %+v, want empty edges", got.Edges)
	}
}

func TestNormalizeKindGroupNameAndTitle(t *testing.T) {
	tests := []struct {
		kind      string
		wantKind  string
		wantGroup string
	}{
		{kind: " HOST ", wantKind: "host", wantGroup: "Hosts"},
		{kind: "switch", wantKind: "switch", wantGroup: "Switches"},
		{kind: "", wantKind: "unknown", wantGroup: "Unknown"},
		{kind: "router", wantKind: "router", wantGroup: "Router"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			gotKind := normalizeKind(tt.kind)
			if gotKind != tt.wantKind {
				t.Fatalf("normalizeKind() = %q, want %q", gotKind, tt.wantKind)
			}

			gotGroup := groupName(gotKind)
			if gotGroup != tt.wantGroup {
				t.Fatalf("groupName() = %q, want %q", gotGroup, tt.wantGroup)
			}
		})
	}
}

type fakeRepository struct {
	log     Log
	nodes   []Node
	ports   []Port
	pingErr error

	getLogErr        error
	getNodesByLogErr error
	getPortsByLogErr error
}

func (f *fakeRepository) Ping(context.Context) error {
	return f.pingErr
}

func (f *fakeRepository) GetLog(context.Context, int64) (Log, error) {
	return f.log, f.getLogErr
}

func (f *fakeRepository) GetNodesByLog(context.Context, int64) ([]Node, error) {
	return f.nodes, f.getNodesByLogErr
}

func (f *fakeRepository) GetPortsByLog(context.Context, int64) ([]Port, error) {
	return f.ports, f.getPortsByLogErr
}
