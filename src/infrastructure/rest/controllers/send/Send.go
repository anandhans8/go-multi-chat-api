package send

import (
	"errors"
	"go-multi-chat-api/src/application/usecases/message"
	"go-multi-chat-api/src/domain/common"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ISendController interface {
	Message(c *gin.Context)
	RetryFailedMessages()
	GetMessageStatus(c *gin.Context)
}

type SendController struct {
	commonService  common.CommonService
	messageUseCase message.IMessageUseCase
	Logger         *logger.Logger
}

func NewSendController(
	commonService common.CommonService,
	messageUseCase message.IMessageUseCase,
	loggerInstance *logger.Logger,
) ISendController {
	return &SendController{
		commonService:  commonService,
		messageUseCase: messageUseCase,
		Logger:         loggerInstance,
	}
}

// RetryFailedMessages delegates to the message use case to retry failed messages
func (c *SendController) RetryFailedMessages() {
	c.Logger.Info("Retrying failed messages")
	err := c.messageUseCase.RetryFailedMessages()
	if err != nil {
		c.Logger.Error("Error retrying failed messages", zap.Error(err))
	}
}

func (c *SendController) Message(ctx *gin.Context) {
	var request MessageRequest
	err := ctx.ShouldBindJSON(&request)
	if err != nil {
		c.Logger.Error("Couldn't process request - invalid request", zap.Error(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			c.Logger.Error("Validation errors occurred", zap.Any("errors", ve))
			c.commonService.AppendValidationErrors(ctx, ve, request)
			return
		}
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// Convert controller request to use case request
	useCaseRequest := &message.MessageRequest{
		Type:       request.Type,
		Message:    request.Message,
		Recipients: request.Recipients,
		UserID:     request.UserID,
	}

	// Call the use case
	useCaseResponse, err := c.messageUseCase.SendMessage(useCaseRequest)
	if err != nil {
		c.Logger.Error("Error sending message", zap.Error(err), zap.Int("userID", request.UserID))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending message"})
		return
	}

	// Convert use case response to controller response
	response := &MessageResponse{
		ID:      useCaseResponse.ID,
		Status:  useCaseResponse.Status,
		Message: useCaseResponse.Message,
	}

	c.Logger.Info("Message queued for processing",
		zap.Int("userID", request.UserID),
		zap.Int("transactionID", useCaseResponse.ID))

	// Return accepted response
	ctx.JSON(http.StatusAccepted, response)
}

// GetMessageStatus handles requests to check the status of a message
func (c *SendController) GetMessageStatus(ctx *gin.Context) {
	var request MessageStatusRequest
	if err := ctx.ShouldBindUri(&request); err != nil {
		c.Logger.Error("Invalid message ID", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Convert controller request to use case request
	useCaseRequest := &message.MessageStatusRequest{
		ID: request.ID,
	}

	// Call the use case
	useCaseResponse, err := c.messageUseCase.GetMessageStatus(useCaseRequest)
	if err != nil {
		c.Logger.Error("Error getting message status", zap.Error(err), zap.Int("messageID", request.ID))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting message status"})
		return
	}

	// Convert use case response to controller response
	response := &MessageStatusResponse{
		ID:           useCaseResponse.ID,
		Status:       useCaseResponse.Status,
		Message:      useCaseResponse.Message,
		Recipients:   useCaseResponse.Recipients,
		ErrorMessage: useCaseResponse.ErrorMessage,
		RetryCount:   useCaseResponse.RetryCount,
		CreatedAt:    useCaseResponse.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    useCaseResponse.UpdatedAt.Format(time.RFC3339),
	}

	c.Logger.Info("Retrieved message status", zap.Int("messageID", request.ID), zap.String("status", useCaseResponse.Status))
	ctx.JSON(http.StatusOK, response)
}
