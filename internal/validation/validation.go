// Perform validation during DR events
package validation

import (
	"context"
	"time"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/notification"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/server"
	"github.com/grid-stream-org/validator/internal/types"
)

// ValidateAverageOutputs performs validation of average outputs.
func ValidateAverageOutputs(ctx context.Context, s *server.ValidatorServer, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	logger.Default().Info("Received validation request")

	// Check if there are any average outputs in the request
	if len(req.AverageOutputs) == 0 {
		logger.Default().Info("No averages found")
		return &pb.ValidateAverageOutputsResponse{
			Success: false,
			Errors:  []*pb.ValidationError{},
		}, nil
	}

	s.Lock() // Lock to prevent concurrent map modification
	defer s.Unlock()

	// Ensure the map is initialized before using it
	if s.Summaries == nil {
		s.Summaries = make(map[string]*types.Summary)
	}
	s.LastRequestTime = time.Now() // Reset the timer
	go report.MonitorDREvent(s)    // Start monitoring if not already

	var validationErrors []*pb.ValidationError

	// Iterate over all received average outputs
	for _, avg := range req.AverageOutputs {
		logger.Default().Info("Validating project", "projectId", avg.ProjectId, "Average", avg.AverageOutput)

		summary, exists := s.Summaries[avg.ProjectId]
		if !exists {
			summary = &types.Summary{
				ProjectID:         avg.ProjectId,
				TimeStarted:       avg.StartTime,
				ContractThreshold: avg.ContractThreshold,
				ViolationRecords:  []types.ViolationRecord{},
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
			fault := types.ViolationRecord{
				StartTime: avg.StartTime,
				EndTime:   avg.EndTime,
				Average:   avg.AverageOutput,
			}
			summary.ViolationRecords = append(summary.ViolationRecords, fault)

			// Notify the frontend
			notification.SendFaultNotification(&types.FaultNotification{
				ProjectID: avg.ProjectId,
				Message:   "Validation is below the threshold",
				StartTime: avg.StartTime,
				EndTime:   avg.EndTime,
				Average:   avg.AverageOutput,
			})
		}
	}

	response := &pb.ValidateAverageOutputsResponse{
		Success: len(validationErrors) == 0,
		Errors:  validationErrors,
	}

	// Log project summaries
	for projectId, summary := range s.Summaries {
		logger.Default().Info("Project Summary", "projectId", projectId, "Total Violations", len(summary.ViolationRecords))
	}

	return response, nil
}
