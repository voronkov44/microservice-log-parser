package req

import (
	"errors"
	"net/http"

	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/res"
)

func HandleBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	return handleBody[T](w, r)
}

func HandleBodyLimit[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (*T, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return handleBody[T](w, r)
}

func handleBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	body, err := Decode[T](r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			res.Json(w, map[string]string{"error": "request body is too large"}, http.StatusRequestEntityTooLarge)
			return nil, err
		}

		res.Json(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return nil, err
	}

	return &body, nil
}
