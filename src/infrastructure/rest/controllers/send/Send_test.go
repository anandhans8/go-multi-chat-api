package send

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-multi-chat-api/src/application/usecases/message"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// MockMessageUseCase implements message.IMessageUseCase for testing
type MockMessageUseCase struct {
	sendMessageFunc         func(*message.MessageRequest) (*message.MessageResponse, error)
	retryFailedMessagesFunc func() error
	getMessageStatusFunc    func(*message.MessageStatusRequest) (*message.MessageStatusResponse, error)
}

func (m *MockMessageUseCase) SendMessage(req *message.MessageRequest) (*message.MessageResponse, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(req)
	}
	return nil, nil
}

func (m *MockMessageUseCase) RetryFailedMessages() error {
	if m.retryFailedMessagesFunc != nil {
		return m.retryFailedMessagesFunc()
	}
	return nil
}

func (m *MockMessageUseCase) GetMessageStatus(req *message.MessageStatusRequest) (*message.MessageStatusResponse, error) {
	if m.getMessageStatusFunc != nil {
		return m.getMessageStatusFunc(req)
	}
	return nil, nil
}

// MockCommonService mocks the common service for testing
type MockCommonService struct {
	appendValidationErrorsFunc func(*gin.Context, validator.ValidationErrors, interface{})
	generateRandomIntegerFunc  func() string
}

func (m *MockCommonService) AppendValidationErrors(ctx *gin.Context, ve validator.ValidationErrors, intr interface{}) {
	if m.appendValidationErrorsFunc != nil {
		m.appendValidationErrorsFunc(ctx, ve, intr)
	}
}

func (m *MockCommonService) GenerateRandomInteger() string {
	if m.generateRandomIntegerFunc != nil {
		return m.generateRandomIntegerFunc()
	}
	return "1234"
}

func setupLogger(t *testing.T) *logger.Logger {
	loggerInstance, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return loggerInstance
}

func TestNewSendController(t *testing.T) {
	mockCommonService := &MockCommonService{}
	mockMessageUseCase := &MockMessageUseCase{}
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	if controller == nil {
		t.Error("Expected NewSendController to return a non-nil controller")
	}
}

func TestSendController_Message_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockMessageUseCase := &MockMessageUseCase{
		sendMessageFunc: func(req *message.MessageRequest) (*message.MessageResponse, error) {
			return &message.MessageResponse{
				ID:      123,
				Status:  "queued",
				Message: "Message queued for processing",
			}, nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Create test request
	messageRequest := MessageRequest{
		Type:       "signal",
		Message:    "Test message",
		Recipients: []string{"+1234567890"},
		UserID:     1,
	}

	requestBody, _ := json.Marshal(messageRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/send", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Message(c)

	// Check response
	assert.Equal(t, http.StatusAccepted, w.Code)

	// Parse response
	var response MessageResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response fields
	assert.Equal(t, 123, response.ID)
	assert.Equal(t, "queued", response.Status)
	assert.Equal(t, "Message queued for processing", response.Message)
}

func TestSendController_Message_ValidationError(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock common service that handles validation errors
	mockCommonService := &MockCommonService{
		appendValidationErrorsFunc: func(ctx *gin.Context, ve validator.ValidationErrors, intr interface{}) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": []string{"validation error"}})
		},
	}

	// Create mock use case
	mockMessageUseCase := &MockMessageUseCase{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Create invalid request (missing required fields)
	requestBody := []byte(`{"type": "signal"}`) // Missing message, recipients, and userID

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/send", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Message(c)

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendController_Message_UseCaseError(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case with error
	mockMessageUseCase := &MockMessageUseCase{
		sendMessageFunc: func(req *message.MessageRequest) (*message.MessageResponse, error) {
			return nil, errors.New("failed to send message")
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Create test request
	messageRequest := MessageRequest{
		Type:       "signal",
		Message:    "Test message",
		Recipients: []string{"+1234567890"},
		UserID:     1,
	}

	requestBody, _ := json.Marshal(messageRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/send", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Message(c)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSendController_GetMessageStatus_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockMessageUseCase := &MockMessageUseCase{
		getMessageStatusFunc: func(req *message.MessageStatusRequest) (*message.MessageStatusResponse, error) {
			return &message.MessageStatusResponse{
				ID:           123,
				Status:       "delivered",
				Message:      "Test message",
				Recipients:   "+1234567890",
				ErrorMessage: "",
				RetryCount:   0,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/messages/123", nil)

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "123"}}

	// Set up the URI parameters
	var messageStatusRequest MessageStatusRequest
	messageStatusRequest.ID = 123
	c.Set("MessageStatusRequest", messageStatusRequest)

	// Call the method
	controller.GetMessageStatus(c)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response MessageStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response fields
	assert.Equal(t, 123, response.ID)
	assert.Equal(t, "delivered", response.Status)
	assert.Equal(t, "Test message", response.Message)
	assert.Equal(t, "+1234567890", response.Recipients)
	assert.Equal(t, "", response.ErrorMessage)
	assert.Equal(t, 0, response.RetryCount)
}

func TestSendController_GetMessageStatus_Error(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case with error
	mockMessageUseCase := &MockMessageUseCase{
		getMessageStatusFunc: func(req *message.MessageStatusRequest) (*message.MessageStatusResponse, error) {
			return nil, errors.New("failed to get message status")
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/messages/123", nil)

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: "123"}}

	// Set up the URI parameters
	var messageStatusRequest MessageStatusRequest
	messageStatusRequest.ID = 123
	c.Set("MessageStatusRequest", messageStatusRequest)

	// Call the method
	controller.GetMessageStatus(c)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSendController_RetryFailedMessages(t *testing.T) {
	// Create mock use case
	mockMessageUseCase := &MockMessageUseCase{
		retryFailedMessagesFunc: func() error {
			return nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewSendController(mockCommonService, mockMessageUseCase, logger)

	// Call the method
	controller.RetryFailedMessages()

	// Since this method doesn't return anything, we're just testing that it doesn't panic
	// A more thorough test would use a mock logger to verify that the appropriate log messages were generated
}

func TestSendController_RetryFailedMessages_Error(t *testing.T) {
	// Create mock use case with error
	mockMessageUseCase := &MockMessageUseCase{
		retryFailedMessagesFunc: func() error {
			return errors.New("failed to retry messages")
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create a mock logger that captures log messages
	loggerMock := setupLogger(t)

	// Create controller
	controller := NewSendController(mockCommonService, mockMessageUseCase, loggerMock)

	// Call the method
	controller.RetryFailedMessages()

	// Since this method doesn't return anything and we can't easily check the log output,
	// we're just testing that it doesn't panic
}
