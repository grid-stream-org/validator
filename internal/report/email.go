package report

import (
	"fmt"
	"net/smtp"
	"os"
)

func sendEmail(to string, reportContent string) error {

	//need to get details from DB???
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	smtpServer := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, smtpServer)
	toEmails := []string{to}

	// Email message format
	msg := []byte(fmt.Sprintf(
		"Subject: Validation Report\n\n%s",
		reportContent,
	))

	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, from, toEmails, msg)
	if err != nil {
		return err
	}

	return nil
}