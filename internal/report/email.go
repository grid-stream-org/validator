package report

import (
	"fmt"
	"io"
	"net/http"

	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/server"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// API needs to change
const apiURL = "http://your-api-url.com/user-email?projectID="

func SendUserReports(v *server.ValidatorServer, projectID string) error {
	userEmail, err := getUserEmail(projectID)
	if err != nil {
		fmt.Println("Error fetching user email:", err)
		return err
	}

	reportContent := GenerateReport(v, projectID)
	logger.Default().Info("SENDING EMAILS", "report content", reportContent)

	err = sendEmail(userEmail, "Your Demand Response Event Report", reportContent)
	if err != nil {
		fmt.Println("Failed to send email:", err)
		return err
	}

	fmt.Println("Email successfully sent to", userEmail)
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
func sendEmail(to, subject, body string) error {
	logger.Default().Info("ATEEMPT TO SEND EMAIL")

	cfg, err := config.Load()
	if err != nil {
		return err
	}
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
