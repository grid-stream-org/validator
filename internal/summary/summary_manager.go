// summary manager manages the summaries sent in from the batcher, instead of using map[string]*types.Summary inside server.go we can use this guy to handle all the shit coming in
// This is going to be so much fun to debug, but doing this also removes the use of the global mutex so prolly a good thing anyways, I should really be doing the frontend instead right now
// helper package to manage all the shit coming in from the Batcher guy, who knows if this shit is going to work I am making so many packages at this point,
//
//	Matt just trust me bro
package summary

import (
	"sync"

	"maps"

	"github.com/grid-stream-org/validator/internal/types"
)

// SummaryManager handles storing and retrieving project summaries safely.
type SummaryManager struct {
	summaries map[string]*types.Summary
	mu        sync.Mutex
}

func NewSummaryManager() *SummaryManager {
	return &SummaryManager{
		summaries: make(map[string]*types.Summary),
	}
}

func (s *SummaryManager) GetSummary(projectID string) (*types.Summary, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary, exists := s.summaries[projectID]
	return summary, exists
}

// I am assuming this means end time of DR Event otherwise we are cooked
func (s *SummaryManager) GetEndTime(projId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary := s.summaries[projId].TimeEnded
	return summary
}

func (s *SummaryManager) SetSummary(projectID string, summary *types.Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.summaries[projectID] = summary
}

func (s *SummaryManager) GetAllSummaries() map[string]*types.Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Returning a copy to avoid race conditions
	copy := make(map[string]*types.Summary, len(s.summaries))
	maps.Copy(copy, s.summaries)
	return copy
}
