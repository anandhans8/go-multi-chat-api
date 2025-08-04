# Domain Layer Documentation

## Overview

The domain layer is the core of the application, containing the business entities and interfaces that define the core business rules. This layer is independent of any external frameworks or technologies and represents the heart of the business logic.

## Entities

### User

The `User` entity represents a user in the system.

```go
// User represents a user in the system
type User struct {
    ID        int
    Username  string
    Email     string
    Password  string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Provider

The `Provider` entity represents a message provider that can be used to send messages.

```go
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
```

### UserProvider

The `UserProvider` entity represents the relationship between a user and a provider.

```go
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
```

### MessageTransaction

The `MessageTransaction` entity represents a message transaction.

```go
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
    RetryCount   int       // Number of retry attempts
    NextRetryAt  time.Time // When to retry next
    Processing   bool      // Whether the message is currently being processed
    ProcessedAt  time.Time // When the message was last processed
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### MessageTransactionHistory

The `MessageTransactionHistory` entity represents the history of a message transaction.

```go
// MessageTransactionHistory represents the history of a message transaction
type MessageTransactionHistory struct {
    ID              int
    MessageID       int       // Reference to the original message transaction
    UserID          int
    ProviderID      int
    Recipients      string    // JSON array of recipients
    Message         string
    RequestData     string    // JSON request data
    ResponseData    string    // JSON response data
    Status          string    // success, failed
    ErrorMessage    string
    RetryCount      int       // Number of retry attempts
    ProcessedAt     time.Time // When the message was processed
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

## Interfaces

### IProviderService

The `IProviderService` interface defines the operations for provider services.

```go
// IProviderService defines the interface for provider service operations
type IProviderService interface {
    GetAllProviders() (*[]Provider, error)
    GetProviderByID(id int) (*Provider, error)
    CreateProvider(provider *Provider) (*Provider, error)
    UpdateProvider(id int, providerMap map[string]interface{}) (*Provider, error)
    DeleteProvider(id int) error
}
```

### IUserProviderService

The `IUserProviderService` interface defines the operations for user provider services.

```go
// IUserProviderService defines the interface for user provider service operations
type IUserProviderService interface {
    GetUserProviders(userID int) (*[]UserProvider, error)
    GetUserProviderByID(id int) (*UserProvider, error)
    CreateUserProvider(userProvider *UserProvider) (*UserProvider, error)
    UpdateUserProvider(id int, userProviderMap map[string]interface{}) (*UserProvider, error)
    DeleteUserProvider(id int) error
    GetUserProvidersByPriority(userID int) (*[]UserProvider, error)
}
```

### IMessageTransactionService

The `IMessageTransactionService` interface defines the operations for message transaction services.

```go
// IMessageTransactionService defines the interface for message transaction service operations
type IMessageTransactionService interface {
    CreateMessageTransaction(messageTransaction *MessageTransaction) (*MessageTransaction, error)
    GetMessageTransactionByID(id int) (*MessageTransaction, error)
    GetUserMessageTransactions(userID int) (*[]MessageTransaction, error)
    UpdateMessageTransaction(id int, messageTransactionMap map[string]interface{}) (*MessageTransaction, error)
}
```

### IMessageTransactionHistoryService

The `IMessageTransactionHistoryService` interface defines the operations for message transaction history services.

```go
// IMessageTransactionHistoryService defines the interface for message transaction history service operations
type IMessageTransactionHistoryService interface {
    CreateMessageTransactionHistory(history *MessageTransactionHistory) (*MessageTransactionHistory, error)
    GetMessageTransactionHistoryByID(id int) (*MessageTransactionHistory, error)
    GetMessageTransactionHistoryByMessageID(messageID int) (*[]MessageTransactionHistory, error)
    GetUserMessageTransactionHistory(userID int) (*[]MessageTransactionHistory, error)
}
```

## Common

The domain layer also includes common interfaces and utilities that are used across the application.

### CommonService

The `CommonService` interface provides common functionality used across the application.

```go
// CommonService defines common functionality used across the application
type CommonService interface {
    AppendValidationErrors(ctx *gin.Context, validationErrors validator.ValidationErrors, request interface{})
}
```