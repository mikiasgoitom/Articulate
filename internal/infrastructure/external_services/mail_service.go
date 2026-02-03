package external_services

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
)

// smtp attribute
type EmailService struct {
	Host        string
	Port        string
	Username    string
	AppPassword string
	From        string
}

// EmailService factory
func NewEmailService(host, port, username, appPassword, from string) *EmailService {
	return &EmailService{
		Host:        host,
		Port:        port,
		Username:    username,
		AppPassword: appPassword,
		From:        from,
	}
}

// make sure EmailService implements contract.IEmailService.go
var _ contract.IEmailService = (*EmailService)(nil)

// send email method
func (es *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	// write the msg header
	msg := []byte(
		fmt.Sprintf(
			"To: %s\r\n"+
				"From: %s\r\n"+
				"Subject: %s\r\n"+
				"\r\n"+
				"%s\r\n",
			to, es.From, subject, body,
		),
	)
	// smtp auth
	auth := smtp.PlainAuth("", es.Username, es.AppPassword, es.Host)
	// send address
	addr := fmt.Sprintf("%s:%s", es.Host, es.Port)
	err := smtp.SendMail(addr, auth, es.From, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email via Gmail SMTP: %w", err)
	}
	return nil
}
