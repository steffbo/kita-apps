package email

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService_Disabled_WhenNoConfig(t *testing.T) {
	// Empty config should result in disabled service
	service := NewService(Config{})
	assert.False(t, service.IsEnabled())
}

func TestBuildMultipartMessage_EncodesNonASCIISubject(t *testing.T) {
	message, err := buildMultipartMessage(
		"noreply@example.com",
		[]string{"parent@example.com"},
		"Kita Mahnung: offene Beiträge",
		"Text",
		"<p>Text</p>",
		"",
		nil,
	)
	assert.NoError(t, err)
	assert.Contains(t, string(message), "Subject: =?UTF-8?")

	decoded, err := new(mime.WordDecoder).DecodeHeader(headerValue(message, "Subject"))
	assert.NoError(t, err)
	assert.Equal(t, "Kita Mahnung: offene Beiträge", decoded)
}

func headerValue(message []byte, name string) string {
	prefix := []byte(name + ": ")
	for _, line := range bytes.Split(message, []byte("\r\n")) {
		if bytes.HasPrefix(line, prefix) {
			return string(bytes.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func TestNewService_Disabled_WhenMissingHost(t *testing.T) {
	service := NewService(Config{
		Username: "user",
		Password: "pass",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Disabled_WhenMissingUsername(t *testing.T) {
	service := NewService(Config{
		Host:     "smtp.example.com",
		Password: "pass",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Disabled_WhenMissingPassword(t *testing.T) {
	service := NewService(Config{
		Host:     "smtp.example.com",
		Username: "user",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Enabled_WhenFullConfig(t *testing.T) {
	service := NewService(Config{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "noreply@example.com",
		Username: "user",
		Password: "pass",
	})
	assert.True(t, service.IsEnabled())
}

func TestSendPasswordResetEmail_WhenDisabled_NoError(t *testing.T) {
	// When service is disabled, SendPasswordResetEmail should not error
	// (it just logs the token instead)
	service := NewService(Config{})
	assert.False(t, service.IsEnabled())

	err := service.SendPasswordResetEmail("test@example.com", "test-token", "https://example.com")
	assert.NoError(t, err)
}

func TestBuildTextEmailWithAttachments(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	body := "Grüße aus der Kita – " + strings.TrimSpace(
		strings.Repeat("lange Zeile mit Umlauten äöü ", 5),
	)
	attachments := []Attachment{
		{
			Filename:    "Einstufung_Görs_Simeon.pdf",
			ContentType: "application/pdf",
			Data:        allBytes,
		},
		{
			Filename: "weitere.bin",
			Data:     []byte{0, 1, 2, 255},
		},
	}

	message, err := buildTextEmailWithAttachments(
		"noreply@example.com",
		[]string{"first@example.com", "second@example.com"},
		"Einstufung für Görs",
		body,
		attachments,
	)
	assert.NoError(t, err)

	parsedMessage, err := mail.ReadMessage(bytes.NewReader(message))
	assert.NoError(t, err)
	assert.Equal(t, "noreply@example.com", parsedMessage.Header.Get("From"))
	assert.Equal(t, "first@example.com, second@example.com", parsedMessage.Header.Get("To"))
	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(parsedMessage.Header.Get("Subject"))
	assert.NoError(t, err)
	assert.Equal(t, "Einstufung für Görs", decodedSubject)
	assert.Equal(t, "1.0", parsedMessage.Header.Get("MIME-Version"))

	mediaType, mediaParams, err := mime.ParseMediaType(parsedMessage.Header.Get("Content-Type"))
	assert.NoError(t, err)
	assert.Equal(t, "multipart/mixed", mediaType)

	reader := multipart.NewReader(parsedMessage.Body, mediaParams["boundary"])
	textPart, err := reader.NextPart()
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", mediaTypeOf(t, textPart.Header.Get("Content-Type")))
	textData, err := io.ReadAll(quotedprintable.NewReader(textPart))
	assert.NoError(t, err)
	assert.Equal(t, body, string(textData))

	firstPart, err := reader.NextPart()
	assert.NoError(t, err)
	firstType, firstParams, err := mime.ParseMediaType(firstPart.Header.Get("Content-Type"))
	assert.NoError(t, err)
	assert.Equal(t, "application/pdf", firstType)
	assert.Equal(t, "Einstufung_Görs_Simeon.pdf", firstParams["name"])
	firstDisposition, firstDispositionParams, err := mime.ParseMediaType(
		firstPart.Header.Get("Content-Disposition"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "attachment", firstDisposition)
	assert.Equal(t, "Einstufung_Görs_Simeon.pdf", firstDispositionParams["filename"])
	firstEncoded, err := io.ReadAll(firstPart)
	assert.NoError(t, err)
	firstDecoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(firstEncoded)))
	assert.NoError(t, err)
	assert.Equal(t, allBytes, firstDecoded)
	assertBase64LinesAtMost76(t, firstEncoded)

	secondPart, err := reader.NextPart()
	assert.NoError(t, err)
	secondType, secondParams, err := mime.ParseMediaType(secondPart.Header.Get("Content-Type"))
	assert.NoError(t, err)
	assert.Equal(t, "application/octet-stream", secondType)
	assert.Equal(t, "weitere.bin", secondParams["name"])
	secondEncoded, err := io.ReadAll(secondPart)
	assert.NoError(t, err)
	secondDecoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(secondEncoded)))
	assert.NoError(t, err)
	assert.Equal(t, attachments[1].Data, secondDecoded)

	_, err = reader.NextPart()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSendTextEmailWithAttachments_WhenDisabled_NoError(t *testing.T) {
	service := NewService(Config{})

	err := service.SendTextEmailWithAttachments(nil, "Subject", "Body", []Attachment{{
		Filename: "file.pdf",
		Data:     []byte("data"),
	}})

	assert.NoError(t, err)
}

func mediaTypeOf(t *testing.T, value string) string {
	t.Helper()
	mediaType, _, err := mime.ParseMediaType(value)
	assert.NoError(t, err)
	return mediaType
}

func assertBase64LinesAtMost76(t *testing.T, encoded []byte) {
	t.Helper()
	for _, line := range bytes.Split(bytes.TrimSpace(encoded), []byte("\r\n")) {
		assert.LessOrEqual(t, len(line), 76)
	}
}
