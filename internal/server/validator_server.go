// validator service is responsible for delegating business logic to other packages, registered to grpc server. grpc server forwards requests here
package server

import (
	"context"
	"sync"
	"time"

	"github.com/grid-stream-org/batcher/pkg/logger"
	pb "github.com/grid-stream-org/grid-stream-protos/gen/validator/v1"
	"github.com/grid-stream-org/validator/internal/types"
	"github.com/grid-stream-org/validator/internal/validation"
)

type ValidatorService struct {
	pb.UnimplementedValidatorServiceServer
	mu              sync.Mutex
	summaries       map[string]*types.Summary
	lastRequestTime time.Time
}

func NewValidatorService() *ValidatorService {
	return &ValidatorService{
		summaries: make(map[string]*types.Summary),
	}
}

func (s *ValidatorService) GetSummaries() map[string]types.Summary {
	s.mu.Lock()
	defer s.mu.Unlock()

	copySummaries := make(map[string]types.Summary)
	for k, v := range s.summaries {
		copySummaries[k] = *v
	}
	return copySummaries
}

func (s *ValidatorService) LastRequestTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRequestTime
}

func (s *ValidatorService) ValidateAverageOutputs(ctx context.Context, req *pb.ValidateAverageOutputsRequest) (*pb.ValidateAverageOutputsResponse, error) {
	logger.Default().Info("Received validation request")

	s.mu.Lock()
	s.lastRequestTime = time.Now()
	s.mu.Unlock()

	// Delegate actual validation logic
	return validation.ValidateRequest(req)
}
