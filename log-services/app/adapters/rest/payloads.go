package rest

type healthResponse struct {
	Replies map[string]string `json:"replies"`
}

type errorResponse struct {
	Error string `json:"error"`
}
