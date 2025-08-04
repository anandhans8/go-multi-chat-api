package alert

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

var (
	// ErrAlertWithInvalidDescription is the error with which Gatus will panic if an alert has an invalid character
	ErrAlertWithInvalidDescription = errors.New("alert description must not have \" or \\")
)

// Alert is endpoint.Endpoint's alert configuration
type Alert struct {
	Type Type `yaml:"type"`

	Enabled *bool `yaml:"enabled,omitempty"`

	Description *string `yaml:"description,omitempty"`

	Subject *string `yaml:"subject,omitempty"`

	Recipients []string `yaml:"recipients,omitempty"`

	// ProviderOverride is an optional field that can be used to override the provider's configuration
	// It is freeform so that it can be used for any provider-specific configuration.
	ProviderOverride map[string]any `yaml:"provider-override,omitempty"`
}

// ValidateAndSetDefaults validates the alert's configuration and sets the default value of fields that have one
func (alert *Alert) ValidateAndSetDefaults() error {
	if strings.ContainsAny(alert.GetDescription(), "\"\\") {
		return ErrAlertWithInvalidDescription
	}
	return nil
}

// GetDescription retrieves the description of the alert
func (alert *Alert) GetDescription() string {
	if alert.Description == nil {
		return ""
	}
	return *alert.Description
}

// IsEnabled returns whether an alert is enabled or not
// Returns true if not set
func (alert *Alert) IsEnabled() bool {
	if alert.Enabled == nil {
		return true
	}
	return *alert.Enabled
}

// Checksum returns a checksum of the alert
// Used to determine which persisted triggered alert should be deleted on application start
func (alert *Alert) Checksum() string {
	hash := sha256.New()
	hash.Write([]byte(string(alert.Type) + "_" +
		strconv.FormatBool(alert.IsEnabled()) + "_" +
		alert.GetDescription()),
	)
	return hex.EncodeToString(hash.Sum(nil))
}

func (alert *Alert) ProviderOverrideAsBytes() []byte {
	yamlBytes, err := yaml.Marshal(alert.ProviderOverride)
	if err != nil {
		//logr.Warnf("[alert.ProviderOverrideAsBytes] Failed to marshal alert override of type=%s as bytes: %v", alert.Type, err)
	}
	return yamlBytes
}
