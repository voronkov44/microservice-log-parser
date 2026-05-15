//go:build integration

package db

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/repository/core"
)

func TestStorageIntegrationLogLifecycle(t *testing.T) {
	storage := openIntegrationDB(t)
	ctx := context.Background()

	logID, err := storage.CreateLog(ctx, "log.zip")
	if err != nil {
		t.Fatalf("CreateLog() error = %v", err)
	}

	created, err := storage.GetLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetLog(created) error = %v", err)
	}
	if created.Status != core.LogStatusProcessing {
		t.Fatalf("created status = %q, want processing", created.Status)
	}
	if created.UploadedAt == "" {
		t.Fatalf("created uploaded_at is empty")
	}
	assertRFC3339(t, created.UploadedAt)

	parsed := core.ParsedLog{
		Nodes: []core.Node{
			{NodeGUID: "node-1", NodeDesc: "Host", NodeKind: "host", NodeType: 1, NumPorts: 1, RawJSON: `{"node":"1"}`},
			{NodeGUID: "node-2", NodeDesc: "Switch", NodeKind: "switch", NodeType: 2, NumPorts: 1, RawJSON: `{"node":"2"}`},
		},
		NodesInfo: []core.NodeInfo{
			{NodeGUID: "node-1", SerialNumber: "SN-1", PartNumber: "PN-1", Revision: "A1", ProductName: "Host Product"},
		},
		Ports: []core.Port{
			{NodeGUID: "node-1", PortGUID: "port-1", PortNum: 1, LID: 11, LocalPortNum: 1, PortState: 4, PortPhyState: 5, LinkWidthActive: 4, LinkSpeedActive: 100},
			{NodeGUID: "node-2", PortGUID: "port-2", PortNum: 1, LID: 22, LocalPortNum: 1, PortState: 4, PortPhyState: 5, LinkWidthActive: 4, LinkSpeedActive: 100},
		},
	}

	result, err := storage.SaveParsedLog(ctx, logID, parsed)
	if err != nil {
		t.Fatalf("SaveParsedLog() error = %v", err)
	}
	if result.LogID != logID || result.NodesCount != 2 || result.PortsCount != 2 {
		t.Fatalf("SaveParsedLog() = %+v", result)
	}

	saved, err := storage.GetLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetLog(saved) error = %v", err)
	}
	if saved.Status != core.LogStatusParsed || saved.NodesCount != 2 || saved.PortsCount != 2 {
		t.Fatalf("saved log = %+v", saved)
	}
	if saved.ParsedAt == "" {
		t.Fatalf("parsed_at is empty")
	}
	assertRFC3339(t, saved.UploadedAt)
	assertRFC3339(t, saved.ParsedAt)

	nodes, err := storage.GetNodesByLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetNodesByLog() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes count = %d, want 2", len(nodes))
	}
	if nodes[0].Info == nil || nodes[0].Info.SerialNumber != "SN-1" || nodes[0].Info.ProductName != "Host Product" {
		t.Fatalf("node info = %+v", nodes[0].Info)
	}

	node, err := storage.GetNode(ctx, nodes[0].ID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if node.Info == nil || node.Info.PartNumber != "PN-1" {
		t.Fatalf("GetNode() info = %+v", node.Info)
	}

	nodePorts, err := storage.GetPortsByNode(ctx, nodes[0].ID)
	if err != nil {
		t.Fatalf("GetPortsByNode() error = %v", err)
	}
	if len(nodePorts) != 1 || nodePorts[0].PortGUID != "port-1" {
		t.Fatalf("node ports = %+v", nodePorts)
	}

	logPorts, err := storage.GetPortsByLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetPortsByLog() error = %v", err)
	}
	if len(logPorts) != 2 {
		t.Fatalf("log ports count = %d, want 2", len(logPorts))
	}

	replacement := core.ParsedLog{
		Nodes: []core.Node{{NodeGUID: "node-3", NodeDesc: "Replacement", NodeKind: "host"}},
		Ports: []core.Port{{NodeGUID: "node-3", PortGUID: "port-3", PortNum: 1, PortState: 4}},
	}
	if _, err := storage.SaveParsedLog(ctx, logID, replacement); err != nil {
		t.Fatalf("SaveParsedLog(replacement) error = %v", err)
	}

	replacedNodes, err := storage.GetNodesByLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetNodesByLog(replaced) error = %v", err)
	}
	if len(replacedNodes) != 1 || replacedNodes[0].NodeGUID != "node-3" {
		t.Fatalf("replaced nodes = %+v", replacedNodes)
	}

	replacedPorts, err := storage.GetPortsByLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetPortsByLog(replaced) error = %v", err)
	}
	if len(replacedPorts) != 1 || replacedPorts[0].PortGUID != "port-3" {
		t.Fatalf("replaced ports = %+v", replacedPorts)
	}

	if err := storage.FailLog(ctx, logID, "broken log"); err != nil {
		t.Fatalf("FailLog() error = %v", err)
	}

	failed, err := storage.GetLog(ctx, logID)
	if err != nil {
		t.Fatalf("GetLog(failed) error = %v", err)
	}
	if failed.Status != core.LogStatusFailed || failed.Error != "broken log" {
		t.Fatalf("failed log = %+v", failed)
	}
}

func TestStorageIntegrationNotFound(t *testing.T) {
	storage := openIntegrationDB(t)
	ctx := context.Background()

	if _, err := storage.GetLog(ctx, 999999); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("GetLog() error = %v, want ErrNotFound", err)
	}

	if _, err := storage.GetNode(ctx, 999999); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("GetNode() error = %v, want ErrNotFound", err)
	}

	if _, err := storage.SaveParsedLog(ctx, 999999, core.ParsedLog{}); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("SaveParsedLog() error = %v, want ErrNotFound", err)
	}

	if err := storage.FailLog(ctx, 999999, "missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("FailLog() error = %v, want ErrNotFound", err)
	}
}

func openIntegrationDB(t *testing.T) *DB {
	t.Helper()

	address := os.Getenv("TEST_DATABASE_URL")
	if address == "" {
		t.Skip("TEST_DATABASE_URL is empty")
	}

	storage, err := New(slog.New(slog.NewTextHandler(os.Stderr, nil)), address)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	if err := storage.Migrate(); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	cleanIntegrationDB(t, storage)
	t.Cleanup(func() {
		cleanIntegrationDB(t, storage)
	})

	return storage
}

func cleanIntegrationDB(t *testing.T, storage *DB) {
	t.Helper()

	_, err := storage.conn.ExecContext(context.Background(), `
		TRUNCATE TABLE ports, nodes_info, nodes, logs RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("clean test db: %v", err)
	}
}

func assertRFC3339(t *testing.T, value string) {
	t.Helper()

	if _, err := time.Parse(time.RFC3339, value); err != nil {
		t.Fatalf("timestamp %q is not RFC3339: %v", value, err)
	}
}
