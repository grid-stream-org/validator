package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/grid-stream-org/api/pkg/firebase"
	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/grid-stream-org/batcher/pkg/sigctx"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/validator"
	"go.uber.org/multierr"
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

func run() (err error) {
	ctx, cancel := sigctx.New(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Log, nil)
	if err != nil {
		return err
	}

	fc, err := firebase.NewFirebaseClient(ctx, cfg.Firebase, log)
	if err != nil {
		return err
	}
	defer fc.Close()

	srv, err := validator.New(cfg, fc, log)
	if err != nil {
		return err
	}

	log.Info("Starting Validator Server...")
	if runErr := srv.Run(ctx, log); runErr != nil {
		err = multierr.Combine(err, runErr)
	}

	return err
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
