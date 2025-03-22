package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"log/slog"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/types"
)

type Service struct {
	pb.UnimplementedValidatorServiceServer
	summaries *sync.Map
	Log       *slog.Logger
	Cfg       *config.Config
}

// New returns a new Validator gRPC service handler.
func New(cfg *config.Config, log *slog.Logger) *Service {
	return &Service{
		Log:       log.With("component", "handler"),
		summaries: new(sync.Map),
		Cfg:       cfg,
	}
}

// ValidateAverageOutputs implements the ValidatorService RPC.
func (s *Service) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	s.Log.Info("ValidateAverageOutputs called", "project_count", len(req.AverageOutputs))

	// Check if there are any average outputs in the request
	if len(req.AverageOutputs) == 0 {
		s.Log.Info("No averages found")
		return &pb.ValidateAverageOutputsResponse{
			Success: false,
			Errors:  []*pb.ValidationError{},
		}, nil
	}

	var validationErrors []*pb.ValidationError

	// Iterate over all received average outputs
	for _, avg := range req.AverageOutputs {
		logger.Default().Info("Validating project", "projectId", avg.ProjectId, "Average", avg.AverageOutput)

		summaryIface, exists := s.summaries.Load(avg.ProjectId)

		var summary *types.Summary
		// sync map Load() returns interface so we need to do some bs casting
		if exists {
			summary = summaryIface.(*types.Summary)
		} else {
			summary = &types.Summary{
				ProjectID:         avg.ProjectId,
				TimeStarted:       avg.StartTime,
				ContractThreshold: avg.ContractThreshold,
				ViolationRecords:  []types.ViolationRecord{},
			}
			s.summaries.Store(avg.ProjectId, summary)
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

			notification := &types.FaultNotification{
				ProjectID: avg.ProjectId,
				Message:   "Validation is below the threshold",
				StartTime: avg.StartTime,
				EndTime:   avg.EndTime,
				Average:   avg.AverageOutput,
			}
			go s.sendFaultNotification(notification)
		}
	}

	response := &pb.ValidateAverageOutputsResponse{
		Success: len(validationErrors) == 0,
		Errors:  validationErrors,
	}

	s.summaries.Range(func(key, value any) bool {
		summary := value.(*types.Summary)
		projectID := key.(string)

		s.Log.Info("Project Summary", "projectId", projectID, "Total Violations", len(summary.ViolationRecords))
		return true
	})

	return response, nil
}

func (s *Service) sendFaultNotification(fault *types.FaultNotification) {
	frontendURL := "https://api.gridstream.app/v1/notifications"

	jsonData, err := json.Marshal(fault)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post(frontendURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error sending fault notification:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Fault notification sent successfully, response:", resp.Status)
}

func (s *Service) OnShutdown(ctx context.Context) error {
	s.Log.Info("sending final reports...")

	var err error

	s.summaries.Range(func(key, value any) bool {
		summary := value.(*types.Summary)
		projectID := key.(string)
		if err := report.SendUserReport(s.Cfg, s.Log, summary); err != nil {
			s.Log.Error("failed to send user report", "projectID:", projectID)
			// just log and keep going, YOLO
		}
		return true
	})

	return err
}
