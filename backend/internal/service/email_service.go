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
	bodyText := fmt.Sprintf("<strong>%s</strong> size <strong>%s</strong> projesinde yeni bir görev atadı:<br><br><strong>\"%s\"</strong>", assignerName, repoName, taskContent)
	
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
	bodyText := fmt.Sprintf("<strong>%s</strong>, <strong>%s</strong> projesindeki bir göreve yorum yazdı:<br><br>Görev: <i>\"%s\"</i><br>Yorum: <strong>\"%s\"</strong>", commenterName, repoName, taskContent, comment)
	
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

const emailHTMLTemplate = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office"><head><meta name="viewport" content="width=device-width, initial-scale=1.0"><link rel="preload" as="image" href="images/0b574f62d42f26ee2e4f9af42ec192da.jpg"><meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><meta name="format-detection" content="telephone=no, date=no, address=no, email=no"><meta name="x-apple-disable-message-reformatting"><meta name="keywords" content="DAHMuAMxgAA, BAEnHEhIKmc"><style>body{margin:0;padding:0}table{mso-table-lspace:0;mso-table-rspace:0}p,span,h1,h2,h3,h4,h5,h6{margin:0;padding:0}p{line-height:inherit}a[x-apple-data-detectors]{color:inherit!important;text-decoration:inherit!important}#MessageViewBody a{color:inherit;text-decoration:none}img+div{display:none}.ecw{width:100%!important;min-width:0!important}</style><!--[if mso]><div>                 <noscript>                   <xml>                     <w:WordDocument xmlns:w="urn:schemas-microsoft-com:office:word">                       <w:DontUseAdvancedTypographyReadingMail/>                     </w:WordDocument>                     <o:OfficeDocumentSettings>                       <o:AllowPNG/>                       <o:PixelsPerInch>96</o:PixelsPerInch>                     </o:OfficeDocumentSettings>                   </xml>                 </noscript></div><![endif]--><!--[if !mso]><!--><style>@media (max-width:200px){ .l0-c0,.l0-c1{display:block!important;width:100%!important} .l0-s0{display:block!important;width:auto!important;height:16px;font-size:0} }</style><!--<![endif]--><style>@media(max-width:550px){.ers-fs-173{font-size:16.7px!important}.ers-fs-213{font-size:18.7px!important}.ers-fs-867{font-size:51.4px!important}}</style></head><body style="width:100%;-webkit-text-size-adjust:100%;text-size-adjust:100%;background-color:#f0f1f5;margin:0;padding:0"><table width="100%" border="0" cellpadding="0" cellspacing="0" bgcolor="#f0f1f5" style="background-color:#f0f1f5"><tbody><tr><td style="background-color:#f0f1f5"><!--[if mso]><center>                     <table align="center" border="0" cellpadding="0" cellspacing="0" width="600">                       <tbody>                         <tr>                           <td><![endif]--><table align="center" width="600" border="0" cellpadding="0" cellspacing="0" role="presentation" class="ecw" style="max-width:600px;min-height:600px;margin:0 auto;background-color:#ffffff;width:600px;min-width:600px"><tbody><tr><td style="vertical-align:top"></td></tr><tr><td style="vertical-align:top"><table border="0" cellpadding="0" cellspacing="0" class="layout-0" align="center" style="display:table;border-spacing:0px;border-collapse:separate;width:100%;max-width:100%;table-layout:fixed;margin:0 auto;background-color:#ffffff"><tbody><tr><td style="text-align:center;padding:15.929782228638457px 24px"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;max-width:552px;table-layout:fixed;margin:0 auto"><tbody><tr><td width="50.78%" class="l0-c0" style="width:50.78%;box-sizing:border-box;vertical-align:middle;background-color:#ffffff"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;table-layout:fixed"><tbody><tr><td style="padding:2px"><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation" style="color:#000;font-style:normal;font-weight:normal;font-size:16px;line-height:1.4;letter-spacing:0;text-align:left;direction:ltr;border-collapse:collapse;font-family:Arial, Helvetica, sans-serif;white-space:normal;word-wrap:break-word;word-break:break-word"><tbody><tr><td dir="ltr" style="color:#280f91;font-size:13.3px;letter-spacing:0.04em;white-space:pre-wrap;text-align:left;line-height:12.8px;mso-line-height-alt:13.3px">&lt;/CODETASKER&gt;<br></td></tr></tbody></table></td></tr></tbody></table></td><td width="2" class="l0-s0" style="width:2px;box-sizing:border-box;font-size:0">&nbsp;</td><td width="48.86%" class="l0-c1" style="width:48.86%;box-sizing:border-box;vertical-align:middle;border-top-left-radius:100px;border-top-right-radius:100px;border-bottom-left-radius:100px;border-bottom-right-radius:100px"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;table-layout:fixed"><tbody><tr><td style="padding:2px"><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation" style="color:#000;font-style:normal;font-weight:normal;font-size:16px;line-height:1.4;letter-spacing:0;text-align:left;direction:ltr;border-collapse:collapse;font-family:Arial, Helvetica, sans-serif;white-space:normal;word-wrap:break-word;word-break:break-word"><tbody><tr><td dir="ltr" style="color:#0e1b10;font-size:16px;font-weight:700;letter-spacing:-0.04em;white-space:pre-wrap;text-align:right;line-height:1;mso-line-height-alt:16px"><a href="http://www.noirLang.tr" target="_blank" rel="noopener noreferrer" style="color:#0e1b10;text-decoration:none">www.noirLang.tr</a><br></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr><tr><td style="vertical-align:top;padding:0px            0px            0px            0px"><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation"><tbody><tr><td style="padding:24px 0 24px 0;vertical-align:top"><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation" style="color:#000;font-style:normal;font-weight:normal;font-size:16px;line-height:1.4;letter-spacing:0;text-align:left;direction:ltr;border-collapse:collapse;font-family:Arial, Helvetica, sans-serif;white-space:normal;word-wrap:break-word;word-break:break-word"><tbody><tr><td dir="ltr" class="ers-fs-867" style="font-size:86.7px;letter-spacing:-0.06em;white-space:pre-wrap;text-align:center;padding:0px 24px 16px;line-height:1;mso-line-height-alt:86.7px">Hello!<br></td></tr><tr><td dir="ltr" class="ers-fs-173" style="font-size:17.3px;letter-spacing:-0.01em;font-family:&quot;Times New Roman&quot;, Times, serif;white-space:pre-wrap;text-align:center;padding:0px 24px 16px;line-height:1;mso-line-height-alt:17.3px">{{RECIPIENT_NAME}}<br></td></tr><tr><td style="padding:0px 24px 16px"><table cellpadding="0" cellspacing="0" border="0" style="width:100%"><tbody><tr><td align="center"><table cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:552px"><tbody><tr><td style="width:100%"><img src="https://qfg0m4_jrx-wglmkqj7_yghnsvxvqmxwheq_i4b5qt4.canva-cdn.email/0b574f62d42f26ee2e4f9af42ec192da.jpg" width="552" height="293" style="display:block;width:100%;height:auto;max-width:100%"></td></tr></tbody></table></td></tr></tbody></table></td></tr><tr><td dir="ltr" style="font-size:13.3px;white-space:pre-wrap;text-align:left;padding:0px 24px 16px;line-height:22.4px;mso-line-height-alt:22.4px;text-decoration:none">&nbsp;</td></tr><tr><td style="padding:0px 0px 16px"><table border="0" cellpadding="0" cellspacing="0" class="layout-1" align="center" style="display:table;border-spacing:0px;border-collapse:separate;width:100%;max-width:100%;table-layout:fixed;margin:0 auto"><tbody><tr><td style="text-align:center;padding:18.20674465371333px 24px"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;max-width:452px;table-layout:fixed;margin:0 auto"><tbody><tr><td width="100.00%" style="width:100.00%;box-sizing:border-box;vertical-align:top"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;table-layout:fixed"><tbody><tr><td><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation" style="color:#000;font-style:normal;font-weight:normal;font-size:16px;line-height:1.4;letter-spacing:0;text-align:left;direction:ltr;border-collapse:collapse;font-family:Arial, Helvetica, sans-serif;white-space:normal;word-wrap:break-word;word-break:break-word"><tbody><tr><td dir="ltr" class="ers-fs-213" style="font-size:21.3px;font-weight:700;letter-spacing:-0.01em;white-space:pre-wrap;text-align:center;padding:0px 0px 16px;line-height:1.4;mso-line-height-alt:29.8px">{{PROJECT_NAME}}<br></td></tr><tr><td dir="ltr" style="font-size:16px;letter-spacing:-0.01em;font-family:&quot;Times New Roman&quot;, Times, serif;white-space:pre-wrap;text-align:center;line-height:1.4;mso-line-height-alt:22.4px">{{BODY_TEXT}}<br></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr><tr><td style="padding:0px 24px 16px"><table cellpadding="0" cellspacing="0" border="0" style="width:100%"><tbody><tr><td align="center"><table cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:552px"><tbody><tr><td height="1" style="height:1px;border-radius:999px;line-height:1px;mso-line-height-rule:exactly;font-size:0;background-color:#bfc3c8">&nbsp;</td></tr></tbody></table></td></tr></tbody></table></td></tr><tr><td dir="ltr" style="font-size:16px;white-space:pre-wrap;text-align:left;padding:0px 24px 16px;line-height:1.4;mso-line-height-alt:22.4px;text-decoration:none">&nbsp;</td></tr><tr><td style="padding:0px 24px"><table cellpadding="0" cellspacing="0" border="0" style="width:100%"><tbody><tr><td align="center"><table cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:552px"><tbody><tr><td style="width:100%"><a href="{{ACTION_URL}}" target="_blank" rel="noopener" ses:no-track="" style="color:#ffffff;text-decoration:none"><!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{{ACTION_URL}}" style="height:50px;width:552px;v-text-anchor:middle;" arcsize="20%" fillcolor="#280f91"><v:stroke dashstyle="Solid" weight="0px" color="#280f91"/><w:anchorlock/><v:textbox inset="0px,0px,0px,0px"><center dir="false" style="color:#ffffff;font-family:sans-serif;font-size:16px"><![endif]--><span style="background-color:#280f91;border-bottom:0px solid transparent;border-left:0px solid transparent;border-radius:10px;border-right:0px solid transparent;border-top:0px solid transparent;color:#ffffff;display:table;font-family:Arial, Helvetica, sans-serif;font-size:16px;font-weight:700;height:50px;mso-border-alt:none;text-align:center;width:100%;max-width:100%;box-sizing:border-box;border-spacing:0;border-collapse:separate;letter-spacing:0em;line-height:22.4px"><span style="word-break:break-word;padding-left:8px;padding-right:8px;display:table-cell;height:100%;vertical-align:middle"><span style="word-break:break-word;line-height:22.4px;mso-style-textfill-type:solid;mso-style-textfill-fill-color:#ffffff">tıkla ve git</span></span></span><!--[if mso]></center></v:textbox></v:roundrect><![endif]--></a></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr><tr><td height="100%" style="height:100%;font-size:0;line-height:0" aria-hidden="true">&nbsp;</td></tr><tr><td style="vertical-align:top"><table border="0" cellpadding="0" cellspacing="0" class="layout-2" align="center" style="display:table;border-spacing:0px;border-collapse:separate;width:100%;max-width:100%;table-layout:fixed;margin:0 auto;background-color:#000000"><tbody><tr><td style="text-align:center;padding:0px 24px"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;max-width:552px;table-layout:fixed;margin:0 auto"><tbody><tr><td width="100.00%" style="width:100.00%;box-sizing:border-box;vertical-align:top;border-top-left-radius:0;border-top-right-radius:0;border-bottom-left-radius:0;border-bottom-right-radius:0"><table border="0" cellpadding="0" cellspacing="0" style="border-spacing:0px;border-collapse:separate;width:100%;table-layout:fixed"><tbody><tr><td style="padding:13px"><table align="center" width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation" style="color:#000;font-style:normal;font-weight:normal;font-size:16px;line-height:1.4;letter-spacing:0;text-align:left;direction:ltr;border-collapse:collapse;font-family:Arial, Helvetica, sans-serif;white-space:normal;word-wrap:break-word;word-break:break-word"><tbody><tr><td dir="ltr" style="color:#ffffff;font-size:13.3px;letter-spacing:-0.01em;white-space:pre-wrap;text-align:left;padding:0px 0px 16px;line-height:22.4px;mso-line-height-alt:22.4px;text-decoration:none">&nbsp;</td></tr><tr><td dir="ltr" style="color:#ffffff;font-size:13.3px;letter-spacing:-0.01em;white-space:pre-wrap;text-align:center;line-height:8px;mso-line-height-alt:13.3px">© 2026 noirLang. All rights reserved.<br></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table></td></tr></tbody></table><!--[if mso]></td>                 </tr>               </tbody>             </table>           </center><![endif]--></td></tr></tbody></table></body></html>`

