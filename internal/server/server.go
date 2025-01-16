package server

import (
	"context"
	"log/slog"
	"net"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1" // Import the generated protobuf package
	"github.com/grid-stream-org/validator/internal/config"
	"google.golang.org/grpc"
)

// ValidatorServer implements the ValidatorService gRPC interface
type ValidatorServer struct {
	pb.UnimplementedValidatorServiceServer
}

// ValidateAverageOutputs handles validation requests
func (s *ValidatorServer) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	logger.Default().Info("Recieved validation request")


	return nil, nil
}

func (s *ValidatorServer) NotifyProject(ctx context.Context, req *pb.NotifyProjectRequest) (*pb.NotifyProjectResponse, error) {
		logger.Default().Info("Recieved new project", "projectId", req.ProjectId)
		return nil, nil
}

func Start(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	address := cfg.Server.Address // Dynamically read server address from config

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterValidatorServiceServer(grpcServer, &ValidatorServer{})

	// Run the server in a goroutine to allow for graceful shutdown
	go func() {
		log.Info("Validator server is running on ", "address", address)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC server", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	return nil
}
