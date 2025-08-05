package provider

import (
	"time"

	domainErrors "go-multi-chat-api/src/domain/errors"
	domainProvider "go-multi-chat-api/src/domain/provider"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MessageTransaction is the database model for message transactions
type MessageTransaction struct {
	ID           int        `gorm:"primaryKey"`
	UserID       int        `gorm:"column:user_id;index"`
	ProviderID   int        `gorm:"column:provider_id;index"`
	Recipients   string     `gorm:"column:recipients;type:text"`
	Message      string     `gorm:"column:message;type:text"`
	RequestData  string     `gorm:"column:request_data;type:text"`
	ResponseData string     `gorm:"column:response_data;type:text"`
	Status       string     `gorm:"column:status;index"`
	ErrorMessage string     `gorm:"column:error_message;type:text"`
	RetryCount   int        `gorm:"column:retry_count;default:0"`
	NextRetryAt  *time.Time `gorm:"column:next_retry_at;index"`
	Processing   bool       `gorm:"column:processing;default:false;index"`
	ProcessedAt  *time.Time `gorm:"column:processed_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime:mili"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime:mili"`
}

func (MessageTransaction) TableName() string {
	return "message_transactions"
}

var ColumnsMessageTransactionMapping = map[string]string{
	"id":           "id",
	"userID":       "user_id",
	"providerID":   "provider_id",
	"recipients":   "recipients",
	"message":      "message",
	"requestData":  "request_data",
	"responseData": "response_data",
	"status":       "status",
	"errorMessage": "error_message",
	"retryCount":   "retry_count",
	"nextRetryAt":  "next_retry_at",
	"processing":   "processing",
	"processedAt":  "processed_at",
	"createdAt":    "created_at",
	"updatedAt":    "updated_at",
}

// MessageTransactionRepositoryInterface defines the interface for message transaction repository operations
type MessageTransactionRepositoryInterface interface {
	Create(messageTransactionDomain *domainProvider.MessageTransaction) (*domainProvider.MessageTransaction, error)
	GetByID(id int) (*domainProvider.MessageTransaction, error)
	GetUserMessageTransactions(userID int) (*[]domainProvider.MessageTransaction, error)
	Update(id int, messageTransactionMap map[string]interface{}) (*domainProvider.MessageTransaction, error)
	GetFailedMessagesForRetry() (*[]domainProvider.MessageTransaction, error)
	GetPendingMessages() (*[]domainProvider.MessageTransaction, error)
	GetUndeliveredMessages() (*[]domainProvider.MessageTransaction, error)
	MoveToHistory(id int, historyRepository MessageTransactionHistoryRepositoryInterface) error
	CountUserMessagesForToday(userID int) (int, error)
}

type MessageTransactionRepository struct {
	DB     *gorm.DB
	Logger *logger.Logger
}

func NewMessageTransactionRepository(db *gorm.DB, loggerInstance *logger.Logger) MessageTransactionRepositoryInterface {
	return &MessageTransactionRepository{DB: db, Logger: loggerInstance}
}

func (r *MessageTransactionRepository) Create(messageTransactionDomain *domainProvider.MessageTransaction) (*domainProvider.MessageTransaction, error) {
	r.Logger.Info("Creating new message transaction", zap.Int("userID", messageTransactionDomain.UserID), zap.Int("providerID", messageTransactionDomain.ProviderID))
	messageTransactionRepository := messageTransactionFromDomainMapper(messageTransactionDomain)
	txDb := r.DB.Create(messageTransactionRepository)
	err := txDb.Error
	if err != nil {
		r.Logger.Error("Error creating message transaction", zap.Error(err), zap.Int("userID", messageTransactionDomain.UserID))
		return &domainProvider.MessageTransaction{}, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully created message transaction", zap.Int("userID", messageTransactionDomain.UserID), zap.Int("id", messageTransactionRepository.ID))
	return messageTransactionRepository.toDomainMapper(), err
}

func (r *MessageTransactionRepository) GetByID(id int) (*domainProvider.MessageTransaction, error) {
	var messageTransaction MessageTransaction
	err := r.DB.Where("id = ?", id).First(&messageTransaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.Logger.Warn("Message transaction not found", zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.NotFound)
		} else {
			r.Logger.Error("Error getting message transaction by ID", zap.Error(err), zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
		return &domainProvider.MessageTransaction{}, err
	}
	r.Logger.Info("Successfully retrieved message transaction by ID", zap.Int("id", id))
	return messageTransaction.toDomainMapper(), nil
}

func (r *MessageTransactionRepository) GetUserMessageTransactions(userID int) (*[]domainProvider.MessageTransaction, error) {
	var messageTransactions []MessageTransaction
	if err := r.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&messageTransactions).Error; err != nil {
		r.Logger.Error("Error getting user message transactions", zap.Error(err), zap.Int("userID", userID))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved user message transactions", zap.Int("userID", userID), zap.Int("count", len(messageTransactions)))
	return messageTransactionArrayToDomainMapper(&messageTransactions), nil
}

func (r *MessageTransactionRepository) Update(id int, messageTransactionMap map[string]interface{}) (*domainProvider.MessageTransaction, error) {
	var messageTransactionObj MessageTransaction
	messageTransactionObj.ID = id

	// Map JSON field names to DB column names
	updateData := make(map[string]interface{})
	for k, v := range messageTransactionMap {
		if column, ok := ColumnsMessageTransactionMapping[k]; ok {
			updateData[column] = v
		} else {
			updateData[k] = v
		}
	}

	err := r.DB.Model(&messageTransactionObj).
		Updates(updateData).Error
	if err != nil {
		r.Logger.Error("Error updating message transaction", zap.Error(err), zap.Int("id", id))
		return &domainProvider.MessageTransaction{}, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	if err := r.DB.Where("id = ?", id).First(&messageTransactionObj).Error; err != nil {
		r.Logger.Error("Error retrieving updated message transaction", zap.Error(err), zap.Int("id", id))
		return &domainProvider.MessageTransaction{}, err
	}
	r.Logger.Info("Successfully updated message transaction", zap.Int("id", id))
	return messageTransactionObj.toDomainMapper(), nil
}

// GetFailedMessagesForRetry retrieves failed message transactions that are ready for retry
func (r *MessageTransactionRepository) GetFailedMessagesForRetry() (*[]domainProvider.MessageTransaction, error) {
	var messageTransactions []MessageTransaction

	// Get failed messages where next_retry_at is in the past
	now := time.Now()
	if err := r.DB.Where("status = ? AND next_retry_at <= ?", "failed", now).
		Find(&messageTransactions).Error; err != nil {
		r.Logger.Error("Error getting failed messages for retry", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	r.Logger.Info("Successfully retrieved failed messages for retry", zap.Int("count", len(messageTransactions)))
	return messageTransactionArrayToDomainMapper(&messageTransactions), nil
}

// GetPendingMessages retrieves pending message transactions and locks them for processing
// It retrieves up to 1000 messages that are not currently being processed
func (r *MessageTransactionRepository) GetPendingMessages() (*[]domainProvider.MessageTransaction, error) {
	var messageTransactions []MessageTransaction

	// Start a transaction
	tx := r.DB.Begin()
	if tx.Error != nil {
		r.Logger.Error("Error starting transaction", zap.Error(tx.Error))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	// Get messages with status "pending" that are not being processed, limited to 1000
	if err := tx.Where("status = ? AND processing = ?", "pending", false).
		Limit(1000).
		Find(&messageTransactions).Error; err != nil {
		tx.Rollback()
		r.Logger.Error("Error getting pending messages", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	// If no messages found, commit the transaction and return
	if len(messageTransactions) == 0 {
		tx.Commit()
		return &[]domainProvider.MessageTransaction{}, nil
	}

	// Get the IDs of the messages to lock
	var messageIDs []int
	for _, msg := range messageTransactions {
		messageIDs = append(messageIDs, msg.ID)
	}

	// Mark the messages as being processed
	now := time.Now()
	if err := tx.Model(&MessageTransaction{}).
		Where("id IN (?)", messageIDs).
		Updates(map[string]interface{}{
			"processing":   true,
			"processed_at": now,
		}).Error; err != nil {
		tx.Rollback()
		r.Logger.Error("Error locking pending messages", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		r.Logger.Error("Error committing transaction", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	r.Logger.Info("Successfully retrieved and locked pending messages", zap.Int("count", len(messageTransactions)))
	return messageTransactionArrayToDomainMapper(&messageTransactions), nil
}

// Mappers
func (mt *MessageTransaction) toDomainMapper() *domainProvider.MessageTransaction {
	return &domainProvider.MessageTransaction{
		ID:           mt.ID,
		UserID:       mt.UserID,
		ProviderID:   mt.ProviderID,
		Recipients:   mt.Recipients,
		Message:      mt.Message,
		RequestData:  mt.RequestData,
		ResponseData: mt.ResponseData,
		Status:       mt.Status,
		ErrorMessage: mt.ErrorMessage,
		RetryCount:   mt.RetryCount,
		//NextRetryAt:  mt.NextRetryAt,
		Processing: mt.Processing,
		//ProcessedAt:  mt.ProcessedAt,
		CreatedAt: mt.CreatedAt,
		UpdatedAt: mt.UpdatedAt,
	}
}

func messageTransactionFromDomainMapper(mt *domainProvider.MessageTransaction) *MessageTransaction {
	return &MessageTransaction{
		ID:           mt.ID,
		UserID:       mt.UserID,
		ProviderID:   mt.ProviderID,
		Recipients:   mt.Recipients,
		Message:      mt.Message,
		RequestData:  mt.RequestData,
		ResponseData: mt.ResponseData,
		Status:       mt.Status,
		ErrorMessage: mt.ErrorMessage,
		RetryCount:   mt.RetryCount,
		//NextRetryAt:  mt.NextRetryAt,
		Processing: mt.Processing,
		//ProcessedAt:  mt.ProcessedAt,
		CreatedAt: mt.CreatedAt,
		UpdatedAt: mt.UpdatedAt,
	}
}

func messageTransactionArrayToDomainMapper(messageTransactions *[]MessageTransaction) *[]domainProvider.MessageTransaction {
	messageTransactionsDomain := make([]domainProvider.MessageTransaction, len(*messageTransactions))
	for i, messageTransaction := range *messageTransactions {
		messageTransactionsDomain[i] = *messageTransaction.toDomainMapper()
	}
	return &messageTransactionsDomain
}

// GetUndeliveredMessages retrieves messages that were sent successfully but not delivered within 5 minutes
func (r *MessageTransactionRepository) GetUndeliveredMessages() (*[]domainProvider.MessageTransaction, error) {
	var messageTransactions []MessageTransaction

	// Get messages that were sent successfully more than 5 minutes ago
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	if err := r.DB.Where("status = ? AND processing = ? AND updated_at <= ?", "success", false, fiveMinutesAgo).
		Find(&messageTransactions).Error; err != nil {
		r.Logger.Error("Error getting undelivered messages", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	r.Logger.Info("Successfully retrieved undelivered messages", zap.Int("count", len(messageTransactions)))
	return messageTransactionArrayToDomainMapper(&messageTransactions), nil
}

// MoveToHistory moves a message transaction to the history table
func (r *MessageTransactionRepository) MoveToHistory(id int, historyRepository MessageTransactionHistoryRepositoryInterface) error {
	// Get the message transaction
	messageTransaction, err := r.GetByID(id)
	if err != nil {
		r.Logger.Error("Error getting message transaction for history", zap.Error(err), zap.Int("id", id))
		return err
	}

	// Create a new history entry
	history := &domainProvider.MessageTransactionHistory{
		MessageID:    messageTransaction.ID,
		UserID:       messageTransaction.UserID,
		ProviderID:   messageTransaction.ProviderID,
		Recipients:   messageTransaction.Recipients,
		Message:      messageTransaction.Message,
		RequestData:  messageTransaction.RequestData,
		ResponseData: messageTransaction.ResponseData,
		Status:       messageTransaction.Status,
		ErrorMessage: messageTransaction.ErrorMessage,
		RetryCount:   messageTransaction.RetryCount,
		ProcessedAt:  messageTransaction.UpdatedAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save the history entry
	_, err = historyRepository.Create(history)
	if err != nil {
		r.Logger.Error("Error creating message transaction history", zap.Error(err), zap.Int("id", id))
		return err
	}

	r.Logger.Info("Successfully moved message transaction to history", zap.Int("id", id))
	return nil
}

// CountUserMessagesForToday counts the number of messages sent by a user on the current day
func (r *MessageTransactionRepository) CountUserMessagesForToday(userID int) (int, error) {
	r.Logger.Info("Counting messages sent by user today", zap.Int("userID", userID))

	// Get the start and end of the current day in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var count int64
	err := r.DB.Model(&MessageTransaction{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startOfDay, endOfDay).
		Count(&count).Error

	if err != nil {
		r.Logger.Error("Error counting user messages for today", zap.Error(err), zap.Int("userID", userID))
		return 0, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}

	r.Logger.Info("Successfully counted user messages for today",
		zap.Int("userID", userID),
		zap.Int64("count", count))

	return int(count), nil
}
