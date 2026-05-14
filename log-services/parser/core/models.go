package core

type ParsedLog struct {
	Nodes     []Node
	Ports     []Port
	NodesInfo []NodeInfo
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
