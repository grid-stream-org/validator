package server

import (
	"context"
	"log"
	"net"

	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1" // Import the generated protobuf package
	"google.golang.org/grpc"
)

// ValidatorServer implements the ValidatorService gRPC interface
type ValidatorServer struct {
	pb.UnimplementedValidatorServiceServer
}

// ValidateAverageOutputs handles validation requests
func (s *ValidatorServer) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	log.Println("Received validation request")


	// logic
	var errors []*pb.ValidationError
	for _, avg := range req.Averages {
		if avg.AverageOutput < 50.0 { 
			errors = append(errors, &pb.ValidationError{
				ProjectId: avg.ProjectId,
				Message:   "Average output below threshold",
			})
		}
	}

	success := len(errors) == 0
	message := "Validation successful"
	if !success {
		message = "Validation failed"
	}

	response := &pb.ValidateAverageOutputsResponse{
		Success: success,
		Message: message,
		Errors:  errors,
	}

	log.Printf("Validation completed: %s", message)
	return response, nil
}

// StartValidatorServer starts the gRPC server for the ValidatorService
func StartValidatorServer(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterValidatorServiceServer(grpcServer, &ValidatorServer{})

	log.Printf("Validator server is listening on %s", address)
	return grpcServer.Serve(lis)
}
