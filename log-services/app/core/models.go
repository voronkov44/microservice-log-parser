package core

type LogStatus string

const (
	LogStatusProcessing LogStatus = "processing"
	LogStatusParsed     LogStatus = "parsed"
	LogStatusFailed     LogStatus = "failed"
)

type ParsedLog struct {
	Nodes     []Node
	Ports     []Port
	NodesInfo []NodeInfo
}

type ParseLogResult struct {
	LogID          int64
	Status         LogStatus
	NodesCount     int32
	PortsCount     int32
	NodesInfoCount int32
}

type SaveParsedLogResult struct {
	LogID      int64
	NodesCount int32
	PortsCount int32
}

type Node struct {
	NodeGUID string
	NodeDesc string
	NodeType int32
	NodeKind string
	NumPorts int32

	ClassVersion    int32
	BaseVersion     int32
	SystemImageGUID string
	PortGUID        string

	RawJSON string
}

type Port struct {
	NodeGUID string
	PortGUID string
	PortNum  int32

	LID             int32
	LocalPortNum    int32
	PortState       int32
	PortPhyState    int32
	LinkWidthActive int32
	LinkSpeedActive int32

	RawJSON string
}

type NodeInfo struct {
	NodeGUID string

	SerialNumber string
	PartNumber   string
	Revision     string
	ProductName  string

	RawJSON string
}

type Topology struct {
	LogID   int64
	Summary TopologySummary
	Nodes   []TopologyNode
	Groups  []TopologyGroup
	Edges   []TopologyEdge
}

type TopologySummary struct {
	NodesCount    int32
	PortsCount    int32
	EdgesCount    int32
	HostsCount    int32
	SwitchesCount int32
}

type TopologyNode struct {
	ID                 int64
	LogID              int64
	NodeGUID           string
	NodeDesc           string
	NodeType           int32
	NodeKind           string
	DeclaredPortsCount int32
	ParsedPortsCount   int32
	SerialNumber       string
	ProductName        string
}

type TopologyGroup struct {
	Name      string
	Kind      string
	NodeIDs   []int64
	NodeGUIDs []string
}

type TopologyEdge struct {
	SourceNodeID    int64
	SourceNodeGUID  string
	SourcePortNum   int32
	SourcePortGUID  string
	TargetNodeID    int64
	TargetNodeGUID  string
	TargetPortNum   int32
	TargetPortGUID  string
	Relation        string
	LinkWidthActive int32
	LinkSpeedActive int32
	PortState       int32
}
