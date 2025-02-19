package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1" // Import the generated protobuf package
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/report"
	"github.com/grid-stream-org/validator/internal/types"
	"google.golang.org/grpc"
)

type ValidatorServer struct {
    pb.UnimplementedValidatorServiceServer
    Summaries       map[string]*types.Summary
    Mu              sync.Mutex
    lastRequestTime time.Time
}

type FaultNotification struct {
    ProjectID string  `json:"project_id"`
    Message   string  `json:"message"`
    StartTime string  `json:"start_time"`
    EndTime   string  `json:"end_time"`
    Average   float64 `json:"average"`
}

func sendFaultNotification(fault FaultNotification) {
    frontendURL := "http://frontend-app.com/api/faults" // NEEDS TO BE REPLACED

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

func (s *ValidatorServer) GetSummary(projectID string) (*types.Summary, bool) {
    summary, exists := s.Summaries[projectID]
    return summary, exists
}

func (s *ValidatorServer) GetSummaryMutex() *sync.Mutex {
    return &s.Mu
}

func (s *ValidatorServer) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
    logger.Default().Info("Received validation request")

    // Check if there are any average outputs in the request
    if len(req.AverageOutputs) == 0 {
        logger.Default().Info("No averages found")
        return &pb.ValidateAverageOutputsResponse{
            Success: false,
            Errors:  []*pb.ValidationError{},
        }, nil
    }

    s.Mu.Lock() // Lock to prevent concurrent map modification
    defer s.Mu.Unlock()

    // Ensure the map is initialized before using it
    if s.Summaries == nil {
        s.Summaries = make(map[string]*types.Summary) 
    }
    s.lastRequestTime = time.Now() // Reset the timer
    go s.monitorDREvent() // Start monitoring if not already

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
            notification := FaultNotification{
                ProjectID: avg.ProjectId,
                Message:   "Validation is below the threshold",
                StartTime: avg.StartTime,
                EndTime:   avg.EndTime,
                Average:   avg.AverageOutput,
            }
            go sendFaultNotification(notification)
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


func (s *ValidatorServer) monitorDREvent() {
    for {
        time.Sleep(1 * time.Minute) 

        s.Mu.Lock()
        if time.Since(s.lastRequestTime) > 1*time.Minute {
            s.Mu.Unlock()

            // Generate & Send Reports
            for projectId, summary := range s.Summaries {
                logger.Default().Info("Sending Email for", "projectId", projectId)
                summary.TimeEnded = time.Now().Format(time.RFC3339) 
                report.SendUserReports(s, projectId)
            } 
            continue 
        }
        s.Mu.Unlock()
    }
}


func Start(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	address := cfg.Server.Address // Dynamically read server address from config

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterValidatorServiceServer(grpcServer, &ValidatorServer{})

	// Run the server in a goroutine to allow for graceful shutdown
	go func() {
		log.Info("Validator server is running on ", "address", address)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC server", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	return nil
}
