package server

import (
	"context"
	"log/slog"
	"net"

	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/summary"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
)

// ValidatorServer implements the gRPC service.

// Server struct encapsulates all server components.
type Server struct {
	cfg          *config.Config
	log          *slog.Logger
	grpcServer   *grpc.Server
	validatorSvc *ValidatorServer
	listener     net.Listener
}

type ValidatorServer struct {
	pb.UnimplementedValidatorServiceServer
	SummaryManager *summary.SummaryManager
}

// New initializes and returns a new Server instance.
func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Server, error) {
	address := cfg.Server.Address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create listener")
	}

	grpcServer := grpc.NewServer()
	validatorSvc := &ValidatorServer{}

	pb.RegisterValidatorServiceServer(grpcServer, validatorSvc)

	return &Server{
		cfg:          cfg,
		log:          log.With("component", "grpc-server"),
		grpcServer:   grpcServer,
		validatorSvc: validatorSvc,
		listener:     lis,
	}, nil
}

// Run starts the gRPC server and handles its lifecycle.
func (s *Server) Run(ctx context.Context) (err error) {
	s.log.Info("starting validator server", "address", s.listener.Addr().String())

	// Handle shutdown on exit
	defer func() {
		if stopErr := s.Stop(ctx); stopErr != nil {
			err = multierr.Combine(err, stopErr)
		}
	}()

	// Run the gRPC server in a goroutine
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			s.log.Error("failed to serve gRPC", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	if ctx.Err() != nil {
		err = multierr.Combine(err, ctx.Err())
	}
	return err
}

// Lock locks the ValidatorServer's mutex
// func (s *ValidatorServer) Lock() {
// 	s.mu.Lock()
// }

// // Unlock unlocks the ValidatorServer's mutex
// func (s *ValidatorServer) Unlock() {
// 	s.mu.Unlock()
// }

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("shutting down validator server")

	// Gracefully stop the gRPC server
	s.grpcServer.GracefulStop()

	// Set listener to nil before closing to prevent double close
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return errors.WithStack(err)
		}
	}

	s.log.Info("server shutdown complete")
	return nil
}

// func (s *ValidatorServer) MonitorDREvent() {
// 	for {
// 		time.Sleep(1 * time.Minute)

// 		s.Lock()
// 		if time.Since(s.LastRequestTime) > 1*time.Minute {
// 			s.Unlock()

// 			// Generate & Send Reports
// 			for projectId, summary := range s.Summaries {
// 				logger.Default().Info("Sending Email for", "projectId", projectId)
// 				summary.TimeEnded = time.Now().Format(time.RFC3339)
// 				report.SendUserReports(s, projectId)
// 			}
// 			continue
// 		}
// 		s.Unlock()
// 	}
// }

// func sendFaultNotification(fault *types.FaultNotification) {
// 	frontendURL := "http://frontend-app.com/api/faults" // NEEDS TO BE REPLACED

// 	jsonData, err := json.Marshal(fault)
// 	if err != nil {
// 		log.Println("Error marshalling JSON:", err)
// 		return
// 	}

// 	resp, err := http.Post(frontendURL, "application/json", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		log.Println("Error sending fault notification:", err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	log.Println("Fault notification sent successfully, response:", resp.Status)
// }

// func (s *ValidatorServer) GetSummary(projectID string) (*types.Summary, bool) {
// 	summary, exists := s.Summaries[projectID]
// 	return summary, exists
// }

// // func (s *ValidatorServer) GetSummaryMutex() *sync.Mutex {
// // 	return &s.Mu
// // }

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
// 	s.LastRequestTime = time.Now() // Reset the timer
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
// 			notification := &types.FaultNotification{
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
