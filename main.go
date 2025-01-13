package main

import (
	"log"

	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"validator/client"
)

func main() {
	// Validator service address
	serverAddress := "localhost:50051"

	// Create a new ValidatorClient
	validatorClient, err := client.NewValidatorClient(serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to ValidatorService: %v", err)
	}

	// Sample data to validate
	averages := []*pb.AverageOutput{
		{
			ProjectId:     "project_123",
			AverageOutput: 50.0,
			StartTime:     "2023-12-01T00:00:00Z",
			EndTime:       "2023-12-31T23:59:59Z",
		},
		{
			ProjectId:     "project_456",
			AverageOutput: 75.0,
			StartTime:     "2023-12-01T00:00:00Z",
			EndTime:       "2023-12-31T23:59:59Z",
		},
	}

	// Send validation request
	validatorClient.ValidateAverageOutputs(averages)
}
