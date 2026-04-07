package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog/log"
)

// Config holds SMTP configuration.
type Config struct {
	Host     string
	Port     int
	From     string
	Username string
	Password string
	UseTLS   bool
}

// Service handles email sending.
type Service struct {
	config  Config
	enabled bool
}

// NewService creates a new email service.
func NewService(cfg Config) *Service {
	enabled := cfg.Host != "" && cfg.Username != "" && cfg.Password != ""
	if !enabled {
		log.Warn().Msg("Email service disabled: SMTP not configured")
	} else {
		log.Info().Str("host", cfg.Host).Int("port", cfg.Port).Msg("Email service initialized")
	}
	return &Service{
		config:  cfg,
		enabled: enabled,
	}
}

// IsEnabled returns whether the email service is configured.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// SendTextEmail sends a plain text email to a single recipient.
func (s *Service) SendTextEmail(to, subject, body string) error {
	return s.SendTextEmailMulti([]string{to}, subject, body)
}

// SendTextEmailMulti sends a plain text email to multiple recipients.
func (s *Service) SendTextEmailMulti(to []string, subject, body string) error {
	if !s.enabled {
		log.Info().Strs("to", to).Str("subject", subject).Msg("Email sending disabled, skipping email")
		return nil
	}

	return s.sendMulti(to, subject, body)
}

// SendPasswordResetEmail sends a password reset email.
func (s *Service) SendPasswordResetEmail(to, token, baseURL string) error {
	if !s.enabled {
		log.Info().Str("to", to).Str("token", token).Msg("Email sending disabled, logging password reset token")
		return nil
	}

	subject, body := BuildPasswordResetEmail(token, baseURL)

	return s.sendMulti([]string{to}, subject, body)
}

// BuildPasswordResetEmail builds the subject and body for password reset emails.
func BuildPasswordResetEmail(token, baseURL string) (string, string) {
	resetLink := fmt.Sprintf("%s/passwort-zuruecksetzen?token=%s", strings.TrimSuffix(baseURL, "/"), token)

	subject := "Passwort zurücksetzen - Knirpsenstadt Beiträge"
	body := fmt.Sprintf(`Hallo,

Sie haben angefordert, Ihr Passwort für die Knirpsenstadt Beitrags-App zurückzusetzen.

Klicken Sie auf den folgenden Link, um Ihr Passwort zurückzusetzen:
%s

Dieser Link ist 1 Stunde gültig.

Falls Sie diese Anfrage nicht gestellt haben, können Sie diese E-Mail ignorieren.

Mit freundlichen Grüßen
Ihr Knirpsenstadt-Team`, resetLink)

	return subject, body
}

// sendMulti sends an email via SMTP to one or more recipients.
func (s *Service) sendMulti(to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	toHeader := strings.Join(to, ", ")

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", s.config.From, toHeader, subject, body)

	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	if s.config.UseTLS {
		return s.sendTLS(addr, auth, to, msg)
	}

	err := smtp.SendMail(addr, auth, s.config.From, to, []byte(msg))
	if err != nil {
		log.Error().Err(err).Strs("to", to).Msg("Failed to send email")
		return err
	}

	log.Info().Strs("to", to).Str("subject", subject).Msg("Email sent successfully")
	return nil
}

// sendTLS sends an email using TLS to one or more recipients.
func (s *Service) sendTLS(addr string, auth smtp.Auth, to []string, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: s.config.Host,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT command failed for %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close email body: %w", err)
	}

	log.Info().Strs("to", to).Msg("Email sent successfully (TLS)")
	return nil
}
