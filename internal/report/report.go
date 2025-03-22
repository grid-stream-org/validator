package report

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// API needs to change
const apiURL = "http://your-api-url.com/user-email?projectID="

func SendUserReport(cfg *config.Config, log *slog.Logger, summary *types.Summary) error {
	projectID := summary.ProjectID
	userEmail, err := getUserEmail(projectID)
	if err != nil {
		log.Error("error fetching user email", "projectId", projectID, "error", err)
		return err
	}

	reportContent := generateReport(summary)
	log.Info("sending email report", "projectId", projectID)

	if err := sendEmail(cfg, log, userEmail, reportContent); err != nil {
		log.Error("failed to send email", "projectId", projectID, "error", err)
		return err
	}

	log.Info("email successfully sent", "to", userEmail)
	return nil
}

// getUserEmail queries the API for the user's email based on the project ID
func getUserEmail(projectID string) (string, error) {

	response, err := http.Get(apiURL + projectID)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d", response.StatusCode)
	}

	email, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(email), nil
}

// Send Email to user
func sendEmail(cfg *config.Config, log *slog.Logger, to, body string) error {
	log.Info("Sending Email")
	subject := "Your Demand Response Event Report"

	from := mail.NewEmail("GridStream Reports", cfg.SendGrid.Sender)
	toMail := mail.NewEmail(to, to)
	content := mail.NewContent("text/plain", body)

	message := mail.NewV3MailInit(from, subject, toMail, content)

	client := sendgrid.NewSendClient(cfg.SendGrid.Api)
	response, err := client.Send(message)
	if err != nil {
		return err
	}

	logger.Default().Info("EMAIL SENT", "response", response)
	return nil

}

func generateReport(summary *types.Summary) string {

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
