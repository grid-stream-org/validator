// report manager is an infinite for loop waiting to send reports about dr events. uses report.go to send email
package report

// MonitorDREvent periodically checks for overdue validation reports.
// func MonitorDREvent(s *server.ValidatorServer) {
// 	for {
// 		time.Sleep(1 * time.Minute)

// 		// Get the last request time
// 		lastRequestTime := s.SummaryManager.GetLastRequestTime()

// 		if time.Since(lastRequestTime) > 1*time.Minute {
// 			// Generate & Send Reports
// 			for projectId, summary := range s.SummaryManager.GetAllSummaries() {
// 				slog.Info("Sending Email for", "projectId", projectId)
// 				summary.TimeEnded = time.Now().Format(time.RFC3339)
// 				SendUserReports(s.SummaryManager, projectId)
// 			}
// 		}
// 	}
// }
