package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
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

// SendTextAndHTMLEmailMulti sends a multipart email (text + html) with an optional inline PNG image.
func (s *Service) SendTextAndHTMLEmailMulti(
	to []string,
	subject string,
	textBody string,
	htmlBody string,
	inlineImageCID string,
	inlineImagePNG []byte,
) error {
	if !s.enabled {
		log.Info().Strs("to", to).Str("subject", subject).Msg("Email sending disabled, skipping email")
		return nil
	}

	msg, err := buildMultipartMessage(s.config.From, to, subject, textBody, htmlBody, inlineImageCID, inlineImagePNG)
	if err != nil {
		return err
	}

	return s.sendRawMulti(to, subject, msg)
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
	toHeader := strings.Join(to, ", ")

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", s.config.From, toHeader, subject, body)

	return s.sendRawMulti(to, subject, []byte(msg))
}

func (s *Service) sendRawMulti(to []string, subject string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	if s.config.UseTLS {
		return s.sendTLS(addr, auth, to, subject, msg)
	}

	err := smtp.SendMail(addr, auth, s.config.From, to, msg)
	if err != nil {
		log.Error().Err(err).Strs("to", to).Msg("Failed to send email")
		return err
	}

	log.Info().Strs("to", to).Str("subject", subject).Msg("Email sent successfully")
	return nil
}

// sendTLS sends an email using TLS to one or more recipients.
func (s *Service) sendTLS(addr string, auth smtp.Auth, to []string, subject string, msg []byte) error {
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

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close email body: %w", err)
	}

	log.Info().Strs("to", to).Str("subject", subject).Msg("Email sent successfully (TLS)")
	return nil
}

func buildMultipartMessage(
	from string,
	to []string,
	subject string,
	textBody string,
	htmlBody string,
	inlineImageCID string,
	inlineImagePNG []byte,
) ([]byte, error) {
	var relatedBody bytes.Buffer
	relatedWriter := multipart.NewWriter(&relatedBody)

	var altBody bytes.Buffer
	altWriter := multipart.NewWriter(&altBody)

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textPart, err := altWriter.CreatePart(textHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create text part: %w", err)
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return nil, fmt.Errorf("failed to write text part: %w", err)
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlPart, err := altWriter.CreatePart(htmlHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create html part: %w", err)
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, fmt.Errorf("failed to write html part: %w", err)
	}

	if err := altWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize alternative multipart: %w", err)
	}

	altContainerHeader := textproto.MIMEHeader{}
	altContainerHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", altWriter.Boundary()))
	altContainerPart, err := relatedWriter.CreatePart(altContainerHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create alternative container part: %w", err)
	}
	if _, err := altContainerPart.Write(altBody.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write alternative container part: %w", err)
	}

	if len(inlineImagePNG) > 0 && strings.TrimSpace(inlineImageCID) != "" {
		imageHeader := textproto.MIMEHeader{}
		imageHeader.Set("Content-Type", "image/png")
		imageHeader.Set("Content-Transfer-Encoding", "base64")
		imageHeader.Set("Content-ID", fmt.Sprintf("<%s>", strings.TrimSpace(inlineImageCID)))
		imageHeader.Set("Content-Disposition", "inline; filename=\"payment-qr.png\"")

		imagePart, err := relatedWriter.CreatePart(imageHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to create image part: %w", err)
		}
		if err := writeBase64Lines(imagePart, inlineImagePNG); err != nil {
			return nil, fmt.Errorf("failed to write image part: %w", err)
		}
	}

	if err := relatedWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize related multipart: %w", err)
	}

	toHeader := strings.Join(to, ", ")
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", toHeader)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/related; boundary=%q\r\n", relatedWriter.Boundary())
	fmt.Fprintf(&msg, "\r\n")
	if _, err := msg.Write(relatedBody.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write multipart body: %w", err)
	}

	return msg.Bytes(), nil
}

func writeBase64Lines(w io.Writer, content []byte) error {
	encoded := base64.StdEncoding.EncodeToString(content)
	for len(encoded) > 76 {
		if _, err := io.WriteString(w, encoded[:76]); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if _, err := io.WriteString(w, encoded); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}
