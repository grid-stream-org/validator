package report

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/grid-stream-org/validator/internal/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

//API needs to change
const apiURL = "http://your-api-url.com/user-email?projectID="
var sendgridAPIKey = os.Getenv("SG_API")

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
    from := mail.NewEmail("GridStream", "coopdickson@gmail.com")
    toEmail := mail.NewEmail("Cooper Dickson", to)
    message := mail.NewSingleEmail(from, subject, toEmail, body, body)

    client := sendgrid.NewSendClient(sendgridAPIKey)
    _, err := client.Send(message)
    return err
}