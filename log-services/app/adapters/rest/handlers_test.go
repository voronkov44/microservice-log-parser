package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
)

func TestParseLogHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "valid body", body: `{"path":"log.zip"}`, wantStatus: http.StatusOK},
		{name: "invalid json", body: `{"path":`, wantStatus: http.StatusBadRequest},
		{name: "empty path", body: `{"path":""}`, serviceErr: core.ErrBadArguments, wantStatus: http.StatusBadRequest},
		{name: "conflict", body: `{"path":"log.zip"}`, serviceErr: core.ErrConflict, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAppService{
				parseLogResult: core.ParseLogResult{
					LogID:          101,
					Status:         core.LogStatusParsed,
					NodesCount:     2,
					PortsCount:     3,
					NodesInfoCount: 1,
				},
				parseLogErr: tt.serviceErr,
			}
			handler := NewParseLogHandler(testLogger(), service, time.Second)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parse", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var resp parseLogResponse
				decodeJSON(t, rec.Body.String(), &resp)
				if resp.LogID != 101 || resp.Status != string(core.LogStatusParsed) {
					t.Fatalf("response = %+v", resp)
				}
				if service.parseLogPath != "log.zip" {
					t.Fatalf("ParseLog path = %q, want log.zip", service.parseLogPath)
				}
			}
		})
	}
}

func TestParseLogHandlerRejectsLargeBody(t *testing.T) {
	service := &fakeAppService{}
	handler := NewParseLogHandler(testLogger(), service, time.Second)
	body := `{"path":"` + strings.Repeat("x", parseRequestBodyLimit) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parse", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func TestGetHandlersSuccess(t *testing.T) {
	service := &fakeAppService{
		log: core.Log{
			ID:         101,
			FilePath:   "log.zip",
			Status:     core.LogStatusParsed,
			NodesCount: 1,
			PortsCount: 1,
			UploadedAt: "2026-05-15T00:00:00Z",
		},
		node: core.Node{
			ID:       201,
			LogID:    101,
			NodeGUID: "node-1",
			NodeKind: "host",
		},
		ports: []core.Port{{ID: 301, NodeID: 201, NodeGUID: "node-1", PortNum: 1}},
		topology: core.Topology{
			LogID:   101,
			Summary: core.TopologySummary{NodesCount: 1, PortsCount: 1},
			Nodes:   []core.TopologyNode{{ID: 201, NodeGUID: "node-1"}},
			Groups:  []core.TopologyGroup{{Name: "Hosts", Kind: "host", NodeIDs: []int64{201}}},
		},
	}

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		pathValue  string
		pathName   string
		wantStatus int
		assert     func(*testing.T, string)
	}{
		{
			name:       "get log",
			handler:    NewGetLogHandler(testLogger(), service, time.Second),
			pathValue:  "101",
			pathName:   "log_id",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body string) {
				var resp storedLogResponse
				decodeJSON(t, body, &resp)
				if resp.ID != 101 || resp.Status != string(core.LogStatusParsed) {
					t.Fatalf("log response = %+v", resp)
				}
			},
		},
		{
			name:       "get node",
			handler:    NewGetNodeHandler(testLogger(), service, time.Second),
			pathValue:  "201",
			pathName:   "node_id",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body string) {
				var resp storedNodeResponse
				decodeJSON(t, body, &resp)
				if resp.ID != 201 || resp.NodeGUID != "node-1" {
					t.Fatalf("node response = %+v", resp)
				}
			},
		},
		{
			name:       "get ports by node",
			handler:    NewGetPortsByNodeHandler(testLogger(), service, time.Second),
			pathValue:  "201",
			pathName:   "node_id",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body string) {
				var resp storedPortsResponse
				decodeJSON(t, body, &resp)
				if resp.Count != 1 || resp.Ports[0].NodeID != 201 {
					t.Fatalf("ports response = %+v", resp)
				}
			},
		},
		{
			name:       "get topology",
			handler:    NewGetTopologyHandler(testLogger(), service, time.Second),
			pathValue:  "101",
			pathName:   "log_id",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body string) {
				var resp topologyResponse
				decodeJSON(t, body, &resp)
				if resp.LogID != 101 || resp.Summary.NodesCount != 1 || len(resp.Groups) != 1 {
					t.Fatalf("topology response = %+v", resp)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue(tt.pathName, tt.pathValue)
			rec := httptest.NewRecorder()

			tt.handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			tt.assert(t, rec.Body.String())
		})
	}
}

func TestRawJSONRequiresIncludeRaw(t *testing.T) {
	service := &fakeAppService{
		node: core.Node{
			ID:       201,
			LogID:    101,
			NodeGUID: "node-1",
			RawJSON:  `{"node":"raw"}`,
			Info: &core.NodeInfo{
				ID:      301,
				NodeID:  201,
				RawJSON: `{"info":"raw"}`,
			},
		},
	}
	handler := NewGetNodeHandler(testLogger(), service, time.Second)

	tests := []struct {
		name        string
		target      string
		wantRawJSON bool
	}{
		{name: "default omits raw", target: "/", wantRawJSON: false},
		{name: "include raw", target: "/?include_raw=true", wantRawJSON: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.SetPathValue("node_id", "201")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
			}

			var resp map[string]any
			decodeJSON(t, rec.Body.String(), &resp)
			_, hasRaw := resp["raw_json"]
			if hasRaw != tt.wantRawJSON {
				t.Fatalf("raw_json present = %v, want %v; body = %s", hasRaw, tt.wantRawJSON, rec.Body.String())
			}
		})
	}
}

func TestPortsPaginationAndRawJSON(t *testing.T) {
	service := &fakeAppService{
		node: core.Node{ID: 201},
		ports: []core.Port{
			{ID: 1, NodeID: 201, NodeGUID: "node-1", PortNum: 1, RawJSON: `{"port":1}`},
			{ID: 2, NodeID: 201, NodeGUID: "node-1", PortNum: 2, RawJSON: `{"port":2}`},
			{ID: 3, NodeID: 201, NodeGUID: "node-1", PortNum: 3, RawJSON: `{"port":3}`},
		},
	}
	handler := NewGetPortsByNodeHandler(testLogger(), service, time.Second)

	req := httptest.NewRequest(http.MethodGet, "/?limit=2&offset=1", nil)
	req.SetPathValue("node_id", "201")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}

	var resp storedPortsResponse
	decodeJSON(t, rec.Body.String(), &resp)
	if resp.Count != 2 || resp.Total != 3 || resp.Limit != 2 || resp.Offset != 1 {
		t.Fatalf("pagination response = %+v", resp)
	}
	if resp.Ports[0].ID != 2 || resp.Ports[0].RawJSON != "" {
		t.Fatalf("ports response = %+v", resp.Ports)
	}

	req = httptest.NewRequest(http.MethodGet, "/?limit=1&include_raw=true", nil)
	req.SetPathValue("node_id", "201")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec.Body.String(), &resp)
	if resp.Count != 1 || resp.Ports[0].RawJSON == "" {
		t.Fatalf("include_raw response = %+v", resp)
	}

	req = httptest.NewRequest(http.MethodGet, "/?limit=bad", nil)
	req.SetPathValue("node_id", "201")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandlersRejectInvalidIDs(t *testing.T) {
	service := &fakeAppService{}

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		pathName  string
		pathValue string
	}{
		{name: "log id", handler: NewGetLogHandler(testLogger(), service, time.Second), pathName: "log_id", pathValue: "0"},
		{name: "node id", handler: NewGetNodeHandler(testLogger(), service, time.Second), pathName: "node_id", pathValue: "bad"},
		{name: "port node id", handler: NewGetPortsByNodeHandler(testLogger(), service, time.Second), pathName: "node_id", pathValue: "-1"},
		{name: "topology log id", handler: NewGetTopologyHandler(testLogger(), service, time.Second), pathName: "log_id", pathValue: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue(tt.pathName, tt.pathValue)
			rec := httptest.NewRecorder()

			tt.handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlersMapServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: core.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict", err: core.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unavailable", err: core.ErrUnavailable, wantStatus: http.StatusServiceUnavailable},
		{name: "unknown", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAppService{getLogErr: tt.err}
			handler := NewGetLogHandler(testLogger(), service, time.Second)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/log/101", nil)
			req.SetPathValue("log_id", "101")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

type fakeAppService struct {
	parseLogPath   string
	parseLogResult core.ParseLogResult
	parseLogErr    error

	log       core.Log
	getLogErr error

	node       core.Node
	getNodeErr error

	ports             []core.Port
	getPortsByNodeErr error
	nodes             []core.Node
	getNodesByLogErr  error
	getPortsByLogErr  error

	topology       core.Topology
	getTopologyErr error
}

func (f *fakeAppService) ParseLog(_ context.Context, path string) (core.ParseLogResult, error) {
	f.parseLogPath = path
	return f.parseLogResult, f.parseLogErr
}

func (f *fakeAppService) GetLog(context.Context, int64) (core.Log, error) {
	return f.log, f.getLogErr
}

func (f *fakeAppService) GetNode(context.Context, int64) (core.Node, error) {
	return f.node, f.getNodeErr
}

func (f *fakeAppService) GetPortsByNode(context.Context, int64) ([]core.Port, error) {
	return f.ports, f.getPortsByNodeErr
}

func (f *fakeAppService) GetNodesByLog(context.Context, int64) ([]core.Node, error) {
	return f.nodes, f.getNodesByLogErr
}

func (f *fakeAppService) GetPortsByLog(context.Context, int64) ([]core.Port, error) {
	return f.ports, f.getPortsByLogErr
}

func (f *fakeAppService) GetTopology(context.Context, int64) (core.Topology, error) {
	return f.topology, f.getTopologyErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func decodeJSON(t *testing.T, body string, out any) {
	t.Helper()

	if err := json.Unmarshal([]byte(body), out); err != nil {
		t.Fatalf("decode response %q: %v", body, err)
	}
}
