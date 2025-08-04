package signal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-multi-chat-api/src/domain/common"
	"go-multi-chat-api/src/infrastructure/alerting/provider"
	"go-multi-chat-api/src/infrastructure/alerting/provider/email"
	logger "go-multi-chat-api/src/infrastructure/logger"
	domainSignal "go-multi-chat-api/src/infrastructure/repository/signal-client"
	"go-multi-chat-api/src/infrastructure/utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ISignalController interface {
	RegisterNumber(ctx *gin.Context)
	VerifyRegisteredNumber(ctx *gin.Context)
	GetQrCodeLink(ctx *gin.Context)
	Send(c *gin.Context)
}

type SignalController struct {
	signalService *domainSignal.SignalClient
	commonService common.CommonService
	Logger        *logger.Logger
}

func NewSignalController(signalService *domainSignal.SignalClient, commonService common.CommonService, loggerInstance *logger.Logger) ISignalController {
	return &SignalController{signalService: signalService, commonService: commonService, Logger: loggerInstance}
}

func (c *SignalController) RegisterNumber(ctx *gin.Context) {
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
			c.Logger.Error("Couldn't register number: ", zap.Error(err))
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

	err = c.signalService.RegisterNumber(number, req.UseVoice, req.Captcha)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "success"})
}

func (c *SignalController) VerifyRegisteredNumber(ctx *gin.Context) {
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
			c.Logger.Error("Couldn't verify number: ", zap.Error(err))
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

	err = c.signalService.VerifyRegisteredNumber(number, token, pin)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "success"})
}

func (c *SignalController) GetQrCodeLink(ctx *gin.Context) {
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

	png, err := c.signalService.GetQrCodeLink(deviceName, qrCodeVersionInt)
	if err != nil {
		ctx.JSON(400, Error{Msg: err.Error()})
		return
	}

	ctx.Data(200, "image/png", png)

}

func (c *SignalController) Send(ctx *gin.Context) {

	var req SendMessage
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.Logger.Error("Couldn't process request - invalid request", zap.Error(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			c.Logger.Error("Validation errors occurred", zap.Any("errors", ve))
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

	data, err := c.signalService.SendV2(
		req.Number, req.Message, req.Recipients, req.Base64Attachments, req.Sticker,
		req.Mentions, req.QuoteTimestamp, req.QuoteAuthor, req.QuoteMessage, req.QuoteMentions,
		textMode, req.EditTimestamp, req.NotifySelf, req.LinkPreview, req.ViewOnce)
	if err != nil {
		switch err.(type) {
		case *domainSignal.RateLimitErrorType:
			if rateLimitError, ok := err.(*domainSignal.RateLimitErrorType); ok {
				extendedError := errors.New(err.Error() + ". Use the attached challenge tokens to lift the rate limit restrictions via the '/v1/accounts/{number}/rate-limit-challenge' endpoint.")
				ctx.JSON(429, SendMessageError{Msg: extendedError.Error(), ChallengeTokens: rateLimitError.ChallengeTokens, Account: req.Number})
				return
			} else {
				ctx.JSON(400, Error{Msg: err.Error()})
				return
			}
		default:
			ctx.JSON(400, Error{Msg: err.Error()})
			return
		}
		ctx.JSON(400, Error{Msg: err.Error()})
		return
	}

	ctx.JSON(201, SendMessageResponse{Timestamp: strconv.FormatInt((*data)[0].Timestamp, 10)})
}

func createProviderFromCredentials(providerType string, credentials map[string]interface{}) (provider.AlertProvider, error) {
	// Convert credentials to JSON bytes for unmarshaling
	credentialsBytes, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Create the appropriate provider based on the type
	switch providerType {
	case "email":
		var config email.Config
		if err := json.Unmarshal(credentialsBytes, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal email credentials: %w", err)
		}
		return &email.AlertProvider{DefaultConfig: config}, nil
	default:
		return nil, fmt.Errorf("unsupported alert provider type: %s", providerType)
	}
}
