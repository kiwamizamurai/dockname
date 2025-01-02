package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiwamizamurai/dockname/internal/container"
	"github.com/kiwamizamurai/dockname/internal/proxy"
	"github.com/rs/zerolog"
)

func main() {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(consoleWriter).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerManager, err := container.NewDockerManager(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Docker manager")
	}

	proxyManager := proxy.NewManager(dockerManager, nil, logger)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		logger.Info().Msg("Starting shutdown")
		cancel()
	}()

	if err := proxyManager.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start proxy manager")
	}
}
