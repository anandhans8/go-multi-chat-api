package messaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"go-multi-chat-api/src/domain/provider"
	"go-multi-chat-api/src/infrastructure/alerting/alert"
	logger "go-multi-chat-api/src/infrastructure/logger"
	providerRepo "go-multi-chat-api/src/infrastructure/repository/mysql/provider"
	domainSignal "go-multi-chat-api/src/infrastructure/repository/signal-client"
	"go-multi-chat-api/src/infrastructure/rest/controllers/signal"
	"go-multi-chat-api/src/infrastructure/utils"

	"go.uber.org/zap"
)

// MessageProcessor handles the processing of messages using a worker pool
type MessageProcessor struct {
	signalService                       *domainSignal.SignalClient
	providerRepository                  providerRepo.ProviderRepositoryInterface
	userProviderRepository              providerRepo.UserProviderRepositoryInterface
	messageTransactionRepository        providerRepo.MessageTransactionRepositoryInterface
	messageTransactionHistoryRepository providerRepo.MessageTransactionHistoryRepositoryInterface
	Logger                              *logger.Logger
	workerCount                         int
	messageQueue                        chan *provider.MessageTransaction
	wg                                  sync.WaitGroup
	shutdown                            chan struct{}
}

// WebhookConfig represents the webhook configuration in the user provider config
type WebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
	Enabled    bool   `json:"webhook_enabled"`
}

// NewMessageProcessor creates a new message processor with the specified number of workers
func NewMessageProcessor(
	signalService *domainSignal.SignalClient,
	providerRepository providerRepo.ProviderRepositoryInterface,
	userProviderRepository providerRepo.UserProviderRepositoryInterface,
	messageTransactionRepository providerRepo.MessageTransactionRepositoryInterface,
	messageTransactionHistoryRepository providerRepo.MessageTransactionHistoryRepositoryInterface,
	loggerInstance *logger.Logger,
	workerCount int,
) *MessageProcessor {
	if workerCount <= 0 {
		workerCount = 5 // Default to 5 workers if not specified
	}

	processor := &MessageProcessor{
		signalService:                       signalService,
		providerRepository:                  providerRepository,
		userProviderRepository:              userProviderRepository,
		messageTransactionRepository:        messageTransactionRepository,
		messageTransactionHistoryRepository: messageTransactionHistoryRepository,
		Logger:                              loggerInstance,
		workerCount:                         workerCount,
		messageQueue:                        make(chan *provider.MessageTransaction, 100), // Buffer size of 100
		shutdown:                            make(chan struct{}),
	}

	// Start the worker pool
	processor.startWorkers()

	// Start the watcher for pending messages
	go processor.watchPendingMessages()

	return processor
}

// startWorkers starts the worker pool
func (p *MessageProcessor) startWorkers() {
	p.Logger.Info("Starting message processor workers", zap.Int("workerCount", p.workerCount))

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker processes messages from the queue
func (p *MessageProcessor) worker(id int) {
	defer p.wg.Done()

	p.Logger.Info("Starting message processor worker", zap.Int("workerID", id))

	for {
		select {
		case msg := <-p.messageQueue:
			p.processMessage(msg)
		case <-p.shutdown:
			p.Logger.Info("Shutting down message processor worker", zap.Int("workerID", id))
			return
		}
	}
}

// watchPendingMessages periodically checks for pending messages and undelivered messages and adds them to the queue
func (p *MessageProcessor) watchPendingMessages() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Process pending messages immediately on startup
	p.checkPendingMessages()

	for {
		select {
		case <-ticker.C:
			p.checkPendingMessages()
			p.checkUndeliveredMessages()
		case <-p.shutdown:
			return
		}
	}
}

// checkPendingMessages queries the database for pending messages and adds them to the queue
func (p *MessageProcessor) checkPendingMessages() {
	// Get pending messages
	pendingMessages, err := p.messageTransactionRepository.GetPendingMessages()
	if err != nil {
		p.Logger.Error("Error getting pending messages", zap.Error(err))
		return
	}

	if len(*pendingMessages) == 0 {
		return
	}

	p.Logger.Info("Found pending messages to process", zap.Int("count", len(*pendingMessages)))

	// Add messages to the queue
	for _, msg := range *pendingMessages {
		select {
		case p.messageQueue <- &msg:
			// Message added to queue
		default:
			// Queue is full, log and continue
			p.Logger.Warn("Message queue is full, skipping message", zap.Int("messageID", msg.ID))
		}
	}
}

// checkUndeliveredMessages queries the database for messages that were sent successfully but not delivered within 5 minutes
// and sends them via an alternative provider
func (p *MessageProcessor) checkUndeliveredMessages() {
	// Get undelivered messages
	undeliveredMessages, err := p.messageTransactionRepository.GetUndeliveredMessages()
	if err != nil {
		p.Logger.Error("Error getting undelivered messages", zap.Error(err))
		return
	}

	if len(*undeliveredMessages) == 0 {
		return
	}

	p.Logger.Info("Found undelivered messages to process", zap.Int("count", len(*undeliveredMessages)))

	// Process each undelivered message
	for _, msg := range *undeliveredMessages {
		// Get user providers sorted by priority
		userProviders, err := p.userProviderRepository.GetUserProvidersByPriority(msg.UserID)
		if err != nil {
			p.Logger.Error("Error getting user providers for fallback", zap.Error(err), zap.Int("userID", msg.UserID))
			continue
		}

		// Find the next provider to try (skip the current provider)
		var nextProvider *provider.UserProvider
		for _, up := range *userProviders {
			if up.ProviderID != msg.ProviderID {
				nextProvider = &up
				break
			}
		}

		if nextProvider == nil {
			p.Logger.Warn("No alternative provider found for fallback", zap.Int("userID", msg.UserID), zap.Int("messageID", msg.ID))
			continue
		}

		p.Logger.Info("Found alternative provider for fallback",
			zap.Int("userID", msg.UserID),
			zap.Int("messageID", msg.ID),
			zap.Int("originalProviderID", msg.ProviderID),
			zap.Int("newProviderID", nextProvider.ProviderID))

		// Create a new message transaction with the new provider
		newMsg := &provider.MessageTransaction{
			UserID:     msg.UserID,
			ProviderID: nextProvider.ProviderID,
			Recipients: msg.Recipients,
			Message:    msg.Message,
			Status:     "pending",
			Processing: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Save the new message transaction
		newMsg, err = p.messageTransactionRepository.Create(newMsg)
		if err != nil {
			p.Logger.Error("Error creating fallback message transaction", zap.Error(err), zap.Int("userID", msg.UserID))
			continue
		}

		// Update the original message status to indicate it was not delivered and a fallback was triggered
		updateData := map[string]interface{}{
			"status":       "fallback_triggered",
			"errorMessage": "Message not delivered within 5 minutes, fallback to alternative provider triggered",
			"processing":   false,
		}

		_, err = p.messageTransactionRepository.Update(msg.ID, updateData)
		if err != nil {
			p.Logger.Error("Error updating original message status", zap.Error(err), zap.Int("messageID", msg.ID))
		}

		// Move the original transaction to history
		err = p.messageTransactionRepository.MoveToHistory(msg.ID, p.messageTransactionHistoryRepository)
		if err != nil {
			p.Logger.Error("Error moving original message to history", zap.Error(err), zap.Int("messageID", msg.ID))
		}

		// Add the new message to the queue
		select {
		case p.messageQueue <- newMsg:
			p.Logger.Info("Fallback message added to queue", zap.Int("newMessageID", newMsg.ID), zap.Int("originalMessageID", msg.ID))
		default:
			p.Logger.Warn("Message queue is full, fallback message not queued", zap.Int("newMessageID", newMsg.ID))
		}
	}
}

// EnqueueMessage adds a message to the processing queue
func (p *MessageProcessor) EnqueueMessage(msg *provider.MessageTransaction) {
	select {
	case p.messageQueue <- msg:
		p.Logger.Info("Message added to processing queue", zap.Int("messageID", msg.ID))
	default:
		p.Logger.Warn("Message queue is full, message not queued", zap.Int("messageID", msg.ID))
	}
}

// processMessage processes a single message
func (p *MessageProcessor) processMessage(msg *provider.MessageTransaction) {
	p.Logger.Info("Processing message", zap.Int("messageID", msg.ID), zap.Int("userID", msg.UserID), zap.Int("providerID", msg.ProviderID))

	// Get provider details
	providerDetails, err := p.providerRepository.GetByID(msg.ProviderID)
	if err != nil {
		p.Logger.Error("Error getting provider details", zap.Error(err), zap.Int("providerID", msg.ProviderID))
		p.updateMessageStatus(msg.ID, "failed", err.Error(), "")
		return
	}

	// Skip inactive providers
	if !providerDetails.Status {
		err := errors.New("provider is inactive")
		p.Logger.Warn("Provider is inactive", zap.Int("providerID", msg.ProviderID))
		p.updateMessageStatus(msg.ID, "failed", err.Error(), "")
		return
	}

	// Prepare request data based on provider type
	var requestData []byte
	var responseData []byte
	var sendErr error

	// Parse recipients from JSON
	var recipients []string
	json.Unmarshal([]byte(msg.Recipients), &recipients)

	switch providerDetails.Type {
	case string(alert.TypeSignal):
		// Send via Signal
		var signalRequest = signal.SendMessage{
			Number:     os.Getenv("SIGNAL_FROM_NUMBER"),
			Message:    msg.Message,
			Recipients: recipients,
		}

		textMode := signalRequest.TextMode
		if textMode == nil {
			defaultSignalTextMode := utils.GetEnv("DEFAULT_SIGNAL_TEXT_MODE", "normal")
			if defaultSignalTextMode == "styled" {
				styledStr := "styled"
				textMode = &styledStr
			}
		}

		requestData, _ = json.Marshal(signalRequest)

		data, sendErr := p.signalService.SendV2(
			signalRequest.Number, signalRequest.Message, signalRequest.Recipients, signalRequest.Base64Attachments, signalRequest.Sticker,
			signalRequest.Mentions, signalRequest.QuoteTimestamp, signalRequest.QuoteAuthor, signalRequest.QuoteMessage, signalRequest.QuoteMentions,
			textMode, signalRequest.EditTimestamp, signalRequest.NotifySelf, signalRequest.LinkPreview, signalRequest.ViewOnce)

		if sendErr == nil && data != nil {
			responseData, _ = json.Marshal(data)
		}
	case string(alert.TypeEmail):
		// Email implementation would go here
		sendErr = errors.New("email provider not implemented yet")
	default:
		sendErr = errors.New("unsupported provider type: " + providerDetails.Type)
	}

	// Update transaction with request/response data
	updateData := map[string]interface{}{
		"requestData": string(requestData),
		"processing":  false, // Mark as not being processed anymore
	}

	if sendErr != nil {
		updateData["status"] = "failed"
		updateData["errorMessage"] = sendErr.Error()
		updateData["responseData"] = ""
		// Set next retry time to 3 minutes from now
		nextRetry := time.Now().Add(3 * time.Minute)
		updateData["nextRetryAt"] = nextRetry

		p.Logger.Error("Error sending message",
			zap.Error(sendErr),
			zap.Int("userID", msg.UserID),
			zap.Int("providerID", msg.ProviderID),
			zap.Time("nextRetryAt", nextRetry))

		// Update transaction with error
		_, err = p.messageTransactionRepository.Update(msg.ID, updateData)
		if err != nil {
			p.Logger.Error("Error updating message transaction", zap.Error(err))
		}

		// Move the transaction to history
		err = p.messageTransactionRepository.MoveToHistory(msg.ID, p.messageTransactionHistoryRepository)
		if err != nil {
			p.Logger.Error("Error moving message transaction to history", zap.Error(err), zap.Int("messageID", msg.ID))
		}

		// Send webhook notification for failed message
		p.sendWebhookNotification(msg.UserID, msg.ID, "failed", sendErr.Error())
	} else {
		// Message sent successfully
		updateData["status"] = "success"
		updateData["responseData"] = string(responseData)
		updateData["errorMessage"] = ""

		_, err = p.messageTransactionRepository.Update(msg.ID, updateData)
		if err != nil {
			p.Logger.Error("Error updating message transaction", zap.Error(err))
		}

		// Move the transaction to history
		err = p.messageTransactionRepository.MoveToHistory(msg.ID, p.messageTransactionHistoryRepository)
		if err != nil {
			p.Logger.Error("Error moving message transaction to history", zap.Error(err), zap.Int("messageID", msg.ID))
		}

		p.Logger.Info("Message sent successfully",
			zap.Int("userID", msg.UserID),
			zap.Int("providerID", msg.ProviderID),
			zap.Int("transactionID", msg.ID))

		// Send webhook notification for successful message
		p.sendWebhookNotification(msg.UserID, msg.ID, "success", "")
	}
}

// updateMessageStatus updates the status of a message
func (p *MessageProcessor) updateMessageStatus(id int, status string, errorMessage string, responseData string) {
	updateData := map[string]interface{}{
		"status":       status,
		"errorMessage": errorMessage,
		"processing":   false, // Mark as not being processed anymore
	}

	if responseData != "" {
		updateData["responseData"] = responseData
	}

	if status == "failed" {
		// Set next retry time to 3 minutes from now
		nextRetry := time.Now().Add(3 * time.Minute)
		updateData["nextRetryAt"] = nextRetry
	}

	_, err := p.messageTransactionRepository.Update(id, updateData)
	if err != nil {
		p.Logger.Error("Error updating message status", zap.Error(err), zap.Int("messageID", id))
	}

	// Move the transaction to history if it's completed (success or failed)
	if status == "success" || status == "failed" {
		err = p.messageTransactionRepository.MoveToHistory(id, p.messageTransactionHistoryRepository)
		if err != nil {
			p.Logger.Error("Error moving message transaction to history", zap.Error(err), zap.Int("messageID", id))
		}
	}
}

// sendWebhookNotification sends a webhook notification for a message status update
func (p *MessageProcessor) sendWebhookNotification(userID int, messageID int, status string, errorMessage string) {
	// Get user providers
	userProviders, err := p.userProviderRepository.GetUserProviders(userID)
	if err != nil {
		p.Logger.Error("Error getting user providers for webhook notification", zap.Error(err), zap.Int("userID", userID))
		return
	}

	// Check each provider for webhook configuration
	for _, up := range *userProviders {
		// Parse config to check for webhook URL
		var config WebhookConfig
		if up.Config != "" {
			err := json.Unmarshal([]byte(up.Config), &config)
			if err != nil {
				p.Logger.Error("Error parsing user provider config", zap.Error(err), zap.Int("userProviderID", up.ID))
				continue
			}

			// If webhook is enabled and URL is set, send notification
			if config.Enabled && config.WebhookURL != "" {
				// Prepare webhook payload
				payload := map[string]interface{}{
					"message_id": messageID,
					"user_id":    userID,
					"status":     status,
					"timestamp":  time.Now().Unix(),
				}

				if errorMessage != "" {
					payload["error"] = errorMessage
				}

				// Send webhook request
				go p.sendWebhookRequest(config.WebhookURL, payload)
			}
		}
	}
}

// sendWebhookRequest sends an HTTP request to the webhook URL
func (p *MessageProcessor) sendWebhookRequest(webhookURL string, payload map[string]interface{}) {
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		p.Logger.Error("Error marshaling webhook payload", zap.Error(err))
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		p.Logger.Error("Error creating webhook request", zap.Error(err), zap.String("webhookURL", webhookURL))
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-multi-chat-api-Webhook")

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		p.Logger.Error("Error sending webhook request", zap.Error(err), zap.String("webhookURL", webhookURL))
		return
	}
	defer resp.Body.Close()

	// Log response
	p.Logger.Info("Webhook notification sent",
		zap.String("webhookURL", webhookURL),
		zap.Int("statusCode", resp.StatusCode))
}

// Shutdown gracefully shuts down the message processor
func (p *MessageProcessor) Shutdown() {
	p.Logger.Info("Shutting down message processor")

	// Signal all workers to shut down
	close(p.shutdown)

	// Wait for all workers to finish
	p.wg.Wait()

	p.Logger.Info("Message processor shutdown complete")
}
