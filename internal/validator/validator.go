package validator

import (
	"context"
	"log/slog"
	"net"

	"github.com/grid-stream-org/api/pkg/firebase"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/handler"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	handler    *handler.Service
	log        *slog.Logger
}

func New(cfg *config.Config, fc firebase.FirebaseClient, log *slog.Logger) (*Server, error) {
	lis, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer()
	h := handler.New(cfg, fc, log)
	pb.RegisterValidatorServiceServer(grpcServer, h)
	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
		handler:    h,
		log:        log,
	}, nil
}

func (s *Server) Run(ctx context.Context, log *slog.Logger) error {
	log.Info("starting gRPC server", "address", s.listener.Addr().String())

	// Run server in a goroutine
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil && err != grpc.ErrServerStopped {
			log.Error("server error", "err", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Trigger shutdown process
	s.Stop(ctx)

	return ctx.Err()
}

func (s *Server) Stop(ctx context.Context) {
	s.log.Info("shutting down server...")

	// Stop the gRPC server
	s.grpcServer.GracefulStop()

	// Execute shutdown handler
	if err := s.handler.OnShutdown(ctx); err != nil {
		s.log.Error("shutdown tasks failed", "error", err)
	}
}
