package req

import (
	"encoding/json"
	"errors"
	"io"
)

var ErrEmptyBody = errors.New("empty request body")

func Decode[T any](body io.Reader) (T, error) {
	var payload T

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return payload, ErrEmptyBody
		}

		return payload, err
	}

	var extra struct{}
	if err := decoder.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return payload, nil
		}

		return payload, err
	}

	return payload, errors.New("request body must contain only one JSON value")
}
