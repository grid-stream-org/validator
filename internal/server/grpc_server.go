// grpc_server.go is responsible for handling incoming requests, it creates a grpc server and registers validator_server
package server

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/types"
	"google.golang.org/grpc"
)

type Server struct {
	GRPCServer      *grpc.Server
	ValidatorServer *ValidatorServer
	Listener        net.Listener
}

type ValidatorServer struct {
	pb.UnimplementedValidatorServiceServer
	summaries       map[string]*types.Summary
	mu              sync.Mutex
	lastRequestTime time.Time
}

func NewValidatorServer() *ValidatorServer {
	return &ValidatorServer{
		summaries: make(map[string]*types.Summary),
	}
}

func NewGRPCServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	// Create the network listener
	listener, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		return nil, err
	}

	// Create the gRPC server
	grpcServer := grpc.NewServer()

	// Create the Validator server, server for the business logic
	validatorServer := NewValidatorServer()

	// Register ValidatorServer with gRPC
	pb.RegisterValidatorServiceServer(grpcServer, validatorServer)

	return &Server{
		GRPCServer:      grpcServer,
		ValidatorServer: validatorServer,
		Listener:        listener,
	}, nil
}

func StartMonitor(vs *ValidatorService) {
	for {
		time.Sleep(1 * time.Minute)

		vs.mu.Lock()
		if time.Since(vs.LastRequestTime()) > 1*time.Minute {
			vs.mu.Unlock()

			// Generate & send reports
			for projectId, summary := range vs.GetSummaries() {
				logger.Default().Info("Sending Email for", "projectId", projectId)
				summary.TimeEnded = time.Now().Format(time.RFC3339)
				report.SendUserReports(summary, projectId)
			}
			continue
		}
		vs.mu.Unlock()
	}
}

// func sendFaultNotification(fault types.FaultNotification) {
// 	apiURL := "https://api.gridstream.app/v1/notifications"

// 	jsonData, err := json.Marshal(fault)
// 	if err != nil {
// 		log.Println("Error marshalling JSON:", err)
// 		return
// 	}

// 	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		log.Println("Error sending fault notification:", err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	log.Println("Fault notification sent successfully, response:", resp.Status)
// }

// func (s *ValidatorServer) GetSummary(projectID string) (*types.Summary, bool) {
// 	summary, exists := s.summaries[projectID]
// 	return summary, exists
// }

// func (s *ValidatorServer) GetSummaryMutex() *sync.Mutex {
// 	return &s.mu
// }

// func (s *ValidatorServer) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
// 	logger.Default().Info("Received validation request")

// 	// Check if there are any average outputs in the request
// 	if len(req.AverageOutputs) == 0 {
// 		logger.Default().Info("No averages found")
// 		return &pb.ValidateAverageOutputsResponse{
// 			Success: false,
// 			Errors:  []*pb.ValidationError{},
// 		}, nil
// 	}

// 	s.Mu.Lock() // Lock to prevent concurrent map modification
// 	defer s.Mu.Unlock()

// 	// Ensure the map is initialized before using it
// 	if s.Summaries == nil {
// 		s.Summaries = make(map[string]*types.Summary)
// 	}
// 	s.lastRequestTime = time.Now() // Reset the timer
// 	go s.monitorDREvent()          // Start monitoring if not already

// 	var validationErrors []*pb.ValidationError

// 	// Iterate over all received average outputs
// 	for _, avg := range req.AverageOutputs {
// 		logger.Default().Info("Validating project", "projectId", avg.ProjectId, "Average", avg.AverageOutput)

// 		summary, exists := s.Summaries[avg.ProjectId]
// 		if !exists {
// 			summary = &types.Summary{
// 				ProjectID:         avg.ProjectId,
// 				TimeStarted:       avg.StartTime,
// 				ContractThreshold: avg.ContractThreshold,
// 				ViolationRecords:  []types.ViolationRecord{},
// 			}
// 			s.Summaries[avg.ProjectId] = summary
// 		}

// 		// Check if the average output violates the contract threshold
// 		if avg.Baseline-avg.AverageOutput < avg.ContractThreshold {
// 			logger.Default().Info("Validation not met for project", "projectId", avg.ProjectId, "Threshold", avg.ContractThreshold)

// 			// Add a validation error for the project
// 			validationErrors = append(validationErrors, &pb.ValidationError{
// 				ProjectId: avg.ProjectId,
// 				Message:   "Validation is below the threshold",
// 			})

// 			// Add a fault record
// 			fault := types.ViolationRecord{
// 				StartTime: avg.StartTime,
// 				EndTime:   avg.EndTime,
// 				Average:   avg.AverageOutput,
// 			}
// 			summary.ViolationRecords = append(summary.ViolationRecords, fault)

// 			// Notify the frontend
// 			notification := types.FaultNotification{
// 				ProjectID: avg.ProjectId,
// 				Message:   "Validation is below the threshold",
// 				StartTime: avg.StartTime,
// 				EndTime:   avg.EndTime,
// 				Average:   avg.AverageOutput,
// 			}
// 			go sendFaultNotification(notification)
// 		}
// 	}

// 	response := &pb.ValidateAverageOutputsResponse{
// 		Success: len(validationErrors) == 0,
// 		Errors:  validationErrors,
// 	}

// 	// Log project summaries
// 	for projectId, summary := range s.Summaries {
// 		logger.Default().Info("Project Summary", "projectId", projectId, "Total Violations", len(summary.ViolationRecords))
// 	}

// 	return response, nil
// }

// func (s *ValidatorServer) monitorDREvent() {
// 	for {
// 		time.Sleep(1 * time.Minute)

// 		s.Mu.Lock()
// 		if time.Since(s.lastRequestTime) > 1*time.Minute {
// 			s.Mu.Unlock()

// 			// Generate & Send Reports
// 			for projectId, summary := range s.Summaries {
// 				logger.Default().Info("Sending Email for", "projectId", projectId)
// 				summary.TimeEnded = time.Now().Format(time.RFC3339)
// 				report.SendUserReports(s, projectId)
// 			}
// 			continue
// 		}
// 		s.Mu.Unlock()
// 	}
// }
