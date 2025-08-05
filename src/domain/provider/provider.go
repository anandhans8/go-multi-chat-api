package provider

import (
	"time"
)

// Provider represents a message provider
type Provider struct {
	ID          int
	Name        string
	Type        string // email, signal, etc.
	Description string
	Config      string // JSON configuration for the provider
	Status      bool   // Whether the provider is active
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserProvider represents the relationship between a user and a provider
type UserProvider struct {
	ID         int
	UserID     int
	ProviderID int
	Priority   int    // Lower number means higher priority
	Config     string // JSON configuration specific to this user-provider relationship
	Status     bool   // Whether this provider is active for this user
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MessageTransaction represents a message transaction
type MessageTransaction struct {
	ID           int
	UserID       int
	ProviderID   int
	Recipients   string // JSON array of recipients
	Message      string
	RequestData  string // JSON request data
	ResponseData string // JSON response data
	Status       string // success, failed, pending
	ErrorMessage string
	RetryCount   int        // Number of retry attempts
	NextRetryAt  *time.Time // When to retry next
	Processing   bool       // Whether the message is currently being processed
	ProcessedAt  *time.Time // When the message was last processed
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MessageTransactionHistory represents the history of a message transaction
type MessageTransactionHistory struct {
	ID           int
	MessageID    int // Reference to the original message transaction
	UserID       int
	ProviderID   int
	Recipients   string // JSON array of recipients
	Message      string
	RequestData  string // JSON request data
	ResponseData string // JSON response data
	Status       string // success, failed
	ErrorMessage string
	RetryCount   int       // Number of retry attempts
	ProcessedAt  time.Time // When the message was processed
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IProviderService defines the interface for provider service operations
type IProviderService interface {
	GetAllProviders() (*[]Provider, error)
	GetProviderByID(id int) (*Provider, error)
	CreateProvider(provider *Provider) (*Provider, error)
	UpdateProvider(id int, providerMap map[string]interface{}) (*Provider, error)
	DeleteProvider(id int) error
}

// IUserProviderService defines the interface for user provider service operations
type IUserProviderService interface {
	GetUserProviders(userID int) (*[]UserProvider, error)
	GetUserProviderByID(id int) (*UserProvider, error)
	CreateUserProvider(userProvider *UserProvider) (*UserProvider, error)
	UpdateUserProvider(id int, userProviderMap map[string]interface{}) (*UserProvider, error)
	DeleteUserProvider(id int) error
	GetUserProvidersByPriority(userID int) (*[]UserProvider, error)
}

// IMessageTransactionService defines the interface for message transaction service operations
type IMessageTransactionService interface {
	CreateMessageTransaction(messageTransaction *MessageTransaction) (*MessageTransaction, error)
	GetMessageTransactionByID(id int) (*MessageTransaction, error)
	GetUserMessageTransactions(userID int) (*[]MessageTransaction, error)
	UpdateMessageTransaction(id int, messageTransactionMap map[string]interface{}) (*MessageTransaction, error)
}

// IMessageTransactionHistoryService defines the interface for message transaction history service operations
type IMessageTransactionHistoryService interface {
	CreateMessageTransactionHistory(history *MessageTransactionHistory) (*MessageTransactionHistory, error)
	GetMessageTransactionHistoryByID(id int) (*MessageTransactionHistory, error)
	GetMessageTransactionHistoryByMessageID(messageID int) (*[]MessageTransactionHistory, error)
	GetUserMessageTransactionHistory(userID int) (*[]MessageTransactionHistory, error)
}
