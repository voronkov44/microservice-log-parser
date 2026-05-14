package rest

type healthResponse struct {
	Replies map[string]string `json:"replies"`
}

type parseLogRequest struct {
	Path string `json:"path"`
}

type parseLogResponse struct {
	LogID          int64  `json:"log_id"`
	Status         string `json:"status"`
	NodesCount     int32  `json:"nodes_count"`
	PortsCount     int32  `json:"ports_count"`
	NodesInfoCount int32  `json:"nodes_info_count"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type topologyResponse struct {
	LogID   int64                   `json:"log_id"`
	Summary topologySummaryResponse `json:"summary"`
	Nodes   []topologyNodeResponse  `json:"nodes"`
	Groups  []topologyGroupResponse `json:"groups"`
	Edges   []topologyEdgeResponse  `json:"edges"`
}

type topologySummaryResponse struct {
	NodesCount    int32 `json:"nodes_count"`
	PortsCount    int32 `json:"ports_count"`
	EdgesCount    int32 `json:"edges_count"`
	HostsCount    int32 `json:"hosts_count"`
	SwitchesCount int32 `json:"switches_count"`
}

type topologyNodeResponse struct {
	ID                 int64  `json:"id"`
	LogID              int64  `json:"log_id"`
	NodeGUID           string `json:"node_guid"`
	NodeDesc           string `json:"node_desc"`
	NodeType           int32  `json:"node_type"`
	NodeKind           string `json:"node_kind"`
	DeclaredPortsCount int32  `json:"declared_ports_count"`
	ParsedPortsCount   int32  `json:"parsed_ports_count"`
	SerialNumber       string `json:"serial_number,omitempty"`
	ProductName        string `json:"product_name,omitempty"`
}

type topologyGroupResponse struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	NodeIDs   []int64  `json:"node_ids"`
	NodeGUIDs []string `json:"node_guids"`
}

type topologyEdgeResponse struct {
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
