// Validate that contracts are not being breached
package validation

import (
	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/types"
)

func ValidateRequest(req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	logger.Default().Info("Processing validation...")

	if len(req.AverageOutputs) == 0 {
		logger.Default().Info("No averages found")
		return &pb.ValidateAverageOutputsResponse{
			Success: false,
			Errors:  []*pb.ValidationError{},
		}, nil
	}

	var validationErrors []*pb.ValidationError

	for _, avg := range req.AverageOutputs {
		logger.Default().Info("Validating project", "projectId", avg.ProjectId, "Average", avg.AverageOutput)

		if avg.Baseline-avg.AverageOutput < avg.ContractThreshold {
			validationErrors = append(validationErrors, &pb.ValidationError{
				ProjectId: avg.ProjectId,
				Message:   "Validation is below the threshold",
			})

			go report.SendFaultNotification(types.FaultNotification{
				ProjectID: avg.ProjectId,
				Message:   "Contract breached within last 5 minutes",
				StartTime: avg.StartTime,
				EndTime:   avg.EndTime,
				Average:   avg.AverageOutput,
			})
		}
	}

	return &pb.ValidateAverageOutputsResponse{
		Success: len(validationErrors) == 0,
		Errors:  validationErrors,
	}, nil
}
