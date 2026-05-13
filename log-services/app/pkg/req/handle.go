package req

import (
	"net/http"

	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

func HandleBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	body, err := Decode[T](r.Body)
	if err != nil {
		res.Json(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return nil, err
	}

	return &body, nil
}
