package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func TestServiceParseLogRejectsEmptyPath(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeRepository{}, &fakeParser{}, &fakeTopology{})

	_, err := service.ParseLog(context.Background(), " ")
	if !errors.Is(err, ErrBadArguments) {
		t.Fatalf("ParseLog() error = %v, want ErrBadArguments", err)
	}
}

func TestServiceParseLogSuccessFlow(t *testing.T) {
	parsed := ParsedLog{
		Nodes:     []Node{{NodeGUID: "node-1"}},
		Ports:     []Port{{NodeGUID: "node-1", PortGUID: "port-1"}},
		NodesInfo: []NodeInfo{{NodeGUID: "node-1"}, {NodeGUID: "node-2"}},
	}
	repo := &fakeRepository{
		createLogID: 101,
		saveResult:  SaveParsedLogResult{LogID: 101, NodesCount: 1, PortsCount: 1},
	}
	parser := &fakeParser{parsed: parsed}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), repo, parser, &fakeTopology{})

	got, err := service.ParseLog(context.Background(), " log.zip ")
	if err != nil {
		t.Fatalf("ParseLog() error = %v", err)
	}

	want := ParseLogResult{
		LogID:          101,
		Status:         LogStatusParsed,
		NodesCount:     1,
		PortsCount:     1,
		NodesInfoCount: 2,
	}
	if got != want {
		t.Fatalf("ParseLog() = %+v, want %+v", got, want)
	}

	wantCalls := []string{"CreateLog:log.zip", "Parse:log.zip", "SaveParsedLog:101"}
	if !reflect.DeepEqual(repo.calls, []string{"CreateLog:log.zip", "SaveParsedLog:101"}) {
		t.Fatalf("repository calls = %v", repo.calls)
	}
	if !reflect.DeepEqual(append([]string{repo.calls[0]}, append(parser.calls, repo.calls[1:]...)...), wantCalls) {
		t.Fatalf("flow calls do not match expected order")
	}
}

func TestServiceParseLogFailsLogWhenParserFails(t *testing.T) {
	wantErr := errors.New("parse failed")
	repo := &fakeRepository{createLogID: 101}
	parser := &fakeParser{err: wantErr}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), repo, parser, &fakeTopology{})

	_, err := service.ParseLog(context.Background(), "log.zip")
	if !errors.Is(err, wantErr) {
		t.Fatalf("ParseLog() error = %v, want %v", err, wantErr)
	}

	if len(repo.failLogCalls) != 1 {
		t.Fatalf("FailLog calls = %d, want 1", len(repo.failLogCalls))
	}
	if repo.failLogCalls[0].logID != 101 || repo.failLogCalls[0].errorText != wantErr.Error() {
		t.Fatalf("FailLog call = %+v", repo.failLogCalls[0])
	}
}

func TestServiceParseLogFailsLogWhenSaveFails(t *testing.T) {
	wantErr := errors.New("save failed")
	repo := &fakeRepository{
		createLogID:   101,
		saveParsedErr: wantErr,
	}
	parser := &fakeParser{parsed: ParsedLog{Nodes: []Node{{NodeGUID: "node-1"}}}}
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), repo, parser, &fakeTopology{})

	_, err := service.ParseLog(context.Background(), "log.zip")
	if !errors.Is(err, wantErr) {
		t.Fatalf("ParseLog() error = %v, want %v", err, wantErr)
	}

	if len(repo.failLogCalls) != 1 {
		t.Fatalf("FailLog calls = %d, want 1", len(repo.failLogCalls))
	}
	if repo.failLogCalls[0].logID != 101 || repo.failLogCalls[0].errorText != wantErr.Error() {
		t.Fatalf("FailLog call = %+v", repo.failLogCalls[0])
	}
}

func TestServiceGetTopology(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeRepository{}, &fakeParser{}, &fakeTopology{
		topology: Topology{
			LogID:   101,
			Summary: TopologySummary{NodesCount: 1},
		},
	})

	_, err := service.GetTopology(context.Background(), 0)
	if !errors.Is(err, ErrBadArguments) {
		t.Fatalf("GetTopology() error = %v, want ErrBadArguments", err)
	}

	got, err := service.GetTopology(context.Background(), 101)
	if err != nil {
		t.Fatalf("GetTopology() error = %v", err)
	}
	if got.LogID != 101 || got.Summary.NodesCount != 1 {
		t.Fatalf("GetTopology() = %+v", got)
	}
}

type failLogCall struct {
	logID     int64
	errorText string
}

type fakeRepository struct {
	calls        []string
	createLogID  int64
	createLogErr error

	saveResult    SaveParsedLogResult
	saveParsedErr error
	failLogErr    error
	failLogCalls  []failLogCall
}

func (f *fakeRepository) Ping(context.Context) error {
	return nil
}

func (f *fakeRepository) CreateLog(_ context.Context, filePath string) (int64, error) {
	f.calls = append(f.calls, "CreateLog:"+filePath)
	return f.createLogID, f.createLogErr
}

func (f *fakeRepository) SaveParsedLog(_ context.Context, logID int64, _ ParsedLog) (SaveParsedLogResult, error) {
	f.calls = append(f.calls, "SaveParsedLog:101")
	if logID != 101 {
		f.calls[len(f.calls)-1] = "SaveParsedLog:unexpected"
	}
	return f.saveResult, f.saveParsedErr
}

func (f *fakeRepository) FailLog(_ context.Context, logID int64, errorText string) error {
	f.failLogCalls = append(f.failLogCalls, failLogCall{logID: logID, errorText: errorText})
	return f.failLogErr
}

func (f *fakeRepository) GetLog(context.Context, int64) (Log, error) {
	return Log{}, nil
}

func (f *fakeRepository) GetNode(context.Context, int64) (Node, error) {
	return Node{}, nil
}

func (f *fakeRepository) GetPortsByNode(context.Context, int64) ([]Port, error) {
	return nil, nil
}

func (f *fakeRepository) GetNodesByLog(context.Context, int64) ([]Node, error) {
	return nil, nil
}

func (f *fakeRepository) GetPortsByLog(context.Context, int64) ([]Port, error) {
	return nil, nil
}

type fakeParser struct {
	calls  []string
	parsed ParsedLog
	err    error
}

func (f *fakeParser) Ping(context.Context) error {
	return nil
}

func (f *fakeParser) Parse(_ context.Context, path string) (ParsedLog, error) {
	f.calls = append(f.calls, "Parse:"+path)
	return f.parsed, f.err
}

type fakeTopology struct {
	topology Topology
	err      error
}

func (f *fakeTopology) Ping(context.Context) error {
	return nil
}

func (f *fakeTopology) GetTopology(context.Context, int64) (Topology, error) {
	return f.topology, f.err
}
