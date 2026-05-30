package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// SMTPSender delivers real emails via SMTP (Mailhog, SendGrid SMTP relay, etc.).
type SMTPSender struct {
	cfg config.SMTP
}

func NewSMTPSender(cfg config.SMTP) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Name() string { return "smtp" }

func (s *SMTPSender) Send(_ context.Context, msg plugin.EmailMessage) error {
	if s.cfg.Host == "" || s.cfg.From == "" {
		return fmt.Errorf("smtp host and from address are required")
	}
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	body := msg.HTMLBody
	if body == "" {
		body = msg.TextBody
	}
	raw := strings.Join([]string{
		fmt.Sprintf("From: %s", s.cfg.From),
		fmt.Sprintf("To: %s", msg.To),
		fmt.Sprintf("Subject: %s", msg.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	// Port 465 uses implicit TLS; 587 uses STARTTLS.
	if s.cfg.Port == 465 {
		return sendTLS(addr, auth, s.cfg.From, []string{msg.To}, []byte(raw))
	}
	return smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(raw))
}

func sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
		 return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
