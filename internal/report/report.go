package report

import (
	"fmt"

	"github.com/grid-stream-org/validator/internal/server"
)

func GenerateReport(s *server.ValidatorServer, projectID string) string {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	summary, exists := s.Summaries[projectID]
	if !exists {
		return "No validation data available for this project."
	}

	report := "Validation Report\n"
	report += "-----------------\n"
	report += "Project ID: " + summary.ProjectID + "\n"
	report += "Time Started: " + summary.TimeStarted + "\n"
	report += "Time Ended: " + summary.TimeEnded + "\n"
	report += "Contract Threshold: " + formatFloat(summary.ContractThreshold) + "\n\n"
	report += "Total Violations: " + formatInt(len(summary.ViolationRecords)) + "\n\n"

	if len(summary.ViolationRecords) > 0 {
		report += "Violation Intervals:\n"
		for _, violation := range summary.ViolationRecords {
			report += "- Start: " + violation.StartTime + " | End: " + violation.EndTime + " | Average: " + formatFloat(violation.Average) + "\n"
		}
	}

	return report
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}