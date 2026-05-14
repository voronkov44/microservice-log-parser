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
