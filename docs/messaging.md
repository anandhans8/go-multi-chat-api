# Messaging System Documentation

## Overview

The messaging system is responsible for sending messages through different providers (such as Signal, email, etc.) based on user preferences and provider availability. It implements a fallback mechanism that tries alternative providers if the primary provider fails.

## Components

### Message Use Case

The `MessageUseCase` is the core component of the messaging system. It implements the business logic for sending messages and retrying failed messages.

```
// IMessageUseCase defines the interface for message use cases
type IMessageUseCase interface {
    SendMessage(request *MessageRequest) (*MessageResponse, error)
    GetMessageStatus(request *MessageStatusRequest) (*MessageStatusResponse, error)
    RetryFailedMessages() error
}
```

### Message Processor

The `MessageProcessor` processes messages asynchronously using a worker pool. It is responsible for the actual sending of messages through the appropriate provider.

```
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

### Provider System

The provider system consists of several components:

1. **Provider**: Represents a message provider (e.g., Signal, email).
2. **UserProvider**: Represents the relationship between a user and a provider, including priority settings.
3. **MessageTransaction**: Represents a message transaction, including status and retry information.
4. **MessageTransactionHistory**: Represents the history of a message transaction.

## Message Flow

The messaging system follows this flow:

1. A client sends a message request to the `/send/message` endpoint.
2. The `SendController` receives the request and delegates it to the `MessageUseCase`.
3. The `MessageUseCase` selects the appropriate provider based on:
   - The requested provider type (if specified)
   - The user's provider configuration and priorities
   - The provider's availability
4. The `MessageUseCase` creates a message transaction record and enqueues it for processing by the `MessageProcessor`.
5. The `MessageProcessor` processes the message asynchronously:
   - It retrieves the message transaction from the queue
   - It sends the message through the selected provider
   - It updates the message transaction with the result
   - If the message fails, it marks it for retry
6. If a message fails, the `RetryFailedMessages` method can be called to retry it using the next provider in the priority list.

## Provider Selection

The provider selection process follows these rules:

1. If the user specifies a provider type in the request, the system tries to use a provider of that type.
2. If no provider type is specified or no matching provider is found, the system uses the highest priority provider.
3. Only active providers are considered (both the provider itself and the user-provider relationship must be active).
4. If a provider fails, the system can retry the message using the next provider in the priority list.

## Message Transaction States

A message transaction can be in one of the following states:

- **pending**: The message is waiting to be processed.
- **processing**: The message is currently being processed.
- **success**: The message was sent successfully.
- **failed**: The message failed to send.

## Message Transaction History

The system maintains a history of message transactions for auditing and reporting purposes. When a message transaction is completed (either successfully or after exhausting all retry attempts), it is moved to the message transaction history table.

The `MessageTransactionHistory` component has the following features:

- Stores a complete record of all processed messages
- Maintains the same data structure as the original message transaction
- Provides methods to retrieve historical message data by ID, message ID, or user ID
- Supports auditing and compliance requirements by preserving message history

The message lifecycle with history tracking works as follows:

1. A new message transaction is created in the `message_transaction` table
2. The message is processed by the `MessageProcessor`
3. If the message is successful, it is moved to the `message_transaction_history` table
4. If the message fails and has retry attempts remaining, it stays in the `message_transaction` table
5. If the message fails and has exhausted all retry attempts, it is moved to the `message_transaction_history` table

The `MoveToHistory` method in the `MessageTransactionRepository` handles the transfer of data from the active transaction table to the history table.

## Retry Mechanism

The retry mechanism works as follows:

1. Failed messages are marked with a `failed` status and a `nextRetryAt` timestamp.
2. The `RetryFailedMessages` method checks for failed messages that are ready for retry.
3. For each failed message, it finds the next provider in the priority list.
4. It creates a new message transaction for the retry and enqueues it for processing.
5. The retry count is incremented for each retry attempt.

## Configuration

The messaging system can be configured through the `config.yaml` file:

```yaml
messaging:
  # Maximum number of retry attempts
  max_retries: 3

  # Delay between retry attempts (in minutes)
  retry_delay: 3

  # Number of worker goroutines for processing messages
  worker_count: 10

  # Default provider type to use if none is specified
  default_provider_type: "signal"
```

## Provider Types

The system supports the following provider types:

- **signal**: Sends messages through the Signal messaging service.
- **email**: Sends messages through email (not fully implemented yet).
- **sms**: Sends messages through SMS (not fully implemented yet).

## Adding a New Provider

To add a new provider type:

1. Create a new provider implementation in the `infrastructure/alerting/provider` directory.
2. Implement the `IAlertProvider` interface.
3. Register the provider in the `AlertService`.
4. Add the provider type to the database.

## Message Status Tracking

The system provides functionality to track the status of messages through the `GetMessageStatus` method in the `MessageUseCase`. This allows users to:

- Check if a message was delivered successfully
- See error details if a message failed
- Monitor the retry count for failed messages
- Access timestamps for message creation and updates

The status information is retrieved from either the active message transaction table or the message transaction history table, depending on whether the message has been archived.

## Example: Sending a Message

Here's an example of how to send a message:

```
// Create a message request
request := &message.MessageRequest{
    Type:       "signal",
    Message:    "Hello, world!",
    Recipients: []string{"+1234567890"},
    UserID:     1,
}

// Send the message
response, err := messageUseCase.SendMessage(request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message sent with ID: %d, Status: %s\n", response.ID, response.Status)
```

## Example: Getting Message Status

Here's an example of how to check the status of a message:

```
// Create a status request
request := &message.MessageStatusRequest{
    ID: 123, // The message transaction ID
}

// Get the message status
response, err := messageUseCase.GetMessageStatus(request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message ID: %d, Status: %s\n", response.ID, response.Status)
if response.ErrorMessage != "" {
    fmt.Printf("Error: %s, Retry Count: %d\n", response.ErrorMessage, response.RetryCount)
}
```

## Example: Retrying Failed Messages

Here's an example of how to retry failed messages:

```
// Retry failed messages
err := messageUseCase.RetryFailedMessages()
if err != nil {
    log.Fatal(err)
}
```
