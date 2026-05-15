package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	repositorypb "github.com/voronkov44/microservice-log-parser/log-services/proto/repository"
	repositorydb "github.com/voronkov44/microservice-log-parser/log-services/repository/adapters/db"
	repositorygrpc "github.com/voronkov44/microservice-log-parser/log-services/repository/adapters/grpc"
	"github.com/voronkov44/microservice-log-parser/log-services/repository/config"
	"github.com/voronkov44/microservice-log-parser/log-services/repository/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "repository server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("repository server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting repository server",
		"address", cfg.Address,
		"db_address", maskDSN(cfg.DBAddress),
	)
	log.Debug("debug messages are enabled")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// database adapter
	storage, err := repositorydb.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to create db adapter: %w", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Warn("failed to close db connection", "error", err)
		}
	}()

	// migrations
	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	// service
	repository := core.NewService(log, storage)

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	repositorypb.RegisterRepositoryServer(s, repositorygrpc.NewServer(log, repository))
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		log.Debug("shutting down repository server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func maskDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.User == nil {
		return dsn
	}

	username := parsed.User.Username()
	if _, ok := parsed.User.Password(); ok {
		parsed.User = url.UserPassword(username, "xxxxx")
	}

	return parsed.String()
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
