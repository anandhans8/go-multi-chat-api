# Application Layer Documentation

## Overview

The application layer contains the use cases that orchestrate the flow of data to and from the entities. This layer implements the business rules that are specific to the application itself, rather than the domain entities. It depends on the domain layer but is independent of any external frameworks or technologies.

## Use Cases

### Message Use Case

The `MessageUseCase` implements the business logic for sending messages through different providers.

```go
// IMessageUseCase defines the interface for message use cases
type IMessageUseCase interface {
    SendMessage(request *MessageRequest) (*MessageResponse, error)
    RetryFailedMessages() error
}

// MessageUseCase implements the IMessageUseCase interface
type MessageUseCase struct {
    providerRepository         providerRepo.ProviderRepositoryInterface
    userProviderRepository     providerRepo.UserProviderRepositoryInterface
    messageTransactionRepository providerRepo.MessageTransactionRepositoryInterface
    messageProcessor           *messaging.MessageProcessor
    Logger                     *logger.Logger
}
```

#### SendMessage

The `SendMessage` method sends a message using the appropriate provider based on the user's configuration and the requested provider type.

```go
// SendMessage sends a message using the appropriate provider
func (m *MessageUseCase) SendMessage(request *MessageRequest) (*MessageResponse, error) {
    // Get user providers by priority
    userProviders, err := m.userProviderRepository.GetUserProvidersByPriority(request.UserID)
    if err != nil {
        m.Logger.Error("Error getting user providers", zap.Error(err), zap.Int("userID", request.UserID))
        return nil, err
    }

    // Select the appropriate provider based on the request type and user configuration
    // Create a message transaction record
    // Enqueue the message for processing by the message processor
    // Return immediate response to the user
}
```

#### RetryFailedMessages

The `RetryFailedMessages` method checks for failed messages that are ready for retry and attempts to send them using the next provider in the priority list.

```go
// RetryFailedMessages checks for failed messages that are ready for retry
func (m *MessageUseCase) RetryFailedMessages() error {
    // Get failed messages ready for retry
    // Process each failed message
    // Find the next provider to try (after the one that failed)
    // Create a new message transaction for the retry
    // Enqueue the message for processing
}
```

### Auth Use Case

The `AuthUseCase` implements the business logic for user authentication and authorization.

```go
// IAuthUseCase defines the interface for authentication use cases
type IAuthUseCase interface {
    Login(username string, password string) (*LoginResponse, error)
    Register(user *domain.User) (*domain.User, error)
    ValidateToken(token string) (*domain.User, error)
}

// AuthUseCase implements the IAuthUseCase interface
type AuthUseCase struct {
    userRepository user.UserRepositoryInterface
    jwtService     security.IJWTService
    Logger         *logger.Logger
}
```

#### Login

The `Login` method authenticates a user and returns a JWT token.

```go
// Login authenticates a user and returns a JWT token
func (a *AuthUseCase) Login(username string, password string) (*LoginResponse, error) {
    // Find the user by username
    // Verify the password
    // Generate a JWT token
    // Return the token and user information
}
```

#### Register

The `Register` method creates a new user account.

```go
// Register creates a new user account
func (a *AuthUseCase) Register(user *domain.User) (*domain.User, error) {
    // Hash the password
    // Create the user in the repository
    // Return the created user
}
```

#### ValidateToken

The `ValidateToken` method validates a JWT token and returns the associated user.

```go
// ValidateToken validates a JWT token and returns the associated user
func (a *AuthUseCase) ValidateToken(token string) (*domain.User, error) {
    // Validate the token
    // Get the user ID from the token
    // Find the user by ID
    // Return the user
}
```

### User Use Case

The `UserUseCase` implements the business logic for user management.

```go
// IUserUseCase defines the interface for user use cases
type IUserUseCase interface {
    GetAllUsers() (*[]domain.User, error)
    GetUserByID(id int) (*domain.User, error)
    CreateUser(user *domain.User) (*domain.User, error)
    UpdateUser(id int, userMap map[string]interface{}) (*domain.User, error)
    DeleteUser(id int) error
}

// UserUseCase implements the IUserUseCase interface
type UserUseCase struct {
    userRepository user.UserRepositoryInterface
    Logger         *logger.Logger
}
```

### Signal Use Case

The `SignalUseCase` implements the business logic for interacting with the Signal messaging service.

```go
// ISignalUseCase defines the interface for signal use cases
type ISignalUseCase interface {
    // Account operations
    RegisterNumber(number string, useVoice bool, captcha string) error
    VerifyRegisteredNumber(number string, token string, pin string) error
    UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error
    GetAccounts() ([]string, error)
    
    // Messaging operations
    Send(number string, message string, recipients []string, attachments []string, isGroup bool) (*domainSignal.SendResponse, error)
    Receive(number string, timeout int64, ignoreAttachments bool, ignoreStories bool, maxMessages int64, sendReadReceipts bool) (string, error)
    
    // Group operations
    CreateGroup(number string, name string, members []string, description string, editGroupPermission domainSignal.GroupPermission, addMembersPermission domainSignal.GroupPermission, groupLinkState domainSignal.GroupLinkState, expirationTime *int) (string, error)
    GetGroups(number string) ([]domainSignal.GroupEntry, error)
    GetGroup(number string, groupId string) (*domainSignal.GroupEntry, error)
    UpdateGroup(number string, groupId string, avatar *string, description *string, name *string, expirationTime *int, groupLinkState *domainSignal.GroupLinkState) error
    DeleteGroup(number string, groupId string) error
    AddMembersToGroup(number string, groupId string, members []string) error
    RemoveMembersFromGroup(number string, groupId string, members []string) error
    AddAdminsToGroup(number string, groupId string, admins []string) error
    RemoveAdminsFromGroup(number string, groupId string, admins []string) error
    
    // Identity operations
    ListIdentities(number string) (*[]domainSignal.IdentityEntry, error)
    TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error
    
    // QR code operations
    GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error)
}
```

## Data Transfer Objects (DTOs)

The application layer defines DTOs to transfer data between the use cases and the interfaces layer.

### MessageRequest

```go
// MessageRequest represents a request to send a message
type MessageRequest struct {
    Type       string
    Message    string
    Recipients []string
    UserID     int
}
```

### MessageResponse

```go
// MessageResponse represents the response from sending a message
type MessageResponse struct {
    ID      int
    Status  string
    Message string
}
```

### LoginResponse

```go
// LoginResponse represents the response from a login request
type LoginResponse struct {
    Token string
    User  *domain.User
}
```