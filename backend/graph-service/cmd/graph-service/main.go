// Package main is the Graph Service entry point. It loads configuration,
// initialises shared infrastructure (logger, idgen, tracing) and runs the
// Hertz HTTP server via wire-injected dependencies.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"go.uber.org/zap"

	"tcm-history-ai/backend/graph-service/internal/conf"
	"tcm-history-ai/backend/graph-service/internal/controller"
	"tcm-history-ai/backend/pkg/config"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/tracing"
)

func main() {
	configPath := flag.String("config", "internal/conf/config.dev.yaml", "path to the YAML config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(logger.Config{
		Level:    cfg.Log.Level,
		Encoding: cfg.Log.Encoding,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	idgen.Init(cfg.App.NodeID)

	shutdownTracing, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName:  cfg.Tracing.ServiceName,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
		Enabled:      cfg.Tracing.Enabled,
	})
	if err != nil {
		logger.Default().Warn("tracing init failed; continuing without traces", zap.Error(err))
	}
	defer func() {
		_ = shutdownTracing(context.Background())
	}()

	app, cleanup, err := InitializeApp(cfg)
	if err != nil {
		logger.Default().Error("initialize app", zap.Error(err))
		os.Exit(1)
	}
	defer cleanup()

	h := server.Default(
		server.WithHostPorts(fmt.Sprintf(":%d", cfg.HTTP.Port)),
		server.WithReadTimeout(time.Duration(cfg.HTTP.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(cfg.HTTP.WriteTimeout)*time.Second),
	)

	controller.RegisterRoutes(h, app.ControllerDeps())

	logger.Default().Info("graph-service starting",
		zap.Int("port", cfg.HTTP.Port),
		zap.String("env", cfg.App.Env),
		zap.Int64("node_id", cfg.App.NodeID),
		zap.Bool("neo4j_enabled", cfg.Neo4j.Enabled))

	// Start the RabbitMQ subscriber in the background. The handler dispatches
	// each consumed event into SyncUseCase (doc/05 §5.6). Subscribe blocks
	// until ctx is cancelled or the broker delivery channel closes; best-effort
	// on dial failure (returns nil after logging).
	subscriberCtx, cancelSubscriber := context.WithCancel(context.Background())
	defer cancelSubscriber()
	go func() {
		if err := app.StartSubscriber(subscriberCtx); err != nil {
			logger.Default().Warn("graph subscriber exited", zap.Error(err))
		}
	}()

	go func() {
		h.Spin()
	}()

	waitForShutdown()
	logger.Default().Info("graph-service shutting down")
	_ = h.Close()
}

// loadConfig reads the YAML file and validates it.
func loadConfig(path string) (*conf.Config, error) {
	var cfg conf.Config
	if err := config.Load(path, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// waitForShutdown blocks until the process receives SIGINT or SIGTERM.
func waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
