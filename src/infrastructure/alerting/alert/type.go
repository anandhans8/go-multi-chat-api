package alert

// Type is the type of the alert.
// The value will generally be the name of the alert provider
type Type string

const (

	// TypeEmail is the Type for the email alerting provider
	TypeEmail Type = "email"

	// TypeSignal is the Type for the signal alerting provider
	TypeSignal Type = "signal"
)
