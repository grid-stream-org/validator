package validator

import (
	"context"
	"net"

	"log/slog"

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

func New(cfg *config.Config, log *slog.Logger) (*Server, error) {
	lis, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	h := handler.New(cfg, log)
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

	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			log.Error("gRPC server error", "err", err)
		}
	}()

	<-ctx.Done()
	s.grpcServer.GracefulStop()
	log.Info("server stopped")
	return nil
}

func (s *Server) Stop(ctx context.Context) {
	s.log.Info("shutting down server...")

	// Gracefully stop gRPC
	s.grpcServer.GracefulStop()

	// Call shutdown logic on handler
	if err := s.handler.OnShutdown(ctx); err != nil {
		s.log.Error("failed to complete shutdown tasks", "error", err)
	}
}
