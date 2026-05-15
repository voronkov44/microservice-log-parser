package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
)

func TestServiceRejectsBadArguments(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), &fakeDB{})
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "CreateLog empty path", call: func() error {
			_, err := service.CreateLog(ctx, " ")
			return err
		}},
		{name: "SaveParsedLog bad id", call: func() error {
			_, err := service.SaveParsedLog(ctx, 0, ParsedLog{})
			return err
		}},
		{name: "FailLog bad id", call: func() error {
			return service.FailLog(ctx, 0, "failed")
		}},
		{name: "GetLog bad id", call: func() error {
			_, err := service.GetLog(ctx, -1)
			return err
		}},
		{name: "GetNode bad id", call: func() error {
			_, err := service.GetNode(ctx, -1)
			return err
		}},
		{name: "GetPortsByNode bad id", call: func() error {
			_, err := service.GetPortsByNode(ctx, -1)
			return err
		}},
		{name: "GetNodesByLog bad id", call: func() error {
			_, err := service.GetNodesByLog(ctx, -1)
			return err
		}},
		{name: "GetPortsByLog bad id", call: func() error {
			_, err := service.GetPortsByLog(ctx, -1)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrBadArguments) {
				t.Fatalf("error = %v, want ErrBadArguments", err)
			}
		})
	}
}

func TestServicePropagatesDBErrors(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("db failed")

	tests := []struct {
		name string
		db   *fakeDB
		call func(*Service) error
	}{
		{name: "Ping", db: &fakeDB{pingErr: wantErr}, call: func(s *Service) error {
			return s.Ping(ctx)
		}},
		{name: "CreateLog", db: &fakeDB{createLogErr: wantErr}, call: func(s *Service) error {
			_, err := s.CreateLog(ctx, "log.zip")
			return err
		}},
		{name: "SaveParsedLog", db: &fakeDB{saveParsedLogErr: wantErr}, call: func(s *Service) error {
			_, err := s.SaveParsedLog(ctx, 1, ParsedLog{})
			return err
		}},
		{name: "FailLog", db: &fakeDB{failLogErr: wantErr}, call: func(s *Service) error {
			return s.FailLog(ctx, 1, "failed")
		}},
		{name: "GetLog", db: &fakeDB{getLogErr: wantErr}, call: func(s *Service) error {
			_, err := s.GetLog(ctx, 1)
			return err
		}},
		{name: "GetNode", db: &fakeDB{getNodeErr: wantErr}, call: func(s *Service) error {
			_, err := s.GetNode(ctx, 1)
			return err
		}},
		{name: "GetPortsByNode", db: &fakeDB{getPortsByNodeErr: wantErr}, call: func(s *Service) error {
			_, err := s.GetPortsByNode(ctx, 1)
			return err
		}},
		{name: "GetNodesByLog", db: &fakeDB{getNodesByLogErr: wantErr}, call: func(s *Service) error {
			_, err := s.GetNodesByLog(ctx, 1)
			return err
		}},
		{name: "GetPortsByLog", db: &fakeDB{getPortsByLogErr: wantErr}, call: func(s *Service) error {
			_, err := s.GetPortsByLog(ctx, 1)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), tt.db)

			if err := tt.call(service); !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
		})
	}
}

type fakeDB struct {
	pingErr           error
	createLogErr      error
	saveParsedLogErr  error
	failLogErr        error
	getLogErr         error
	getNodeErr        error
	getPortsByNodeErr error
	getNodesByLogErr  error
	getPortsByLogErr  error
}

func (f *fakeDB) Ping(context.Context) error {
	return f.pingErr
}

func (f *fakeDB) CreateLog(context.Context, string) (int64, error) {
	return 11, f.createLogErr
}

func (f *fakeDB) SaveParsedLog(context.Context, int64, ParsedLog) (SaveParsedLogResult, error) {
	return SaveParsedLogResult{LogID: 11, NodesCount: 1, PortsCount: 2}, f.saveParsedLogErr
}

func (f *fakeDB) FailLog(context.Context, int64, string) error {
	return f.failLogErr
}

func (f *fakeDB) GetLog(context.Context, int64) (Log, error) {
	return Log{ID: 11}, f.getLogErr
}

func (f *fakeDB) GetNode(context.Context, int64) (Node, error) {
	return Node{ID: 21}, f.getNodeErr
}

func (f *fakeDB) GetPortsByNode(context.Context, int64) ([]Port, error) {
	return []Port{{ID: 31}}, f.getPortsByNodeErr
}

func (f *fakeDB) GetNodesByLog(context.Context, int64) ([]Node, error) {
	return []Node{{ID: 21}}, f.getNodesByLogErr
}

func (f *fakeDB) GetPortsByLog(context.Context, int64) ([]Port, error) {
	return []Port{{ID: 31}}, f.getPortsByLogErr
}
