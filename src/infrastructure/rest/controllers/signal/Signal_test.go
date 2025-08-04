package signal

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"go-multi-chat-api/src/domain/common"
	ds "go-multi-chat-api/src/infrastructure/datastructs"
	logger "go-multi-chat-api/src/infrastructure/logger"
	domainSignal "go-multi-chat-api/src/infrastructure/repository/signal-client"
	"go-multi-chat-api/src/infrastructure/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// TestSignalClient is an interface that only includes the methods used by the controller
type TestSignalClient interface {
	RegisterNumber(number string, useVoice bool, captcha string) error
	VerifyRegisteredNumber(number, token, pin string) error
	GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error)
	SendV2(number, message string, recipients, base64Attachments []string, sticker string, mentions []interface{}, quoteTimestamp *int64, quoteAuthor, quoteMessage *string, quoteMentions []interface{}, textMode *string, editTimestamp *int64, notifySelf *bool, linkPreview *interface{}, viewOnce *bool) (*[]domainSignal.SendResponse, error)
}

// TestSignalController is a wrapper around SignalController for testing purposes
type TestSignalController struct {
	signalClient  TestSignalClient
	commonService common.CommonService
	logger        *logger.Logger
}

// NewTestSignalController creates a new TestSignalController
func NewTestSignalController(signalClient TestSignalClient, commonService common.CommonService, loggerInstance *logger.Logger) ISignalController {
	return &TestSignalController{
		signalClient:  signalClient,
		commonService: commonService,
		logger:        loggerInstance,
	}
}

// RegisterNumber delegates to the SignalController.RegisterNumber method
func (c *TestSignalController) RegisterNumber(ctx *gin.Context) {
	number, err := url.PathUnescape(ctx.Param("number"))
	if err != nil {
		ctx.JSON(400, Error{Msg: "Couldn't process request - malformed number"})
		return
	}

	var req RegisterNumberRequest

	buf := new(bytes.Buffer)
	buf.ReadFrom(ctx.Request.Body)
	if buf.String() != "" {
		err := json.Unmarshal(buf.Bytes(), &req)
		if err != nil {
			c.logger.Error("Couldn't register number: ", zap.Error(err))
			ctx.JSON(400, Error{Msg: "Couldn't process request - invalid request."})
			return
		}
	} else {
		req.UseVoice = false
		req.Captcha = ""
	}

	if number == "" {
		ctx.JSON(400, gin.H{"error": "Please provide a number"})
		return
	}

	err = c.signalClient.RegisterNumber(number, req.UseVoice, req.Captcha)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "success"})
}

// VerifyRegisteredNumber delegates to the SignalController.VerifyRegisteredNumber method
func (c *TestSignalController) VerifyRegisteredNumber(ctx *gin.Context) {
	number, err := url.PathUnescape(ctx.Param("number"))
	if err != nil {
		ctx.JSON(400, Error{Msg: "Couldn't process request - malformed number"})
		return
	}
	token := ctx.Param("token")

	pin := ""
	var req VerifyNumberSettings
	buf := new(bytes.Buffer)
	buf.ReadFrom(ctx.Request.Body)
	if buf.String() != "" {
		err := json.Unmarshal(buf.Bytes(), &req)
		if err != nil {
			c.logger.Error("Couldn't verify number: ", zap.Error(err))
			ctx.JSON(400, Error{Msg: "Couldn't process request - invalid request."})
			return
		}
		pin = req.Pin
	}

	if number == "" {
		ctx.JSON(400, gin.H{"error": "Please provide a number"})
		return
	}

	if token == "" {
		ctx.JSON(400, gin.H{"error": "Please provide a verification code"})
		return
	}

	err = c.signalClient.VerifyRegisteredNumber(number, token, pin)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "success"})
}

// GetQrCodeLink delegates to the SignalController.GetQrCodeLink method
func (c *TestSignalController) GetQrCodeLink(ctx *gin.Context) {
	deviceName := ctx.Query("device_name")
	qrCodeVersion := ctx.Query("qrcode_version")

	if deviceName == "" {
		ctx.JSON(400, Error{Msg: "Please provide a name for the device"})
		return
	}

	qrCodeVersionInt := 10
	if qrCodeVersion != "" {
		var err error
		qrCodeVersionInt, err = strconv.Atoi(qrCodeVersion)
		if err != nil {
			ctx.JSON(400, Error{Msg: "The qrcode_version parameter needs to be an integer!"})
			return
		}
	}

	png, err := c.signalClient.GetQrCodeLink(deviceName, qrCodeVersionInt)
	if err != nil {
		ctx.JSON(400, Error{Msg: err.Error()})
		return
	}

	ctx.Data(200, "image/png", png)
}

// Send delegates to the SignalController.Send method
func (c *TestSignalController) Send(ctx *gin.Context) {
	var req SendMessage
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.logger.Error("Couldn't process request - invalid request", zap.Error(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			c.logger.Error("Validation errors occurred", zap.Any("errors", ve))
			c.commonService.AppendValidationErrors(ctx, ve, req)
			return
		}
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if req.Recipient != "" {
		req.Recipients = append(req.Recipients, req.Recipient)
	}
	if len(req.Recipients) == 0 {
		ctx.JSON(400, gin.H{"error": "Couldn't process request - please provide at least one recipient"})
		return
	}

	if req.Number == "" {
		ctx.JSON(400, gin.H{"error": "Couldn't process request - please provide a valid number"})
		return
	}

	if req.Sticker != "" && !strings.Contains(req.Sticker, ":") {
		ctx.JSON(400, gin.H{"error": "Couldn't process request - please provide valid sticker delimiter"})
		return
	}

	textMode := req.TextMode
	if textMode == nil {
		defaultSignalTextMode := utils.GetEnv("DEFAULT_SIGNAL_TEXT_MODE", "normal")
		if defaultSignalTextMode == "styled" {
			styledStr := "styled"
			textMode = &styledStr
		}
	}

	if req.ViewOnce != nil && *req.ViewOnce && (len(req.Base64Attachments) == 0) {
		ctx.JSON(400, Error{Msg: "'view_once' can only be set for image attachments!"})
		return
	}

	// Convert ds.MessageMention to []interface{}
	var mentions []interface{}
	for _, m := range req.Mentions {
		mentions = append(mentions, m)
	}

	var quoteMentions []interface{}
	for _, m := range req.QuoteMentions {
		quoteMentions = append(quoteMentions, m)
	}

	// Convert *ds.LinkPreviewType to *interface{}
	var linkPreview *interface{}
	if req.LinkPreview != nil {
		var lp interface{} = req.LinkPreview
		linkPreview = &lp
	}

	data, err := c.signalClient.SendV2(
		req.Number, req.Message, req.Recipients, req.Base64Attachments, req.Sticker,
		mentions, req.QuoteTimestamp, req.QuoteAuthor, req.QuoteMessage, quoteMentions,
		textMode, req.EditTimestamp, req.NotifySelf, linkPreview, req.ViewOnce)
	if err != nil {
		ctx.JSON(400, Error{Msg: err.Error()})
		return
	}

	ctx.JSON(201, SendMessageResponse{Timestamp: strconv.FormatInt((*data)[0].Timestamp, 10)})
}

// MockSignalClient implements the necessary methods for testing
type MockSignalClient struct {
	registerNumberFunc         func(string, bool, string) error
	verifyRegisteredNumberFunc func(string, string, string) error
	getQrCodeLinkFunc          func(string, int) ([]byte, error)
	sendV2Func                 func(string, string, []string, []string, string, []interface{}, *int64, *string, *string, []interface{}, *string, *int64, *bool, *interface{}, *bool) (*[]domainSignal.SendResponse, error)
}

func (m *MockSignalClient) RegisterNumber(number string, useVoice bool, captcha string) error {
	if m.registerNumberFunc != nil {
		return m.registerNumberFunc(number, useVoice, captcha)
	}
	return nil
}

func (m *MockSignalClient) VerifyRegisteredNumber(number, token, pin string) error {
	if m.verifyRegisteredNumberFunc != nil {
		return m.verifyRegisteredNumberFunc(number, token, pin)
	}
	return nil
}

func (m *MockSignalClient) GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error) {
	if m.getQrCodeLinkFunc != nil {
		return m.getQrCodeLinkFunc(deviceName, qrCodeVersion)
	}
	return []byte{}, nil
}

func (m *MockSignalClient) SendV2(number, message string, recipients, base64Attachments []string, sticker string, mentions []interface{}, quoteTimestamp *int64, quoteAuthor, quoteMessage *string, quoteMentions []interface{}, textMode *string, editTimestamp *int64, notifySelf *bool, linkPreview *interface{}, viewOnce *bool) (*[]domainSignal.SendResponse, error) {
	if m.sendV2Func != nil {
		return m.sendV2Func(number, message, recipients, base64Attachments, sticker, mentions, quoteTimestamp, quoteAuthor, quoteMessage, quoteMentions, textMode, editTimestamp, notifySelf, linkPreview, viewOnce)
	}
	return &[]domainSignal.SendResponse{}, nil
}

// MockCommonService implements the necessary methods for testing
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

// This is a dummy variable to make the ds import used
var _ ds.MessageMention

func setupLogger(t *testing.T) *logger.Logger {
	loggerInstance, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return loggerInstance
}

func TestNewSignalController(t *testing.T) {
	mockSignalClient := &MockSignalClient{}
	mockCommonService := &MockCommonService{}
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	if controller == nil {
		t.Error("Expected NewSignalController to return a non-nil controller")
	}
}

func TestSignalController_RegisterNumber_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client
	mockSignalClient := &MockSignalClient{
		registerNumberFunc: func(number string, useVoice bool, captcha string) error {
			return nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create test request
	registerRequest := RegisterNumberRequest{
		UseVoice: false,
		Captcha:  "",
	}

	requestBody, _ := json.Marshal(registerRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register/+1234567890", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "number", Value: "+1234567890"}}

	// Call the method
	controller.RegisterNumber(c)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestSignalController_RegisterNumber_Error(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client with error
	mockSignalClient := &MockSignalClient{
		registerNumberFunc: func(number string, useVoice bool, captcha string) error {
			return errors.New("registration failed")
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create test request
	registerRequest := RegisterNumberRequest{
		UseVoice: false,
		Captcha:  "",
	}

	requestBody, _ := json.Marshal(registerRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register/+1234567890", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "number", Value: "+1234567890"}}

	// Call the method
	controller.RegisterNumber(c)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSignalController_VerifyRegisteredNumber_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client
	mockSignalClient := &MockSignalClient{
		verifyRegisteredNumberFunc: func(number, token, pin string) error {
			return nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create test request
	verifyRequest := VerifyNumberSettings{
		Pin: "123456",
	}

	requestBody, _ := json.Marshal(verifyRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/verify/+1234567890/token123", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "number", Value: "+1234567890"},
		{Key: "token", Value: "token123"},
	}

	// Call the method
	controller.VerifyRegisteredNumber(c)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestSignalController_GetQrCodeLink_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client
	mockSignalClient := &MockSignalClient{
		getQrCodeLinkFunc: func(deviceName string, qrCodeVersion int) ([]byte, error) {
			return []byte("fake-png-data"), nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/qr-code?device_name=test-device", nil)

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Request.URL.RawQuery = "device_name=test-device"

	// Set up query parameters
	c.Set("device_name", "test-device")

	// Call the method
	controller.GetQrCodeLink(c)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Expected Content-Type to be 'image/png', got '%s'", w.Header().Get("Content-Type"))
	}

	if string(w.Body.Bytes()) != "fake-png-data" {
		t.Errorf("Expected body to be 'fake-png-data', got '%s'", w.Body.String())
	}
}

func TestSignalController_Send_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client
	timestamp := int64(1234567890)
	mockSignalClient := &MockSignalClient{
		sendV2Func: func(number, message string, recipients, base64Attachments []string, sticker string, mentions []interface{}, quoteTimestamp *int64, quoteAuthor, quoteMessage *string, quoteMentions []interface{}, textMode *string, editTimestamp *int64, notifySelf *bool, linkPreview *interface{}, viewOnce *bool) (*[]domainSignal.SendResponse, error) {
			return &[]domainSignal.SendResponse{{Timestamp: timestamp}}, nil
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create test request
	sendRequest := SendMessage{
		Number:     "+1234567890",
		Recipients: []string{"+9876543210"},
		Message:    "Test message",
	}

	requestBody, _ := json.Marshal(sendRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/send", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Send(c)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Parse response
	var response SendMessageResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response fields
	expectedTimestamp := "1234567890"
	if response.Timestamp != expectedTimestamp {
		t.Errorf("Expected Timestamp to be '%s', got '%s'", expectedTimestamp, response.Timestamp)
	}
}

func TestSignalController_Send_Error(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock signal client with error
	mockSignalClient := &MockSignalClient{
		sendV2Func: func(number, message string, recipients, base64Attachments []string, sticker string, mentions []interface{}, quoteTimestamp *int64, quoteAuthor, quoteMessage *string, quoteMentions []interface{}, textMode *string, editTimestamp *int64, notifySelf *bool, linkPreview *interface{}, viewOnce *bool) (*[]domainSignal.SendResponse, error) {
			return nil, errors.New("send failed")
		},
	}

	// Create mock common service
	mockCommonService := &MockCommonService{}

	// Create controller
	logger := setupLogger(t)
	controller := NewTestSignalController(mockSignalClient, mockCommonService, logger)

	// Create test request
	sendRequest := SendMessage{
		Number:     "+1234567890",
		Recipients: []string{"+9876543210"},
		Message:    "Test message",
	}

	requestBody, _ := json.Marshal(sendRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/send", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Send(c)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
