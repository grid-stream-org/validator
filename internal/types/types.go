package types

import "sync"

// Summary struct to store project-wide validation info
type Summary struct {
    ProjectID         string
    TimeStarted       string
    TimeEnded         string
    ContractThreshold float64
    ViolationRecords  []ViolationRecord
}

type ViolationRecord struct {
    StartTime string
    EndTime   string
    Average   float64
}

type Validator interface {
    GetSummary(projectID string) (*Summary, bool)
    GetSummaryMutex() *sync.Mutex
}

type FaultNotification struct {
    ProjectID string  `json:"project_id"`
    Message   string  `json:"message"`
    StartTime string  `json:"start_time"`
    EndTime   string  `json:"end_time"`
    Average   float64 `json:"average"`
}