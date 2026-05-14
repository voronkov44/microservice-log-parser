package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
)

func parseKeyValueSections(file sourceFile) (core.ParsedLog, error) {
	scanner := bufio.NewScanner(bytes.NewReader(file.data))
	scanner.Buffer(make([]byte, 1024), maxFileSize)

	var records []map[string]string
	current := make(map[string]string)

	flush := func() {
		if len(current) == 0 {
			return
		}

		records = append(records, current)
		current = make(map[string]string)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "---") {
			flush()
			continue
		}

		key, value, ok := splitKeyValue(line)
		if !ok {
			continue
		}

		current[key] = value
	}

	if err := scanner.Err(); err != nil {
		return core.ParsedLog{}, fmt.Errorf("%w: scan failed: %v", core.ErrParse, err)
	}

	flush()

	return recordsToParsedLog(file.name, records), nil
}

func splitKeyValue(line string) (string, string, bool) {
	if strings.HasPrefix(line, "#") {
		return "", "", false
	}

	for _, sep := range []string{":", "="} {
		left, right, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}

		key := strings.TrimSpace(left)
		value := strings.Trim(strings.TrimSpace(right), `"'`)

		if key == "" {
			return "", "", false
		}

		return key, value, true
	}

	return "", "", false
}

func recordsToParsedLog(fileName string, records []map[string]string) core.ParsedLog {
	var parsed core.ParsedLog

	for _, rec := range records {
		normalized := normalizeRecord(rec)
		raw := rawJSON(rec)

		switch classifyRecord(fileName, normalized) {
		case "port":
			parsed.Ports = append(parsed.Ports, core.Port{
				NodeGUID:        get(normalized, "nodeguid", "nodeid", "node"),
				PortGUID:        get(normalized, "portguid", "portid"),
				PortNum:         parseInt32(get(normalized, "portnum", "portnumber", "port")),
				LID:             parseInt32(get(normalized, "lid")),
				LocalPortNum:    parseInt32(get(normalized, "localportnum", "localportnumber")),
				PortState:       parseInt32(get(normalized, "portstate", "state")),
				PortPhyState:    parseInt32(get(normalized, "portphystate", "phystate", "physicalstate")),
				LinkWidthActive: parseInt32(get(normalized, "linkwidthactive", "linkwidthactv", "linkwidth")),
				LinkSpeedActive: parseInt32(get(normalized, "linkspeedactive", "linkspeedactv", "linkspeed")),
				RawJSON:         raw,
			})

		case "info":
			parsed.NodesInfo = append(parsed.NodesInfo, core.NodeInfo{
				NodeGUID:     get(normalized, "nodeguid", "nodeid", "node"),
				SerialNumber: get(normalized, "serialnumber", "serial", "sn"),
				PartNumber:   get(normalized, "partnumber", "part", "pn"),
				Revision:     get(normalized, "revision", "rev"),
				ProductName:  get(normalized, "productname", "product", "description"),
				RawJSON:      raw,
			})

		case "node":
			nodeDesc := get(normalized, "nodedesc", "description", "desc", "name")
			nodeType := parseInt32(get(normalized, "nodetype", "type"))

			parsed.Nodes = append(parsed.Nodes, core.Node{
				NodeGUID:        get(normalized, "nodeguid", "nodeid", "node", "guid"),
				NodeDesc:        nodeDesc,
				NodeType:        nodeType,
				NodeKind:        deriveNodeKind(get(normalized, "nodekind", "kind"), nodeDesc, nodeType),
				NumPorts:        parseInt32(get(normalized, "numports", "ports", "portcount")),
				ClassVersion:    parseInt32(get(normalized, "classversion")),
				BaseVersion:     parseInt32(get(normalized, "baseversion")),
				SystemImageGUID: get(normalized, "systemimageguid", "systemimage"),
				PortGUID:        get(normalized, "portguid"),
				RawJSON:         raw,
			})
		}
	}

	return parsed
}

func classifyRecord(fileName string, rec map[string]string) string {
	name := normalizeKey(fileName)

	if hasAny(rec,
		"portnum",
		"portnumber",
		"localportnum",
		"localportnumber",
		"lid",
		"portstate",
		"portphystate",
		"phystate",
		"linkwidthactive",
		"linkspeedactive",
	) || strings.Contains(name, "port") {
		return "port"
	}

	if hasAny(rec,
		"serialnumber",
		"partnumber",
		"revision",
		"productname",
	) ||
		strings.Contains(name, "nodeinfo") ||
		strings.Contains(name, "nodesinfo") ||
		strings.Contains(name, "sharpaninfo") {
		return "info"
	}

	if hasAny(rec, "nodeguid", "nodeid", "guid") ||
		strings.Contains(name, "node") {
		return "node"
	}

	return ""
}

func finalizeParsedLog(parsed core.ParsedLog) core.ParsedLog {
	nodeIndex := make(map[string]int, len(parsed.Nodes))
	nodes := make([]core.Node, 0, len(parsed.Nodes))

	ensureNode := func(nodeGUID string) {
		if nodeGUID == "" {
			return
		}

		if _, ok := nodeIndex[nodeGUID]; ok {
			return
		}

		nodeIndex[nodeGUID] = len(nodes)
		nodes = append(nodes, core.Node{
			NodeGUID: nodeGUID,
			NodeKind: "unknown",
		})
	}

	for _, node := range parsed.Nodes {
		node.NodeGUID = strings.TrimSpace(node.NodeGUID)
		if node.NodeGUID == "" {
			continue
		}

		if idx, ok := nodeIndex[node.NodeGUID]; ok {
			nodes[idx] = mergeNode(nodes[idx], node)
			continue
		}

		nodeIndex[node.NodeGUID] = len(nodes)
		nodes = append(nodes, node)
	}

	infoIndex := make(map[string]int, len(parsed.NodesInfo))
	infos := make([]core.NodeInfo, 0, len(parsed.NodesInfo))

	for _, info := range parsed.NodesInfo {
		info.NodeGUID = strings.TrimSpace(info.NodeGUID)
		if info.NodeGUID == "" {
			continue
		}

		ensureNode(info.NodeGUID)

		if idx, ok := infoIndex[info.NodeGUID]; ok {
			infos[idx] = mergeNodeInfo(infos[idx], info)
			continue
		}

		infoIndex[info.NodeGUID] = len(infos)
		infos = append(infos, info)
	}

	portIndex := make(map[string]int, len(parsed.Ports))
	ports := make([]core.Port, 0, len(parsed.Ports))

	for _, port := range parsed.Ports {
		port.NodeGUID = strings.TrimSpace(port.NodeGUID)
		if port.NodeGUID == "" {
			continue
		}

		ensureNode(port.NodeGUID)

		key := port.NodeGUID + "|" + port.PortGUID + "|" + strconv.Itoa(int(port.PortNum))

		if idx, ok := portIndex[key]; ok {
			ports[idx] = mergePort(ports[idx], port)
			continue
		}

		portIndex[key] = len(ports)
		ports = append(ports, port)
	}

	return core.ParsedLog{
		Nodes:     nodes,
		Ports:     ports,
		NodesInfo: infos,
	}
}

func mergeNode(base, patch core.Node) core.Node {
	if patch.NodeDesc != "" {
		base.NodeDesc = patch.NodeDesc
	}
	if patch.NodeType != 0 {
		base.NodeType = patch.NodeType
	}
	if patch.NodeKind != "" && patch.NodeKind != "unknown" {
		base.NodeKind = patch.NodeKind
	}
	if patch.NumPorts != 0 {
		base.NumPorts = patch.NumPorts
	}
	if patch.ClassVersion != 0 {
		base.ClassVersion = patch.ClassVersion
	}
	if patch.BaseVersion != 0 {
		base.BaseVersion = patch.BaseVersion
	}
	if patch.SystemImageGUID != "" {
		base.SystemImageGUID = patch.SystemImageGUID
	}
	if patch.PortGUID != "" {
		base.PortGUID = patch.PortGUID
	}
	if patch.RawJSON != "" {
		base.RawJSON = patch.RawJSON
	}

	return base
}

func mergeNodeInfo(base, patch core.NodeInfo) core.NodeInfo {
	if patch.SerialNumber != "" {
		base.SerialNumber = patch.SerialNumber
	}
	if patch.PartNumber != "" {
		base.PartNumber = patch.PartNumber
	}
	if patch.Revision != "" {
		base.Revision = patch.Revision
	}
	if patch.ProductName != "" {
		base.ProductName = patch.ProductName
	}
	if patch.RawJSON != "" {
		base.RawJSON = patch.RawJSON
	}

	return base
}

func mergePort(base, patch core.Port) core.Port {
	if patch.PortGUID != "" {
		base.PortGUID = patch.PortGUID
	}
	if patch.PortNum != 0 {
		base.PortNum = patch.PortNum
	}
	if patch.LID != 0 {
		base.LID = patch.LID
	}
	if patch.LocalPortNum != 0 {
		base.LocalPortNum = patch.LocalPortNum
	}
	if patch.PortState != 0 {
		base.PortState = patch.PortState
	}
	if patch.PortPhyState != 0 {
		base.PortPhyState = patch.PortPhyState
	}
	if patch.LinkWidthActive != 0 {
		base.LinkWidthActive = patch.LinkWidthActive
	}
	if patch.LinkSpeedActive != 0 {
		base.LinkSpeedActive = patch.LinkSpeedActive
	}
	if patch.RawJSON != "" {
		base.RawJSON = patch.RawJSON
	}

	return base
}
