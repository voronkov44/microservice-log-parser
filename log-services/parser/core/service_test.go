package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
)

func TestServiceParseValidatesPath(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeEngine{})

	_, err := service.Parse(context.Background(), " ")
	if !errors.Is(err, ErrBadArguments) {
		t.Fatalf("Parse() error = %v, want ErrBadArguments", err)
	}
}

func TestServiceParseDelegatesToEngine(t *testing.T) {
	want := ParsedLog{
		Nodes: []Node{{NodeGUID: "node-1"}},
	}
	engine := &fakeEngine{parsed: want}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), engine)

	got, err := service.Parse(context.Background(), "log.zip")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if engine.path != "log.zip" {
		t.Fatalf("engine path = %q, want log.zip", engine.path)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].NodeGUID != "node-1" {
		t.Fatalf("Parse() = %+v, want %+v", got, want)
	}
}

func TestServiceParsePropagatesEngineError(t *testing.T) {
	wantErr := errors.New("engine failed")
	engine := &fakeEngine{err: wantErr}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), engine)

	_, err := service.Parse(context.Background(), "log.zip")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

type fakeEngine struct {
	path   string
	parsed ParsedLog
	err    error
}

func (f *fakeEngine) Parse(_ context.Context, path string) (ParsedLog, error) {
	f.path = path
	return f.parsed, f.err
}
