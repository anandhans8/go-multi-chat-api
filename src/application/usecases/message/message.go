package message

import (
	"encoding/json"
	"errors"
	"go-multi-chat-api/src/domain/provider"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/messaging"
	providerRepo "go-multi-chat-api/src/infrastructure/repository/mysql/provider"
	userRepo "go-multi-chat-api/src/infrastructure/repository/mysql/user"
	"time"

	"go.uber.org/zap"
)

// MessageRequest represents a request to send a message
type MessageRequest struct {
	Type       string
	Message    string
	Recipients []string
	UserID     int
}

// MessageResponse represents the response from sending a message
type MessageResponse struct {
	ID      int
	Status  string
	Message string
}

// MessageStatusRequest represents a request to check message status
type MessageStatusRequest struct {
	ID int
}

// MessageStatusResponse represents the response from checking message status
type MessageStatusResponse struct {
	ID           int
	Status       string
	Message      string
	Recipients   string
	ErrorMessage string
	RetryCount   int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IMessageUseCase defines the interface for message use cases
type IMessageUseCase interface {
	SendMessage(request *MessageRequest) (*MessageResponse, error)
	RetryFailedMessages() error
	GetMessageStatus(request *MessageStatusRequest) (*MessageStatusResponse, error)
}

// MessageUseCase implements the IMessageUseCase interface
type MessageUseCase struct {
	providerRepository           providerRepo.ProviderRepositoryInterface
	userProviderRepository       providerRepo.UserProviderRepositoryInterface
	messageTransactionRepository providerRepo.MessageTransactionRepositoryInterface
	messageProcessor             *messaging.MessageProcessor
	userRepository               userRepo.UserRepositoryInterface
	Logger                       *logger.Logger
}

// NewMessageUseCase creates a new MessageUseCase
func NewMessageUseCase(
	providerRepository providerRepo.ProviderRepositoryInterface,
	userProviderRepository providerRepo.UserProviderRepositoryInterface,
	messageTransactionRepository providerRepo.MessageTransactionRepositoryInterface,
	messageProcessor *messaging.MessageProcessor,
	userRepository userRepo.UserRepositoryInterface,
	loggerInstance *logger.Logger,
) IMessageUseCase {
	return &MessageUseCase{
		providerRepository:           providerRepository,
		userProviderRepository:       userProviderRepository,
		messageTransactionRepository: messageTransactionRepository,
		messageProcessor:             messageProcessor,
		userRepository:               userRepository,
		Logger:                       loggerInstance,
	}
}

// SendMessage sends a message using the appropriate provider
func (m *MessageUseCase) SendMessage(request *MessageRequest) (*MessageResponse, error) {
	// Check user's daily message rate limit
	user, err := m.userRepository.GetByID(request.UserID)
	if err != nil {
		m.Logger.Error("Error getting user", zap.Error(err), zap.Int("userID", request.UserID))
		return nil, err
	}

	// Count messages sent by user today
	messageCount, err := m.messageTransactionRepository.CountUserMessagesForToday(request.UserID)
	if err != nil {
		m.Logger.Error("Error counting user messages for today", zap.Error(err), zap.Int("userID", request.UserID))
		return nil, err
	}

	// Check if user has exceeded their daily message limit
	if messageCount >= user.MessageRateLimit {
		m.Logger.Warn("User has exceeded daily message rate limit",
			zap.Int("userID", request.UserID),
			zap.Int("messageCount", messageCount),
			zap.Int("rateLimit", user.MessageRateLimit))
		return nil, errors.New("daily message rate limit exceeded")
	}

	// Get user providers by priority
	userProviders, err := m.userProviderRepository.GetUserProvidersByPriority(request.UserID)
	if err != nil {
		m.Logger.Error("Error getting user providers", zap.Error(err), zap.Int("userID", request.UserID))
		return nil, err
	}

	if len(*userProviders) == 0 {
		m.Logger.Error("No providers configured for user", zap.Int("userID", request.UserID))
		return nil, err
	}

	// If user specified a provider type, try that provider first
	var selectedProvider provider.UserProvider
	if request.Type != "" {
		// Find providers matching the requested type
		var matchingProviders []provider.UserProvider
		for _, up := range *userProviders {
			providerDetails, err := m.providerRepository.GetByID(up.ProviderID)
			if err != nil {
				continue
			}
			if providerDetails.Type == request.Type && providerDetails.Status && up.Status {
				matchingProviders = append(matchingProviders, up)
			}
		}

		// If we found matching providers, use the highest priority one
		if len(matchingProviders) > 0 {
			selectedProvider = matchingProviders[0]
		} else {
			// No matching providers, fall back to highest priority provider
			for _, up := range *userProviders {
				providerDetails, err := m.providerRepository.GetByID(up.ProviderID)
				if err != nil {
					continue
				}
				if providerDetails.Status && up.Status {
					selectedProvider = up
					break
				}
			}

			m.Logger.Warn("No matching providers found for requested type, using highest priority provider",
				zap.String("type", request.Type),
				zap.Int("userID", request.UserID),
				zap.Int("providerID", selectedProvider.ProviderID))
		}
	} else {
		// No specific type requested, use highest priority provider
		for _, up := range *userProviders {
			providerDetails, err := m.providerRepository.GetByID(up.ProviderID)
			if err != nil {
				continue
			}
			if providerDetails.Status && up.Status {
				selectedProvider = up
				break
			}
		}
	}

	// Verify that the provider exists
	_, err = m.providerRepository.GetByID(selectedProvider.ProviderID)
	if err != nil {
		m.Logger.Error("Error getting provider details", zap.Error(err), zap.Int("providerID", selectedProvider.ProviderID))
		return nil, err
	}

	// Create message transaction record
	recipientsJSON, _ := json.Marshal(request.Recipients)
	messageTransaction := &provider.MessageTransaction{
		UserID:      request.UserID,
		ProviderID:  selectedProvider.ProviderID,
		Recipients:  string(recipientsJSON),
		Message:     request.Message,
		Status:      "pending",
		RetryCount:  0,
		NextRetryAt: time.Time{}, // Zero time
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save initial transaction record
	messageTransaction, err = m.messageTransactionRepository.Create(messageTransaction)
	if err != nil {
		m.Logger.Error("Error creating message transaction", zap.Error(err))
		return nil, err
	}

	// Enqueue the message for processing by the message processor
	m.messageProcessor.EnqueueMessage(messageTransaction)

	// Return immediate response to the user
	response := &MessageResponse{
		ID:      messageTransaction.ID,
		Status:  "pending",
		Message: "Message queued for processing",
	}

	m.Logger.Info("Message queued for processing",
		zap.Int("userID", request.UserID),
		zap.Int("providerID", selectedProvider.ProviderID),
		zap.Int("transactionID", messageTransaction.ID))

	return response, nil
}

// GetMessageStatus retrieves the status of a message by its ID
func (m *MessageUseCase) GetMessageStatus(request *MessageStatusRequest) (*MessageStatusResponse, error) {
	// Get the message transaction by ID
	messageTransaction, err := m.messageTransactionRepository.GetByID(request.ID)
	if err != nil {
		m.Logger.Error("Error getting message status", zap.Error(err), zap.Int("messageID", request.ID))
		return nil, err
	}

	// Convert to response
	response := &MessageStatusResponse{
		ID:           messageTransaction.ID,
		Status:       messageTransaction.Status,
		Message:      messageTransaction.Message,
		Recipients:   messageTransaction.Recipients,
		ErrorMessage: messageTransaction.ErrorMessage,
		RetryCount:   messageTransaction.RetryCount,
		CreatedAt:    messageTransaction.CreatedAt,
		UpdatedAt:    messageTransaction.UpdatedAt,
	}

	m.Logger.Info("Retrieved message status", zap.Int("messageID", request.ID), zap.String("status", messageTransaction.Status))
	return response, nil
}

// RetryFailedMessages checks for failed messages that are ready for retry
func (m *MessageUseCase) RetryFailedMessages() error {
	// Get failed messages ready for retry
	failedMessages, err := m.messageTransactionRepository.GetFailedMessagesForRetry()
	if err != nil {
		m.Logger.Error("Error getting failed messages for retry", zap.Error(err))
		return err
	}

	if len(*failedMessages) == 0 {
		m.Logger.Debug("No failed messages to retry")
		return nil
	}

	m.Logger.Info("Found failed messages to retry", zap.Int("count", len(*failedMessages)))

	// Process each failed message
	for _, failedMsg := range *failedMessages {
		// Get user providers by priority
		userProviders, err := m.userProviderRepository.GetUserProvidersByPriority(failedMsg.UserID)
		if err != nil {
			m.Logger.Error("Error getting user providers for retry", zap.Error(err), zap.Int("userID", failedMsg.UserID))
			continue
		}

		if len(*userProviders) == 0 {
			m.Logger.Error("No providers configured for user", zap.Int("userID", failedMsg.UserID))
			continue
		}

		// Find the next provider to try (after the one that failed)
		var nextProviderFound bool = false
		for i, userProvider := range *userProviders {
			// Skip providers until we find the one that failed
			if userProvider.ProviderID == failedMsg.ProviderID {
				// If there's a next provider in the list, use it
				if i+1 < len(*userProviders) {
					nextProviderFound = true

					// Get the next provider
					nextProvider := (*userProviders)[i+1]

					// Get provider details
					providerDetails, err := m.providerRepository.GetByID(nextProvider.ProviderID)
					if err != nil {
						m.Logger.Error("Error getting provider details for retry", zap.Error(err), zap.Int("providerID", nextProvider.ProviderID))
						continue
					}

					// Skip inactive providers
					if !providerDetails.Status || !nextProvider.Status {
						m.Logger.Warn("Next provider is inactive, skipping", zap.Int("providerID", nextProvider.ProviderID))
						continue
					}

					// Create a new message transaction for the retry
					var recipients []string
					json.Unmarshal([]byte(failedMsg.Recipients), &recipients)

					newTransaction := &provider.MessageTransaction{
						UserID:     failedMsg.UserID,
						ProviderID: nextProvider.ProviderID,
						Recipients: failedMsg.Recipients,
						Message:    failedMsg.Message,
						Status:     "pending",
						RetryCount: failedMsg.RetryCount + 1,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}

					// Save initial transaction record
					newTransaction, err = m.messageTransactionRepository.Create(newTransaction)
					if err != nil {
						m.Logger.Error("Error creating message transaction for retry", zap.Error(err))
						continue
					}

					// Enqueue the message for processing
					m.messageProcessor.EnqueueMessage(newTransaction)

					m.Logger.Info("Retry message queued for processing",
						zap.Int("userID", failedMsg.UserID),
						zap.Int("providerID", nextProvider.ProviderID),
						zap.Int("transactionID", newTransaction.ID),
						zap.Int("retryCount", newTransaction.RetryCount))

					break
				}
			}
		}

		if !nextProviderFound {
			m.Logger.Warn("No next provider found for retry",
				zap.Int("userID", failedMsg.UserID),
				zap.Int("failedProviderID", failedMsg.ProviderID))
		}
	}

	return nil
}
