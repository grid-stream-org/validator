package server

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1" // Import the generated protobuf package
	"github.com/grid-stream-org/validator/internal/config"
	"google.golang.org/grpc"
)

// ValidatorServer implements the ValidatorService gRPC interface
type ValidatorServer struct {
	pb.UnimplementedValidatorServiceServer
	Summaries map[string]*Summary 
	Mu                 sync.Mutex
}

// Summary struct to store project-wide validation info
type Summary struct {
	ProjectID         string
	TimeStarted       string
	TimeEnded         string
	ContractThreshold float64
	ViolationRecords  []ViolationRecord
}

type ViolationRecord struct {
	StartTime string
	EndTime   string
	Average   float64
}


// ValidateAverageOutputs handles validation requests and updates project summaries
func (s *ValidatorServer) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	logger.Default().Info("Received validation request")

	// Check if there are any average outputs in the request
	if len(req.AverageOutputs) == 0 {
		logger.Default().Info("No averages found")
		return &pb.ValidateAverageOutputsResponse{
			Success: false,
			Errors:  []*pb.ValidationError{},
		}, nil
	}

	s.Mu.Lock() // Lock to prevent concurrent map modification
	defer s.Mu.Unlock()

	// Prepare a list to store validation errors
	var validationErrors []*pb.ValidationError

	// Iterate over all received average outputs
	for _, avg := range req.AverageOutputs {
		logger.Default().Info("Validating project", "projectId", avg.ProjectId, "Average", avg.AverageOutput)

		// Check if a summary exists for this project, if not, create one
		summary, exists := s.Summaries[avg.ProjectId]
		if !exists {
			summary = &Summary{
				ProjectID:         avg.ProjectId,
				TimeStarted:       summary.TimeStarted,
				ContractThreshold: avg.ContractThreshold,
				ViolationRecords:            []ViolationRecord{},
			}
			s.Summaries[avg.ProjectId] = summary
		}

		// Check if the average output violates the contract threshold
		if avg.Baseline-avg.AverageOutput < avg.ContractThreshold {
			logger.Default().Info("Validation not met for project", "projectId", avg.ProjectId, "Threshold", avg.ContractThreshold)

			// Add a validation error for the project
			validationErrors = append(validationErrors, &pb.ValidationError{
				ProjectId: avg.ProjectId,
				Message:   "Validation is below the threshold",
			})

			// Add a fault record
			fault := ViolationRecord{
				StartTime: avg.StartTime,
				EndTime:   avg.EndTime,
				Average:   avg.AverageOutput,
			}
			summary.ViolationRecords = append(summary.ViolationRecords, fault)
		}
	}

	// Create the response
	response := &pb.ValidateAverageOutputsResponse{
		Success: len(validationErrors) == 0, // Success is true if no validation errors occurred
		Errors:  validationErrors,
	}

	// Log any threshold violations
	for projectId, summary := range s.Summaries {
		logger.Default().Info("Project Summary", "projectId", projectId, "Total Violations", len(summary.ViolationRecords))
	}

	return response, nil
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
