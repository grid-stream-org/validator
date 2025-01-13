package client

import (
	"context"
	"log"
	"time"

	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1" // Import the generated protobuf package

	"google.golang.org/grpc"
)

// ValidatorClient handles communication with the ValidatorService
type ValidatorClient struct {
	client pb.ValidatorServiceClient
}

// NewValidatorClient creates a new ValidatorClient
func NewValidatorClient(serverAddress string) (*ValidatorClient, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	return &ValidatorClient{
		client: pb.NewValidatorServiceClient(conn),
	}
}

// ValidateAverageOutputs sends a validation request to the ValidatorService
func (vc *ValidatorClient) ValidateAverageOutputs(averages []*pb.AverageOutput) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.ValidateAverageOutputsRequest{
		Averages: averages,
	}

	resp, err := vc.client.ValidateAverageOutputs(ctx, req)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	log.Printf("Validation success: %v, Message: %s", resp.GetSuccess(), resp.GetMessage())
	for _, err := range resp.GetErrors() {
		log.Printf("Error - Project ID: %s, Message: %s", err.GetProjectId(), err.GetMessage())
	}
}
