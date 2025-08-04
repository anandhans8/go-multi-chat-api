package provider

import (
	"time"

	domainErrors "go-multi-chat-api/src/domain/errors"
	domainProvider "go-multi-chat-api/src/domain/provider"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MessageTransactionHistory is the database model for message transaction history
type MessageTransactionHistory struct {
	ID           int       `gorm:"primaryKey"`
	MessageID    int       `gorm:"column:message_id;index"`
	UserID       int       `gorm:"column:user_id;index"`
	ProviderID   int       `gorm:"column:provider_id;index"`
	Recipients   string    `gorm:"column:recipients;type:text"`
	Message      string    `gorm:"column:message;type:text"`
	RequestData  string    `gorm:"column:request_data;type:text"`
	ResponseData string    `gorm:"column:response_data;type:text"`
	Status       string    `gorm:"column:status;index"`
	ErrorMessage string    `gorm:"column:error_message;type:text"`
	RetryCount   int       `gorm:"column:retry_count;default:0"`
	ProcessedAt  time.Time `gorm:"column:processed_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime:mili"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime:mili"`
}

func (MessageTransactionHistory) TableName() string {
	return "message_transaction_history"
}

var ColumnsMessageTransactionHistoryMapping = map[string]string{
	"id":           "id",
	"messageID":    "message_id",
	"userID":       "user_id",
	"providerID":   "provider_id",
	"recipients":   "recipients",
	"message":      "message",
	"requestData":  "request_data",
	"responseData": "response_data",
	"status":       "status",
	"errorMessage": "error_message",
	"retryCount":   "retry_count",
	"processedAt":  "processed_at",
	"createdAt":    "created_at",
	"updatedAt":    "updated_at",
}

// MessageTransactionHistoryRepositoryInterface defines the interface for message transaction history repository operations
type MessageTransactionHistoryRepositoryInterface interface {
	Create(historyDomain *domainProvider.MessageTransactionHistory) (*domainProvider.MessageTransactionHistory, error)
	GetByID(id int) (*domainProvider.MessageTransactionHistory, error)
	GetByMessageID(messageID int) (*[]domainProvider.MessageTransactionHistory, error)
	GetUserMessageTransactionHistory(userID int) (*[]domainProvider.MessageTransactionHistory, error)
}

type MessageTransactionHistoryRepository struct {
	DB     *gorm.DB
	Logger *logger.Logger
}

func NewMessageTransactionHistoryRepository(db *gorm.DB, loggerInstance *logger.Logger) MessageTransactionHistoryRepositoryInterface {
	return &MessageTransactionHistoryRepository{DB: db, Logger: loggerInstance}
}

func (r *MessageTransactionHistoryRepository) Create(historyDomain *domainProvider.MessageTransactionHistory) (*domainProvider.MessageTransactionHistory, error) {
	r.Logger.Info("Creating new message transaction history", zap.Int("messageID", historyDomain.MessageID), zap.Int("providerID", historyDomain.ProviderID))
	historyRepository := messageTransactionHistoryFromDomainMapper(historyDomain)
	txDb := r.DB.Create(historyRepository)
	err := txDb.Error
	if err != nil {
		r.Logger.Error("Error creating message transaction history", zap.Error(err), zap.Int("messageID", historyDomain.MessageID))
		return &domainProvider.MessageTransactionHistory{}, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully created message transaction history", zap.Int("messageID", historyDomain.MessageID), zap.Int("id", historyRepository.ID))
	return historyRepository.toDomainMapper(), err
}

func (r *MessageTransactionHistoryRepository) GetByID(id int) (*domainProvider.MessageTransactionHistory, error) {
	var history MessageTransactionHistory
	err := r.DB.Where("id = ?", id).First(&history).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.Logger.Warn("Message transaction history not found", zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.NotFound)
		} else {
			r.Logger.Error("Error getting message transaction history by ID", zap.Error(err), zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
		return &domainProvider.MessageTransactionHistory{}, err
	}
	r.Logger.Info("Successfully retrieved message transaction history by ID", zap.Int("id", id))
	return history.toDomainMapper(), nil
}

func (r *MessageTransactionHistoryRepository) GetByMessageID(messageID int) (*[]domainProvider.MessageTransactionHistory, error) {
	var histories []MessageTransactionHistory
	if err := r.DB.Where("message_id = ?", messageID).Order("created_at DESC").Find(&histories).Error; err != nil {
		r.Logger.Error("Error getting message transaction history by message ID", zap.Error(err), zap.Int("messageID", messageID))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved message transaction history by message ID", zap.Int("messageID", messageID), zap.Int("count", len(histories)))
	return messageTransactionHistoryArrayToDomainMapper(&histories), nil
}

func (r *MessageTransactionHistoryRepository) GetUserMessageTransactionHistory(userID int) (*[]domainProvider.MessageTransactionHistory, error) {
	var histories []MessageTransactionHistory
	if err := r.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&histories).Error; err != nil {
		r.Logger.Error("Error getting user message transaction history", zap.Error(err), zap.Int("userID", userID))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved user message transaction history", zap.Int("userID", userID), zap.Int("count", len(histories)))
	return messageTransactionHistoryArrayToDomainMapper(&histories), nil
}

// Mappers
func (mth *MessageTransactionHistory) toDomainMapper() *domainProvider.MessageTransactionHistory {
	return &domainProvider.MessageTransactionHistory{
		ID:           mth.ID,
		MessageID:    mth.MessageID,
		UserID:       mth.UserID,
		ProviderID:   mth.ProviderID,
		Recipients:   mth.Recipients,
		Message:      mth.Message,
		RequestData:  mth.RequestData,
		ResponseData: mth.ResponseData,
		Status:       mth.Status,
		ErrorMessage: mth.ErrorMessage,
		RetryCount:   mth.RetryCount,
		ProcessedAt:  mth.ProcessedAt,
		CreatedAt:    mth.CreatedAt,
		UpdatedAt:    mth.UpdatedAt,
	}
}

func messageTransactionHistoryFromDomainMapper(mth *domainProvider.MessageTransactionHistory) *MessageTransactionHistory {
	return &MessageTransactionHistory{
		ID:           mth.ID,
		MessageID:    mth.MessageID,
		UserID:       mth.UserID,
		ProviderID:   mth.ProviderID,
		Recipients:   mth.Recipients,
		Message:      mth.Message,
		RequestData:  mth.RequestData,
		ResponseData: mth.ResponseData,
		Status:       mth.Status,
		ErrorMessage: mth.ErrorMessage,
		RetryCount:   mth.RetryCount,
		ProcessedAt:  mth.ProcessedAt,
		CreatedAt:    mth.CreatedAt,
		UpdatedAt:    mth.UpdatedAt,
	}
}

func messageTransactionHistoryArrayToDomainMapper(histories *[]MessageTransactionHistory) *[]domainProvider.MessageTransactionHistory {
	historiesDomain := make([]domainProvider.MessageTransactionHistory, len(*histories))
	for i, history := range *histories {
		historiesDomain[i] = *history.toDomainMapper()
	}
	return &historiesDomain
}
