package smoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	wantSampleNodes     int32 = 5
	wantSamplePorts     int32 = 151
	wantSampleNodesInfo int32 = 4
	wantSampleHosts     int32 = 1
	wantSampleSwitches  int32 = 4
	wantSampleEdges     int32 = 4
)

func TestSmokeLogParserLifecycle(t *testing.T) {
	baseURL := testBaseURL()
	client := testHTTPClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waitHealthz(t, ctx, client, baseURL)

	parseResp := postParse(t, ctx, client, baseURL, "log.zip")
	if parseResp.LogID <= 0 {
		t.Fatalf("log_id = %d, want positive id", parseResp.LogID)
	}
	if parseResp.Status != "parsed" {
		t.Fatalf("status = %q, want parsed", parseResp.Status)
	}
	if parseResp.NodesCount != wantSampleNodes {
		t.Fatalf("nodes_count = %d, want %d", parseResp.NodesCount, wantSampleNodes)
	}
	if parseResp.PortsCount != wantSamplePorts {
		t.Fatalf("ports_count = %d, want %d", parseResp.PortsCount, wantSamplePorts)
	}
	if parseResp.NodesInfoCount != wantSampleNodesInfo {
		t.Fatalf("nodes_info_count = %d, want %d", parseResp.NodesInfoCount, wantSampleNodesInfo)
	}

	logResp := getLog(t, ctx, client, baseURL, parseResp.LogID)
	if logResp.ID != parseResp.LogID {
		t.Fatalf("log id = %d, want %d", logResp.ID, parseResp.LogID)
	}
	if logResp.Status != "parsed" {
		t.Fatalf("log status = %q, want parsed", logResp.Status)
	}
	if logResp.NodesCount != wantSampleNodes {
		t.Fatalf("log nodes_count = %d, want %d", logResp.NodesCount, wantSampleNodes)
	}
	if logResp.PortsCount != wantSamplePorts {
		t.Fatalf("log ports_count = %d, want %d", logResp.PortsCount, wantSamplePorts)
	}
	assertRFC3339(t, "uploaded_at", logResp.UploadedAt)
	assertRFC3339(t, "parsed_at", logResp.ParsedAt)

	topologyResp := getTopology(t, ctx, client, baseURL, parseResp.LogID)
	if topologyResp.LogID != parseResp.LogID {
		t.Fatalf("topology log_id = %d, want %d", topologyResp.LogID, parseResp.LogID)
	}
	if topologyResp.Summary.NodesCount != wantSampleNodes {
		t.Fatalf("topology nodes_count = %d, want %d", topologyResp.Summary.NodesCount, wantSampleNodes)
	}
	if topologyResp.Summary.PortsCount != wantSamplePorts {
		t.Fatalf("topology ports_count = %d, want %d", topologyResp.Summary.PortsCount, wantSamplePorts)
	}
	if topologyResp.Summary.HostsCount != wantSampleHosts {
		t.Fatalf("topology hosts_count = %d, want %d", topologyResp.Summary.HostsCount, wantSampleHosts)
	}
	if topologyResp.Summary.SwitchesCount != wantSampleSwitches {
		t.Fatalf("topology switches_count = %d, want %d", topologyResp.Summary.SwitchesCount, wantSampleSwitches)
	}
	if topologyResp.Summary.EdgesCount != wantSampleEdges {
		t.Fatalf("topology edges_count = %d, want %d", topologyResp.Summary.EdgesCount, wantSampleEdges)
	}
	if len(topologyResp.Nodes) != int(wantSampleNodes) {
		t.Fatalf("topology nodes len = %d, want %d", len(topologyResp.Nodes), wantSampleNodes)
	}
	if len(topologyResp.Groups) == 0 {
		t.Fatalf("topology groups is empty")
	}
	assertGroupsContain(t, topologyResp.Groups, "host", "switch")
	if len(topologyResp.Edges) == 0 {
		t.Fatalf("topology edges is empty")
	}
	for _, edge := range topologyResp.Edges {
		if edge.Relation == "" {
			t.Fatalf("topology edge relation is empty: %+v", edge)
		}
	}

	nodeID := firstNodeWithPorts(t, topologyResp.Nodes)
	nodeResp := getNode(t, ctx, client, baseURL, nodeID)
	if nodeResp.ID != nodeID {
		t.Fatalf("node id = %d, want %d", nodeResp.ID, nodeID)
	}
	if nodeResp.NodeGUID == "" {
		t.Fatalf("node_guid is empty")
	}

	portsResp := getPortsByNode(t, ctx, client, baseURL, nodeID)
	if portsResp.Count == 0 {
		t.Fatalf("ports count = %d, want > 0", portsResp.Count)
	}
	if len(portsResp.Ports) == 0 {
		t.Fatalf("ports list is empty")
	}
	if portsResp.Ports[0].NodeID != nodeID {
		t.Fatalf("first port node_id = %d, want %d", portsResp.Ports[0].NodeID, nodeID)
	}
}

func TestSmokeInvalidIDsReturnBadRequest(t *testing.T) {
	baseURL := testBaseURL()
	client := testHTTPClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	waitHealthz(t, ctx, client, baseURL)

	tests := []struct {
		name string
		path string
	}{
		{name: "invalid log id", path: "/api/v1/log/0"},
		{name: "invalid topology log id", path: "/api/v1/topology/0"},
		{name: "invalid node id", path: "/api/v1/node/0"},
		{name: "invalid port node id", path: "/api/v1/port/0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := doRequest(t, ctx, client, http.MethodGet, baseURL+test.path, nil)
			defer closeBody(t, resp.Body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func TestSmokeMissingLogFileReturnsNotFound(t *testing.T) {
	baseURL := testBaseURL()
	client := testHTTPClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	waitHealthz(t, ctx, client, baseURL)

	resp := doRequest(t, ctx, client, http.MethodPost, baseURL+"/api/v1/parse", []byte(`{"path":"missing.zip"}`))
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestSmokeInvalidParseBodyReturnsBadRequest(t *testing.T) {
	baseURL := testBaseURL()
	client := testHTTPClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	waitHealthz(t, ctx, client, baseURL)

	tests := []struct {
		name string
		body string
	}{
		{name: "empty json", body: `{}`},
		{name: "blank path", body: `{"path":"   "}`},
		{name: "invalid json", body: `{`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := doRequest(t, ctx, client, http.MethodPost, baseURL+"/api/v1/parse", []byte(test.body))
			defer closeBody(t, resp.Body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func testBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return strings.TrimRight(baseURL, "/")
}

func testHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

func waitHealthz(t *testing.T, ctx context.Context, client *http.Client, baseURL string) {
	t.Helper()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.NewTimer(20 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context canceled while waiting healthz: %v", ctx.Err())
		case <-deadline.C:
			t.Fatalf("timeout waiting for %s/healthz", baseURL)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
			if err != nil {
				t.Fatalf("new healthz request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return
			}
		}
	}
}

func postParse(t *testing.T, ctx context.Context, client *http.Client, baseURL string, path string) parseResponse {
	t.Helper()

	body := []byte(fmt.Sprintf(`{"path":%q}`, path))

	resp := doRequest(t, ctx, client, http.MethodPost, baseURL+"/api/v1/parse", body)
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/v1/parse status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}

	var out parseResponse
	decodeJSON(t, resp.Body, &out)

	return out
}

func getLog(t *testing.T, ctx context.Context, client *http.Client, baseURL string, logID int64) logResponse {
	t.Helper()

	resp := doRequest(t, ctx, client, http.MethodGet, fmt.Sprintf("%s/api/v1/log/%d", baseURL, logID), nil)
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/log/%d status = %d, want %d; body: %s", logID, resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}

	var out logResponse
	decodeJSON(t, resp.Body, &out)

	return out
}

func getTopology(t *testing.T, ctx context.Context, client *http.Client, baseURL string, logID int64) topologyResponse {
	t.Helper()

	resp := doRequest(t, ctx, client, http.MethodGet, fmt.Sprintf("%s/api/v1/topology/%d", baseURL, logID), nil)
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/topology/%d status = %d, want %d; body: %s", logID, resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}

	var out topologyResponse
	decodeJSON(t, resp.Body, &out)

	return out
}

func getNode(t *testing.T, ctx context.Context, client *http.Client, baseURL string, nodeID int64) nodeResponse {
	t.Helper()

	resp := doRequest(t, ctx, client, http.MethodGet, fmt.Sprintf("%s/api/v1/node/%d", baseURL, nodeID), nil)
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/node/%d status = %d, want %d; body: %s", nodeID, resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}

	var out nodeResponse
	decodeJSON(t, resp.Body, &out)

	return out
}

func getPortsByNode(t *testing.T, ctx context.Context, client *http.Client, baseURL string, nodeID int64) portsResponse {
	t.Helper()

	resp := doRequest(t, ctx, client, http.MethodGet, fmt.Sprintf("%s/api/v1/port/%d", baseURL, nodeID), nil)
	defer closeBody(t, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/port/%d status = %d, want %d; body: %s", nodeID, resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}

	var out portsResponse
	decodeJSON(t, resp.Body, &out)

	return out
}

func doRequest(t *testing.T, ctx context.Context, client *http.Client, method string, url string, body []byte) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}

	return resp
}

func decodeJSON(t *testing.T, body io.Reader, out any) {
	t.Helper()

	if err := json.NewDecoder(body).Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return string(data)
}

func closeBody(t *testing.T, body io.Closer) {
	t.Helper()

	if err := body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
}

func assertRFC3339(t *testing.T, field string, value string) {
	t.Helper()

	if value == "" {
		t.Fatalf("%s is empty", field)
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		t.Fatalf("%s = %q is not RFC3339: %v", field, value, err)
	}
}

func assertGroupsContain(t *testing.T, groups []topologyGroup, kinds ...string) {
	t.Helper()

	seen := make(map[string]bool, len(groups))
	for _, group := range groups {
		seen[group.Kind] = true
	}

	for _, kind := range kinds {
		if !seen[kind] {
			t.Fatalf("topology groups do not contain kind %q: %+v", kind, groups)
		}
	}
}

func firstNodeWithPorts(t *testing.T, nodes []topologyNode) int64 {
	t.Helper()

	for _, node := range nodes {
		if node.ID > 0 && node.ParsedPortsCount > 0 {
			return node.ID
		}
	}

	t.Fatalf("no topology node with parsed ports: %+v", nodes)
	return 0
}

type parseResponse struct {
	LogID          int64  `json:"log_id"`
	Status         string `json:"status"`
	NodesCount     int32  `json:"nodes_count"`
	PortsCount     int32  `json:"ports_count"`
	NodesInfoCount int32  `json:"nodes_info_count"`
}

type logResponse struct {
	ID         int64  `json:"id"`
	FilePath   string `json:"file_path"`
	Status     string `json:"status"`
	NodesCount int32  `json:"nodes_count"`
	PortsCount int32  `json:"ports_count"`
	UploadedAt string `json:"uploaded_at"`
	ParsedAt   string `json:"parsed_at"`
}

type topologyResponse struct {
	LogID   int64           `json:"log_id"`
	Summary topologySummary `json:"summary"`
	Nodes   []topologyNode  `json:"nodes"`
	Groups  []topologyGroup `json:"groups"`
	Edges   []topologyEdge  `json:"edges"`
}

type topologySummary struct {
	NodesCount    int32 `json:"nodes_count"`
	PortsCount    int32 `json:"ports_count"`
	EdgesCount    int32 `json:"edges_count"`
	HostsCount    int32 `json:"hosts_count"`
	SwitchesCount int32 `json:"switches_count"`
}

type topologyNode struct {
	ID                 int64  `json:"id"`
	LogID              int64  `json:"log_id"`
	NodeGUID           string `json:"node_guid"`
	NodeDesc           string `json:"node_desc"`
	NodeType           int32  `json:"node_type"`
	NodeKind           string `json:"node_kind"`
	DeclaredPortsCount int32  `json:"declared_ports_count"`
	ParsedPortsCount   int32  `json:"parsed_ports_count"`
	SerialNumber       string `json:"serial_number"`
	ProductName        string `json:"product_name"`
}

type topologyGroup struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	NodeIDs   []int64  `json:"node_ids"`
	NodeGUIDs []string `json:"node_guids"`
}

type topologyEdge struct {
	SourceNodeID    int64  `json:"source_node_id"`
	SourceNodeGUID  string `json:"source_node_guid"`
	SourcePortNum   int32  `json:"source_port_num"`
	SourcePortGUID  string `json:"source_port_guid"`
	TargetNodeID    int64  `json:"target_node_id"`
	TargetNodeGUID  string `json:"target_node_guid"`
	TargetPortNum   int32  `json:"target_port_num"`
	TargetPortGUID  string `json:"target_port_guid"`
	Relation        string `json:"relation"`
	LinkWidthActive int32  `json:"link_width_active"`
	LinkSpeedActive int32  `json:"link_speed_active"`
	PortState       int32  `json:"port_state"`
}

type nodeResponse struct {
	ID       int64  `json:"id"`
	LogID    int64  `json:"log_id"`
	NodeGUID string `json:"node_guid"`
	NodeDesc string `json:"node_desc"`
	NodeKind string `json:"node_kind"`
}

type portsResponse struct {
	Count int            `json:"count"`
	Ports []portResponse `json:"ports"`
}

type portResponse struct {
	ID       int64  `json:"id"`
	LogID    int64  `json:"log_id"`
	NodeID   int64  `json:"node_id"`
	NodeGUID string `json:"node_guid"`
	PortGUID string `json:"port_guid"`
	PortNum  int32  `json:"port_num"`
}
