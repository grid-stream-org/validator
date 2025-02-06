package report

import (
	"fmt"
	"io"
	"net/http"

	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

//API needs to change
const apiURL = "http://your-api-url.com/user-email?projectID="

func SendUserReports(v types.Validator, projectID string) error {
    userEmail, err := getUserEmail(projectID)
    if err != nil {
        fmt.Println("Error fetching user email:", err)
        return err
    }

    reportContent := GenerateReport(v, projectID)

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

//Send Email to user
func sendEmail(to, subject, body string) error {
	cfg, err := config.Load()
	if err != nil{
		return err
	}
	from := mail.NewEmail("GridStream Reports", cfg.SendGrid.Api)
	toMail := mail.NewEmail(to, to)
	content := mail.NewContent("text/plain", body)


	message := mail.NewV3MailInit(from, subject, toMail, content)

    client := sendgrid.NewSendClient(cfg.SendGrid.Api)
	response, err := client.Send(message)
	if err != nil {
		return err
	}
	
    logger.Default().Info("Email sent with code", response)
	return nil

}

func sendTestEmail(config *config.Config) error {
	sendgridAPIKey := config.SendGrid.Api // Ensure this is set in your environment
	if sendgridAPIKey == "" {
		return fmt.Errorf("SendGrid API Key is not set")
	}

	from := mail.NewEmail("GridStream Reports", config.SendGrid.Api)
	to := mail.NewEmail("Test Cooper", "coopdickson@gmail.com")
	subject := "Test Email from Go"
	content := mail.NewContent("text/plain", "Hello, this is a test email from Go using SendGrid!")

	message := mail.NewV3MailInit(from, subject, to, content)

	client := sendgrid.NewSendClient(sendgridAPIKey)
	response, err := client.Send(message)
	if err != nil {
		return err
	}

	fmt.Println("Email sent! Status Code:", response.StatusCode)
	fmt.Println("Response Body:", response.Body)
	fmt.Println("Response Headers:", response.Headers)

	return nil
}