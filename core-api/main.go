package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	defaultConfigPath  = "./config.yaml"
	configPathEnv      = "HOLONIC_ASSET_CONFIG"
	defaultHTTPAddress = ":8080"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, resolveConfigPath(), defaultHTTPAddress); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, configPath string, address string) error {
	app, err := InitServer(configPath)
	if err != nil {
		return err
	}

	startErr := app.Start(ctx, address)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	shutdownErr := app.Shutdown(shutdownCtx)
	return errors.Join(startErr, shutdownErr)
}

func resolveConfigPath() string {
	if path := strings.TrimSpace(os.Getenv(configPathEnv)); path != "" {
		return path
	}
	return defaultConfigPath
}
