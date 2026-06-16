// Package service implements the business logic layer of CodeTasker.
// email_service.go provides SMTP-based email notification delivery.
package service

import (
	"crypto/tls"
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

// SendTaskAssigned sends a "You've been assigned to a task" notification email in HTML format.
// It is a no-op when SMTPEnabled is false or recipient email is empty.
func (s *EmailService) SendTaskAssigned(toEmail, toName, assignerName, taskContent, repoName, frontendURL string) error {
	if !s.cfg.SMTPEnabled {
		s.log.Info("SMTP disabled, skipping SendTaskAssigned", zap.String("toName", toName))
		return nil
	}
	if toEmail == "" {
		s.log.Info("Recipient email is empty, skipping SendTaskAssigned", zap.String("toName", toName))
		return nil
	}
	if frontendURL == "" {
		frontendURL = s.cfg.FrontendURL
	}
	subject := fmt.Sprintf("[CodeTasker] You've been assigned to a task in %s", repoName)
	bodyText := fmt.Sprintf("<strong>%s</strong> assigned you to a new task in <strong>%s</strong>:<br><br><strong>\"%s\"</strong>", assignerName, repoName, taskContent)
	
	bodyHTML := strings.ReplaceAll(emailHTMLTemplate, "{{RECIPIENT_NAME}}", toName)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{PROJECT_NAME}}", repoName)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{BODY_TEXT}}", bodyText)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{ACTION_URL}}", frontendURL)

	return s.send(toEmail, subject, bodyHTML)
}

// SendCommentNotification notifies a user that someone commented on their task.
// It is a no-op when SMTPEnabled is false or recipient email is empty.
func (s *EmailService) SendCommentNotification(toEmail, toName, commenterName, taskContent, comment, repoName, frontendURL string) error {
	if !s.cfg.SMTPEnabled {
		s.log.Info("SMTP disabled, skipping SendCommentNotification", zap.String("toName", toName))
		return nil
	}
	if toEmail == "" {
		s.log.Info("Recipient email is empty, skipping SendCommentNotification", zap.String("toName", toName))
		return nil
	}

	if frontendURL == "" {
		frontendURL = s.cfg.FrontendURL
	}
	subject := fmt.Sprintf("[CodeTasker] New comment on your task in %s", repoName)
	bodyText := fmt.Sprintf("<strong>%s</strong> commented on a task in <strong>%s</strong>:<br><br>Task: <i>\"%s\"</i><br>Comment: <strong>\"%s\"</strong>", commenterName, repoName, taskContent, comment)
	
	bodyHTML := strings.ReplaceAll(emailHTMLTemplate, "{{RECIPIENT_NAME}}", toName)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{PROJECT_NAME}}", repoName)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{BODY_TEXT}}", bodyText)
	bodyHTML = strings.ReplaceAll(bodyHTML, "{{ACTION_URL}}", frontendURL)

	return s.send(toEmail, subject, bodyHTML)
}

// send is the internal SMTP delivery method. It constructs the MIME headers,
// authenticates with PlainAuth, and delivers the HTML message.
func (s *EmailService) send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	headers := strings.Join([]string{
		"From: " + s.cfg.SMTPFrom,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=utf-8",
	}, "\r\n")
	message := []byte(headers + "\r\n\r\n" + htmlBody)

	// If port is 465, use implicit TLS connection
	if s.cfg.SMTPPort == "465" {
		tlsConfig := &tls.Config{
			ServerName: s.cfg.SMTPHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			s.log.Error("failed to connect via TLS on 465", zap.String("to", to), zap.Error(err))
			return fmt.Errorf("tls dial: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
		if err != nil {
			s.log.Error("failed to create SMTP client on 465", zap.String("to", to), zap.Error(err))
			return fmt.Errorf("smtp client: %w", err)
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			s.log.Error("SMTP authentication failed on 465", zap.String("to", to), zap.Error(err))
			return fmt.Errorf("smtp auth: %w", err)
		}

		if err = client.Mail(s.cfg.SMTPUsername); err != nil {
			return fmt.Errorf("smtp mail from: %w", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt to: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		_, err = w.Write(message)
		if err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		err = w.Close()
		if err != nil {
			return fmt.Errorf("close message: %w", err)
		}

		s.log.Info("email sent successfully via TLS (465)", zap.String("to", to), zap.String("subject", subject))
		return nil
	}

	// For other ports (e.g. 587 STARTTLS), use standard smtp.SendMail
	if err := smtp.SendMail(addr, auth, s.cfg.SMTPUsername, []string{to}, message); err != nil {
		s.log.Error("failed to send email", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("send email to %s: %w", to, err)
	}

	s.log.Info("email sent successfully", zap.String("to", to), zap.String("subject", subject))
	return nil
}

const emailHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CodeTasker Notification</title>
  <style>
    @font-face {
      font-family: 'Camiro';
      src: url('{{ACTION_URL}}/fonts/Camiro.ttf') format('truetype');
      font-weight: bold;
      font-style: normal;
    }
    body {
      background-color: #fafafa;
      color: #24292f;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      margin: 0;
      padding: 0;
      -webkit-font-smoothing: antialiased;
    }
    table {
      border-collapse: collapse;
      width: 100%;
    }
    .wrapper {
      background-color: #fafafa;
      padding: 40px 20px;
    }
    .container {
      background-color: #ffffff;
      border: 1px solid #d0d7de;
      border-radius: 8px;
      max-width: 560px;
      margin: 0 auto;
      overflow: hidden;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.02);
    }
    .header {
      border-bottom: 1px solid #d0d7de;
      padding: 24px 32px;
    }
    .logo-text {
      color: #000000;
      font-family: 'Camiro', Georgia, serif;
      font-size: 16px;
      font-weight: bold;
      letter-spacing: 0.15em;
      text-decoration: none;
    }
    .header-link {
      color: #57606a;
      font-size: 12px;
      text-decoration: none;
      font-weight: 500;
    }
    .content {
      padding: 32px 32px 40px 32px;
    }
    .greeting {
      font-size: 14px;
      color: #57606a;
      margin-bottom: 8px;
    }
    .title {
      font-size: 20px;
      font-weight: 600;
      color: #000000;
      margin-top: 0;
      margin-bottom: 24px;
      line-height: 1.3;
    }
    .body-text {
      font-size: 15px;
      line-height: 1.6;
      color: #24292f;
      margin-bottom: 24px;
    }
    .btn-container {
      margin-bottom: 16px;
    }
    .btn {
      background-color: #000000;
      border: 1px solid #000000;
      border-radius: 6px;
      color: #ffffff !important;
      display: inline-block;
      font-size: 14px;
      font-weight: 600;
      line-height: 20px;
      padding: 10px 24px;
      text-align: center;
      text-decoration: none;
      white-space: nowrap;
    }
    .footer {
      background-color: #f6f8fa;
      border-top: 1px solid #d0d7de;
      padding: 24px 32px;
      text-align: center;
    }
    .footer-text {
      font-size: 12px;
      color: #57606a;
      margin: 0;
    }
    .footer-text a {
      color: #0969da;
      text-decoration: none;
    }
    .footer-text a:hover {
      text-decoration: underline;
    }
  </style>
</head>
<body>
  <div class="wrapper">
    <div class="container">
      <div class="header">
        <table border="0" cellpadding="0" cellspacing="0">
          <tr>
            <td align="left">
              <span class="logo-text">&lt;/ CODETASKER &gt;</span>
            </td>
            <td align="right">
              <a href="https://noirlang.tr" target="_blank" class="header-link">noirlang.tr</a>
            </td>
          </tr>
        </table>
      </div>
      
      <div class="content">
        <div class="greeting">Hi {{RECIPIENT_NAME}},</div>
        <h1 class="title">{{PROJECT_NAME}}</h1>
        
        <div class="body-text">
          {{BODY_TEXT}}
        </div>
        
        <div class="btn-container">
          <a href="{{ACTION_URL}}" target="_blank" class="btn">View Task</a>
        </div>
      </div>
      
      <div class="footer">
        <p class="footer-text">
          © 2026 <a href="https://noirlang.tr" target="_blank">noirLang</a>. Sync your TODOs. Ship faster.
        </p>
      </div>
    </div>
  </div>
</body>
</html>`


