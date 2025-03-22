package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"log/slog"

	"github.com/grid-stream-org/api/pkg/firebase"
	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/types"
)

type Service struct {
	pb.UnimplementedValidatorServiceServer
	summaries *sync.Map
	fc        firebase.FirebaseClient
	reporter  *report.Reporter
	log       *slog.Logger
	cfg       *config.Config
}

// New returns a new Validator gRPC service handler.
func New(cfg *config.Config, fc firebase.FirebaseClient, log *slog.Logger) *Service {
	return &Service{
		log:       log.With("component", "handler"),
		reporter:  report.New(cfg, fc, log),
		fc:        fc,
		summaries: new(sync.Map),
		cfg:       cfg,
	}
}

// ValidateAverageOutputs implements the ValidatorService RPC.
func (s *Service) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	s.log.Info("ValidateAverageOutputs called", "project_count", len(req.AverageOutputs))

	// Check if there are any average outputs in the request
	if len(req.AverageOutputs) == 0 {
		s.log.Info("No averages found")
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
			go s.sendFaultNotification(ctx, notification)
		}
	}

	response := &pb.ValidateAverageOutputsResponse{
		Success: len(validationErrors) == 0,
		Errors:  validationErrors,
	}

	s.summaries.Range(func(key, value any) bool {
		summary := value.(*types.Summary)
		projectID := key.(string)

		s.log.Info("Project Summary", "projectId", projectID, "Total Violations", len(summary.ViolationRecords))
		return true
	})

	return response, nil
}

func (s *Service) sendFaultNotification(ctx context.Context, fault *types.FaultNotification) {
	// Create a custom token for your service
	customToken, err := s.fc.Auth().CustomToken(ctx, "validator-service")
	if err != nil {
		s.log.Error("Error creating custom token", "error", err)
		return
	}

	// Exchange for ID token
	exchangeURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=%s", s.cfg.WebAPIKey)

	exchangeData := map[string]string{
		"token":             customToken,
		"returnSecureToken": "true",
	}

	exchangeJSON, err := json.Marshal(exchangeData)
	if err != nil {
		s.log.Error("Error marshalling token request", "error", err)
		return
	}

	exchangeResp, err := http.Post(exchangeURL, "application/json", bytes.NewBuffer(exchangeJSON))
	if err != nil {
		s.log.Error("Error exchanging token", "error", err)
		return
	}
	defer exchangeResp.Body.Close()

	var tokenResponse struct {
		IdToken string `json:"idToken"`
	}

	if err := json.NewDecoder(exchangeResp.Body).Decode(&tokenResponse); err != nil {
		s.log.Error("Error decoding token response", "error", err)
		return
	}

	// Make the API request with the token
	frontendURL := "https://api.gridstream.app/v1/notifications"
	jsonData, err := json.Marshal(fault)
	if err != nil {
		s.log.Error("Error marshalling JSON", "error", err)
		return
	}

	req, err := http.NewRequest("POST", frontendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error("Error creating request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenResponse.IdToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.log.Error("Error sending fault notification", "error", err)
		return
	}
	defer resp.Body.Close()

	s.log.Info("Fault notification sent successfully", "status", resp.Status)
}

func (s *Service) OnShutdown(ctx context.Context) error {
	s.log.Info("sending final reports...")

	var err error

	s.summaries.Range(func(key, value any) bool {
		summary := value.(*types.Summary)
		projectID := key.(string)
		if err := s.reporter.SendReport(ctx, summary); err != nil {
			s.log.Error("failed to send user report", "projectID", projectID)
			// just log and keep going, YOLO
		}
		return true
	})

	return err
}
