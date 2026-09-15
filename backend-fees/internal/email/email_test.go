package email

import (
	"bytes"
	"mime"
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
