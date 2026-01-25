package email_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/email"
)

func TestNewService_Disabled_WhenNoConfig(t *testing.T) {
	// Empty config should result in disabled service
	service := email.NewService(email.Config{})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Disabled_WhenMissingHost(t *testing.T) {
	service := email.NewService(email.Config{
		Username: "user",
		Password: "pass",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Disabled_WhenMissingUsername(t *testing.T) {
	service := email.NewService(email.Config{
		Host:     "smtp.example.com",
		Password: "pass",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Disabled_WhenMissingPassword(t *testing.T) {
	service := email.NewService(email.Config{
		Host:     "smtp.example.com",
		Username: "user",
	})
	assert.False(t, service.IsEnabled())
}

func TestNewService_Enabled_WhenFullConfig(t *testing.T) {
	service := email.NewService(email.Config{
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
	service := email.NewService(email.Config{})
	assert.False(t, service.IsEnabled())

	err := service.SendPasswordResetEmail("test@example.com", "test-token", "https://example.com")
	assert.NoError(t, err)
}
