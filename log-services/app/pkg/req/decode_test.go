package req

import (
	"errors"
	"strings"
	"testing"
)

type decodePayload struct {
	Path string `json:"path"`
}

func TestDecodeValidJSON(t *testing.T) {
	got, err := Decode[decodePayload](strings.NewReader(`{"path":"log.zip"}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if got.Path != "log.zip" {
		t.Fatalf("Decode() path = %q, want log.zip", got.Path)
	}
}

func TestDecodeRejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantEmpty bool
	}{
		{name: "empty body", body: "", wantEmpty: true},
		{name: "unknown field", body: `{"path":"log.zip","extra":1}`},
		{name: "multiple json objects", body: `{"path":"log.zip"}{"path":"other.zip"}`},
		{name: "broken json", body: `{"path":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode[decodePayload](strings.NewReader(tt.body))
			if err == nil {
				t.Fatalf("Decode() error is nil, want error")
			}
			if tt.wantEmpty && !errors.Is(err, ErrEmptyBody) {
				t.Fatalf("Decode() error = %v, want ErrEmptyBody", err)
			}
		})
	}
}
