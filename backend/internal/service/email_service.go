// Package service implements the business logic layer of CodeTasker.
// email_service.go provides SMTP-based email notification delivery.
package service

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/codetasker/backend/internal/config"
	"go.uber.org/zap"
)

// EmailService sends transactional email notifications via SMTP.
type EmailService struct {
	cfg *config.Config
	log *zap.Logger
}

// NewEmailService constructs an EmailService.
func NewEmailService(cfg *config.Config, log *zap.Logger) *EmailService {
	return &EmailService{cfg: cfg, log: log}
}

// SendTaskAssigned sends a "You've been assigned to a task" notification email.
// It is a no-op when SMTPEnabled is false.
func (s *EmailService) SendTaskAssigned(toEmail, toName, assignerName, taskContent, repoName, frontendURL string) error {
	if !s.cfg.SMTPEnabled {
		return nil
	}
	subject := fmt.Sprintf("[CodeTasker] You've been assigned to a task in %s", repoName)
	body := fmt.Sprintf(`Hello %s,

%s has assigned you to the following task in %s:

  "%s"

Visit CodeTasker to view and manage this task:
%s

---
CodeTasker — Task management for developers
`, toName, assignerName, repoName, taskContent, frontendURL)
	return s.send(toEmail, subject, body)
}

// SendCommentNotification notifies a user that someone commented on their task.
// It is a no-op when SMTPEnabled is false.
func (s *EmailService) SendCommentNotification(toEmail, toName, commenterName, taskContent, comment, repoName, frontendURL string) error {
	if !s.cfg.SMTPEnabled {
		return nil
	}
	subject := fmt.Sprintf("[CodeTasker] New comment on your task in %s", repoName)
	body := fmt.Sprintf(`Hello %s,

%s commented on a task in %s:

Task: "%s"

Comment:
  %s

Visit CodeTasker to reply:
%s

---
CodeTasker — Task management for developers
`, toName, commenterName, repoName, taskContent, comment, frontendURL)
	return s.send(toEmail, subject, body)
}

// send is the internal SMTP delivery method. It constructs the MIME headers,
// authenticates with PlainAuth, and delivers the message.
func (s *EmailService) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	headers := strings.Join([]string{
		"From: " + s.cfg.SMTPFrom,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
	}, "\r\n")
	message := []byte(headers + "\r\n\r\n" + body)

	if err := smtp.SendMail(addr, auth, s.cfg.SMTPUsername, []string{to}, message); err != nil {
		s.log.Error("failed to send email", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("send email to %s: %w", to, err)
	}

	s.log.Info("email sent", zap.String("to", to), zap.String("subject", subject))
	return nil
}
