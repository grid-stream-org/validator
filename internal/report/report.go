package report

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grid-stream-org/api/pkg/firebase"
	"github.com/grid-stream-org/validator/internal/config"
	"github.com/grid-stream-org/validator/internal/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Reporter struct {
	cfg *config.Config
	fc  firebase.FirebaseClient
	log *slog.Logger
}

func New(cfg *config.Config, fc firebase.FirebaseClient, log *slog.Logger) *Reporter {
	return &Reporter{
		cfg: cfg,
		fc:  fc,
		log: log,
	}
}

func (r *Reporter) SendReport(ctx context.Context, summary *types.Summary) error {
	userEmail, err := r.getUserEmail(ctx, summary.ProjectID)
	if err != nil {
		r.log.Error("error fetching user email", "projectId", summary.ProjectID, "error", err)
		return err
	}

	reportContent := r.generateReport(summary)
	r.log.Info("sending email report", "projectId", summary.ProjectID)

	if err := r.sendEmail(userEmail, reportContent); err != nil {
		r.log.Error("failed to send email", "projectId", summary.ProjectID, "error", err)
		return err
	}

	r.log.Info("email successfully sent", "to", userEmail)
	return nil
}

func (r *Reporter) getUserEmail(ctx context.Context, projectID string) (string, error) {
	usersRef := r.fc.Firestore().Collection("users")
	query := usersRef.Where("projectId", "==", projectID).Limit(1)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return "", fmt.Errorf("error querying users: %w", err)
	}

	if len(docs) == 0 {
		return "", fmt.Errorf("no user found with project ID: %s", projectID)
	}

	userData := docs[0].Data()
	email, ok := userData["email"].(string)
	if !ok {
		return "", fmt.Errorf("user found but email field is missing or invalid")
	}

	return email, nil
}

func (r *Reporter) sendEmail(to, body string) error {
	subject := "Your Demand Response Event Report"
	from := mail.NewEmail("GridStream Reports", r.cfg.SendGrid.Sender)
	toMail := mail.NewEmail(to, to)
	content := mail.NewContent("text/plain", body)
	message := mail.NewV3MailInit(from, subject, toMail, content)

	client := sendgrid.NewSendClient(r.cfg.SendGrid.Api)
	response, err := client.Send(message)
	if err != nil {
		return err
	}

	r.log.Info("email sent", "status", response.StatusCode)
	return nil
}

func (r *Reporter) generateReport(summary *types.Summary) string {
	var sb strings.Builder

	sb.WriteString("Validation Report\n")
	sb.WriteString("-----------------\n")
	sb.WriteString(fmt.Sprintf("Project ID: %s\n", summary.ProjectID))
	sb.WriteString(fmt.Sprintf("Time Started: %s\n", summary.TimeStarted))
	sb.WriteString(fmt.Sprintf("Time Ended: %s\n", summary.TimeEnded))
	sb.WriteString(fmt.Sprintf("Contract Threshold: %.2f\n\n", summary.ContractThreshold))
	sb.WriteString(fmt.Sprintf("Total Violations: %d\n\n", len(summary.ViolationRecords)))

	if len(summary.ViolationRecords) > 0 {
		sb.WriteString("Violation Intervals:\n")
		for _, violation := range summary.ViolationRecords {
			sb.WriteString(fmt.Sprintf("- Start: %s | End: %s | Average: %.2f\n",
				violation.StartTime, violation.EndTime, violation.Average))
		}
	}

	return sb.String()
}
