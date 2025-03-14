package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/grid-stream-org/batcher/pkg/sigctx"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/server"
	"github.com/grid-stream-org/validator/internal/validation"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
)

func main() {
	log := logger.Default()
	exitCode := 0
	if err := run(); err != nil {
		exitCode = handleErrors(err, log)
	}
	log.Info("Done", "exitCode", exitCode)
	os.Exit(exitCode)

}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Initialize logger
	log, err := logger.New(cfg.Log, nil)
	if err != nil {
		return err
	}

	log.Info("Starting Validator Server...")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for system interrupt signals
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		log.Info("Received shutdown signal, shutting down...")
		cancel() // Cancel context when signal is received
	}()

	// Initialize gRPC server
	server, err := server.NewGRPCServer(ctx, cfg)
	if err != nil {
		return err
	}

	// Start monitoring routine
	go validation.StartMonitor(server.ValidatorServer)

	// Start the gRPC server in a goroutine
	grpcErrChan := make(chan error, 1)
	go func() {
		log.Info("Validator server is running", "address", cfg.Server.Address)
		if err := server.GRPCServer.Serve(server.Listener); err != nil && err != grpc.ErrServerStopped {
			grpcErrChan <- errors.Wrap(err, "gRPC server failed")
		}
		close(grpcErrChan)
	}()

	// Wait for shutdown signal or server failure
	select {
	case <-ctx.Done():
		log.Info("Shutting down gRPC server...")
		server.GRPCServer.GracefulStop()
	case err := <-grpcErrChan:
		log.Error("gRPC server encountered an error shutting down", "error", err)
		return err
	}

	return nil
}

func handleErrors(err error, log *slog.Logger) int {
	if err == nil {
		return 0
	}
	var exitCode int
	errs := []error{}
	// Filter and process errors
	for _, mErr := range multierr.Errors(err) {
		var sigErr *sigctx.SignalError
		if errors.As(mErr, &sigErr) {
			exitCode = sigErr.SigNum()
		} else if !errors.Is(mErr, context.Canceled) {
			errs = append(errs, mErr)
		}
	}
	// Log non-signal errors
	if len(errs) > 0 {
		for _, err := range errs {
			log.Error("error occurred", "error", err, "stack", fmt.Sprintf("%+v", err))
		}
		if exitCode == 0 {
			exitCode = 255
		}
	}
	return exitCode
}
