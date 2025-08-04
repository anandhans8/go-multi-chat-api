package provider

import (
	"go-multi-chat-api/src/infrastructure/alerting/alert"
	"go-multi-chat-api/src/infrastructure/alerting/provider/email"
)

// AlertProvider is the interface that each provider should implement
type AlertProvider interface {
	// Validate the provider's configuration
	Validate() error

	// Send an alert using the provider
	Send(alert *alert.Alert) error

	// GetDefaultAlert returns the provider's default alert configuration
	GetDefaultAlert() *alert.Alert

	// ValidateOverrides validates the alert's provider override and, if present, the group override
	ValidateOverrides(alert *alert.Alert) error
}

type Config[T any] interface {
	Validate() error
	Merge(override *T)
}

// MergeProviderDefaultAlertIntoEndpointAlert parses an Endpoint alert by using the provider's default alert as a baseline
func MergeProviderDefaultAlertIntoEndpointAlert(providerDefaultAlert, endpointAlert *alert.Alert) {
	if providerDefaultAlert == nil || endpointAlert == nil {
		return
	}
	if endpointAlert.Enabled == nil {
		endpointAlert.Enabled = providerDefaultAlert.Enabled
	}
	if endpointAlert.Description == nil {
		endpointAlert.Description = providerDefaultAlert.Description
	}
}

var (
	// Validate provider interface implementation on compile
	//_ AlertProvider = (*custom.AlertProvider)(nil)
	_ AlertProvider = (*email.AlertProvider)(nil)

	_ Config[email.Config] = (*email.Config)(nil)
)
