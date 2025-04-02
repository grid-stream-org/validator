package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"log/slog"

	"github.com/grid-stream-org/api/pkg/firebase"
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

		summaryIface, exists := s.summaries.Load(avg.ProjectId)

		var summary *types.Summary
		// sync map Load() returns interface so we need to do some bs casting
		if exists {
			summary = summaryIface.(*types.Summary)
		} else {
			summary = &types.Summary{
				ProjectID:         avg.ProjectId,
				TimeStarted:       avg.StartTime,
				TimeEnded:         avg.EndTime,
				ContractThreshold: avg.ContractThreshold,
				ViolationRecords:  []types.ViolationRecord{},
			}
			s.summaries.Store(avg.ProjectId, summary)
		}

		// Check if the average output violates the contract threshold
		if avg.Baseline-avg.AverageOutput < avg.ContractThreshold {

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
	}

	return response, nil
}

func (s *Service) sendFaultNotification(ctx context.Context, fault *types.FaultNotification) {
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

	// Create a new background context with timeout to ensure reports can complete
	sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	s.summaries.Range(func(key, value any) bool {
		summary := value.(*types.Summary)
		projectID := key.(string)

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.log.Info("sending report for project", "projectID", projectID)

			if err := s.reporter.SendReport(sendCtx, summary); err != nil {
				s.log.Error("failed to send user report", "projectID", projectID, "error", err)
			} else {
				s.log.Info("successfully sent report", "projectID", projectID)
			}
		}()

		return true
	})

	// Wait for all reports to complete or timeout
	wg.Wait()

	s.log.Info("all reports sent successfully")
	return nil
}
