package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	parserclient "github.com/voronkov44/microservice-log-parser/log-services/app/adapters/parser"
	repositoryclient "github.com/voronkov44/microservice-log-parser/log-services/app/adapters/repository"
	"github.com/voronkov44/microservice-log-parser/log-services/app/adapters/rest"
	"github.com/voronkov44/microservice-log-parser/log-services/app/adapters/rest/middleware"
	topologyclient "github.com/voronkov44/microservice-log-parser/log-services/app/adapters/topology"
	"github.com/voronkov44/microservice-log-parser/log-services/app/config"
	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	"github.com/voronkov44/microservice-log-parser/log-services/app/pkg/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "app server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log, cleanup, err := logger.New(cfg.LogLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := run(cfg, log); err != nil {
		log.Error("app server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting app server")
	log.Debug("debug messages are enabled")

	// Clients
	parserClient, err := parserclient.NewClient(cfg.ParserAddress, log)
	if err != nil {
		return fmt.Errorf("failed to create parser client: %w", err)
	}
	defer closeClient(log, "parser", parserClient.Close)

	repositoryClient, err := repositoryclient.NewClient(cfg.RepositoryAddress, log)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}
	defer closeClient(log, "repository", repositoryClient.Close)

	topologyClient, err := topologyclient.NewClient(cfg.TopologyAddress, log)
	if err != nil {
		return fmt.Errorf("failed to create topology client: %w", err)
	}
	defer closeClient(log, "topology", topologyClient.Close)

	// приведение типов для компилятора
	pingers := map[string]core.Pinger{
		"parser":     parserClient,
		"repository": repositoryClient,
		"topology":   topologyClient,
	}

	// Service
	appService := core.NewService(log, repositoryClient, parserClient, topologyClient)

	// Route
	router := http.NewServeMux()
	router.Handle("GET /healthz", rest.NewHealthHandler(log, pingers, cfg.HTTPConfig.Timeout))

	router.Handle("POST /logs/parse", rest.NewParseLogHandler(log, appService, cfg.HTTPConfig.Timeout))

	router.Handle("GET /logs/{log_id}", rest.NewGetLogHandler(log, appService, cfg.HTTPConfig.Timeout))
	router.Handle("GET /logs/{log_id}/nodes", rest.NewGetNodesByLogHandler(log, appService, cfg.HTTPConfig.Timeout))
	router.Handle("GET /logs/{log_id}/ports", rest.NewGetPortsByLogHandler(log, appService, cfg.HTTPConfig.Timeout))

	router.Handle("GET /nodes/{node_id}", rest.NewGetNodeHandler(log, appService, cfg.HTTPConfig.Timeout))
	router.Handle("GET /nodes/{node_id}/ports", rest.NewGetPortsByNodeHandler(log, appService, cfg.HTTPConfig.Timeout))

	router.Handle("GET /logs/{log_id}/topology", rest.NewGetTopologyHandler(log, appService, cfg.HTTPConfig.Timeout))

	stack := middleware.Chain(
		middleware.Recover(log),
		middleware.Logging(log),
		middleware.CORS,
	)

	server := &http.Server{
		Addr:              cfg.HTTPConfig.Address,
		Handler:           stack(router),
		ReadHeaderTimeout: cfg.HTTPConfig.ReadHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		log.Info("app server started",
			"address", cfg.HTTPConfig.Address,
			"parser_address", cfg.ParserAddress,
			"repository_address", cfg.RepositoryAddress,
			"topology_address", cfg.TopologyAddress,
			"log_level", cfg.LogLevel,
			"log_file_path", cfg.LogFilePath,
		)

		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown requested")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server stopped unexpectedly: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPConfig.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("app server stopped")

	return nil
}

func closeClient(log *slog.Logger, name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		log.Warn("failed to close grpc client",
			"service", name,
			"error", err,
		)
	}
}
