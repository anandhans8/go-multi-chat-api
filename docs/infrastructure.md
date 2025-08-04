# Infrastructure Layer Documentation

## Overview

The infrastructure layer contains the implementation details such as databases, external services, and frameworks. This layer is responsible for implementing the interfaces defined in the domain layer and providing the necessary infrastructure for the application to run.

## Components

### Repository Implementations

The repository implementations provide the concrete implementations of the repository interfaces defined in the domain layer.

#### MySQL Repository

The MySQL repository implementations use GORM to interact with the MySQL database.

```go
// UserRepository implements the UserRepositoryInterface
type UserRepository struct {
    DB     *gorm.DB
    Logger *logger.Logger
}

// ProviderRepository implements the ProviderRepositoryInterface
type ProviderRepository struct {
    DB     *gorm.DB
    Logger *logger.Logger
}

// UserProviderRepository implements the UserProviderRepositoryInterface
type UserProviderRepository struct {
    DB     *gorm.DB
    Logger *logger.Logger
}

// MessageTransactionRepository implements the MessageTransactionRepositoryInterface
type MessageTransactionRepository struct {
    DB     *gorm.DB
    Logger *logger.Logger
}

// MessageTransactionHistoryRepository implements the MessageTransactionHistoryRepositoryInterface
type MessageTransactionHistoryRepository struct {
    DB     *gorm.DB
    Logger *logger.Logger
}
```

### REST Controllers

The REST controllers handle HTTP requests and responses. They use the Gin framework to define routes and handle requests.

#### Auth Controller

The `AuthController` handles authentication-related requests.

```go
// IAuthController defines the interface for authentication controllers
type IAuthController interface {
    Login(c *gin.Context)
    Register(c *gin.Context)
}

// AuthController implements the IAuthController interface
type AuthController struct {
    authUseCase auth.IAuthUseCase
    Logger      *logger.Logger
}
```

#### User Controller

The `UserController` handles user-related requests.

```go
// IUserController defines the interface for user controllers
type IUserController interface {
    GetAllUsers(c *gin.Context)
    GetUserByID(c *gin.Context)
    CreateUser(c *gin.Context)
    UpdateUser(c *gin.Context)
    DeleteUser(c *gin.Context)
}

// UserController implements the IUserController interface
type UserController struct {
    userUseCase user.IUserUseCase
    Logger      *logger.Logger
}
```

#### Send Controller

The `SendController` handles message sending requests.

```go
// ISendController defines the interface for send controllers
type ISendController interface {
    Message(c *gin.Context)
    RetryFailedMessages()
}

// SendController implements the ISendController interface
type SendController struct {
    commonService common.CommonService
    messageUseCase message.IMessageUseCase
    Logger        *logger.Logger
}
```

#### Signal Controller

The `SignalController` handles Signal-related requests.

```go
// ISignalController defines the interface for signal controllers
type ISignalController interface {
    RegisterNumber(c *gin.Context)
    VerifyRegisteredNumber(c *gin.Context)
    UnregisterNumber(c *gin.Context)
    GetAccounts(c *gin.Context)
    Send(c *gin.Context)
    Receive(c *gin.Context)
    CreateGroup(c *gin.Context)
    GetGroups(c *gin.Context)
    GetGroup(c *gin.Context)
    UpdateGroup(c *gin.Context)
    DeleteGroup(c *gin.Context)
    AddMembersToGroup(c *gin.Context)
    RemoveMembersFromGroup(c *gin.Context)
    AddAdminsToGroup(c *gin.Context)
    RemoveAdminsFromGroup(c *gin.Context)
    ListIdentities(c *gin.Context)
    TrustIdentity(c *gin.Context)
    GetQrCodeLink(c *gin.Context)
}
```

### Messaging

The messaging component handles the processing of messages through different providers.

#### Message Processor

The `MessageProcessor` processes messages asynchronously using a worker pool.

```go
// MessageProcessor processes messages asynchronously
type MessageProcessor struct {
    signalService                    signalClient.ISignalClient
    providerRepository               providerRepo.ProviderRepositoryInterface
    userProviderRepository           providerRepo.UserProviderRepositoryInterface
    messageTransactionRepository     providerRepo.MessageTransactionRepositoryInterface
    messageTransactionHistoryRepository providerRepo.MessageTransactionHistoryRepositoryInterface
    Logger                           *logger.Logger
    messageQueue                     chan *provider.MessageTransaction
    workerCount                      int
    workerWaitGroup                  sync.WaitGroup
}
```

### Security

The security component handles authentication and authorization.

#### JWT Service

The `JWTService` handles JWT token generation and validation.

```go
// IJWTService defines the interface for JWT services
type IJWTService interface {
    GenerateToken(userID int, username string) (string, error)
    ValidateToken(token string) (*jwt.Token, error)
    GetUserIDFromToken(token string) (int, error)
}

// JWTService implements the IJWTService interface
type JWTService struct {
    secretKey string
    issuer    string
}
```

### Alerting

The alerting component handles sending alerts through different channels.

#### Alert Service

The `AlertService` sends alerts through different channels.

```go
// IAlertService defines the interface for alert services
type IAlertService interface {
    SendAlert(alert *Alert) error
}

// AlertService implements the IAlertService interface
type AlertService struct {
    providers map[AlertType]provider.IAlertProvider
    Logger    *logger.Logger
}
```

### Dependency Injection

The dependency injection component handles the creation and wiring of dependencies.

#### Application Context

The `ApplicationContext` holds all application dependencies and services.

```go
// ApplicationContext holds all application dependencies and services
type ApplicationContext struct {
    DB                              *gorm.DB
    Logger                          *logger.Logger
    AuthController                  authController.IAuthController
    UserController                  userController.IUserController
    SignalController                signalController.ISignalController
    SendController                  sendController.ISendController
    JWTService                      security.IJWTService
    CommonService                   common.CommonService
    UserRepository                  user.UserRepositoryInterface
    AuthUseCase                     authUseCase.IAuthUseCase
    UserUseCase                     userUseCase.IUserUseCase
    MessageProcessor                *messaging.MessageProcessor
    ProviderRepository              providerRepo.ProviderRepositoryInterface
    UserProviderRepository          providerRepo.UserProviderRepositoryInterface
    MessageTransactionRepository    providerRepo.MessageTransactionRepositoryInterface
    MessageTransactionHistoryRepository providerRepo.MessageTransactionHistoryRepositoryInterface
}
```

### Utilities

The utilities component provides utility functions and helpers.

#### Config

The `Config` provides configuration management.

```go
// Config provides configuration management
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Signal   SignalConfig
}
```

#### Logger

The `Logger` provides logging functionality.

```go
// Logger provides logging functionality
type Logger struct {
    logger *zap.Logger
}
```

## Middleware

The middleware component provides HTTP middleware for the Gin framework.

### RequiresLogin

The `RequiresLogin` middleware ensures that the user is authenticated.

```go
// RequiresLogin ensures that the user is authenticated
func RequiresLogin(jwtService security.IJWTService, logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get the Authorization header
        // Validate the token
        // Set the user ID in the context
        // Continue to the next handler
    }
}
```

### ErrorHandler

The `ErrorHandler` middleware handles errors and returns appropriate HTTP responses.

```go
// ErrorHandler handles errors and returns appropriate HTTP responses
func ErrorHandler(logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Process the request
        // Handle any errors
        // Return appropriate HTTP responses
    }
}
```

### Headers

The `Headers` middleware sets HTTP headers for security and other purposes.

```go
// Headers sets HTTP headers for security and other purposes
func Headers() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Set security headers
        // Set CORS headers
        // Continue to the next handler
    }
}
```