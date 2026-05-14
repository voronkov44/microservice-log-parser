package core

type LogStatus string

const (
	LogStatusProcessing LogStatus = "processing"
	LogStatusParsed     LogStatus = "parsed"
	LogStatusFailed     LogStatus = "failed"
)

type Log struct {
	ID         int64
	FilePath   string
	Status     LogStatus
	NodesCount int32
	PortsCount int32
	Error      string
	UploadedAt string
	ParsedAt   string
}

type ParsedLog struct {
	Nodes     []Node
	Ports     []Port
	NodesInfo []NodeInfo
}

type SaveParsedLogResult struct {
	LogID      int64
	NodesCount int32
	PortsCount int32
}

type Node struct {
	ID    int64
	LogID int64

	NodeGUID string
	NodeDesc string
	NodeType int32
	NodeKind string
	NumPorts int32

	ClassVersion    int32
	BaseVersion     int32
	SystemImageGUID string
	PortGUID        string

	Info    *NodeInfo
	RawJSON string
}

type NodeInfo struct {
	ID     int64
	NodeID int64

	NodeGUID string

	SerialNumber string
	PartNumber   string
	Revision     string
	ProductName  string

	RawJSON string
}

type Port struct {
	ID     int64
	LogID  int64
	NodeID int64

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
