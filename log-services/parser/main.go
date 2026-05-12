package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	parsergrpc "github.com/voronkov44/microservice-log-parser/log-services/parser/adapters/grpc"
	"github.com/voronkov44/microservice-log-parser/log-services/parser/config"
	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
	parserpb "github.com/voronkov44/microservice-log-parser/log-services/proto/parser"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "parser server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("parser server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting parser server", "address", cfg.Address)
	log.Debug("debug messages are enabled")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// service
	parser := core.NewService(log)

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	parserpb.RegisterParserServer(s, parsergrpc.NewServer(log, parser))
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		log.Debug("shutting down parser server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func mustMakeLogger(levelStr string) *slog.Logger {
	var level slog.Level

	switch levelStr {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + levelStr)
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
