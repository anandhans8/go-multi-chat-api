package di

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go-multi-chat-api/src/domain/common"
	"go-multi-chat-api/src/infrastructure/helper"
	"go-multi-chat-api/src/infrastructure/messaging"
	"go-multi-chat-api/src/infrastructure/utils"
	"log"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"

	authUseCase "go-multi-chat-api/src/application/usecases/auth"
	messageUseCase "go-multi-chat-api/src/application/usecases/message"
	userUseCase "go-multi-chat-api/src/application/usecases/user"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/repository/mysql"
	providerRepo "go-multi-chat-api/src/infrastructure/repository/mysql/provider"
	"go-multi-chat-api/src/infrastructure/repository/mysql/user"
	signalClient "go-multi-chat-api/src/infrastructure/repository/signal-client"
	authController "go-multi-chat-api/src/infrastructure/rest/controllers/auth"
	sendController "go-multi-chat-api/src/infrastructure/rest/controllers/send"
	signalController "go-multi-chat-api/src/infrastructure/rest/controllers/signal"
	userController "go-multi-chat-api/src/infrastructure/rest/controllers/user"
	"go-multi-chat-api/src/infrastructure/security"

	"gorm.io/gorm"
)

// ApplicationContext holds all application dependencies and services
type ApplicationContext struct {
	DB                                  *gorm.DB
	Logger                              *logger.Logger
	AuthController                      authController.IAuthController
	UserController                      userController.IUserController
	SignalController                    signalController.ISignalController
	SendController                      sendController.ISendController
	JWTService                          security.IJWTService
	LDAPService                         security.ILDAPService
	AzureADService                      security.IAzureADService
	CommonService                       common.CommonService
	UserRepository                      user.UserRepositoryInterface
	AuthUseCase                         authUseCase.IAuthUseCase
	UserUseCase                         userUseCase.IUserUseCase
	MessageProcessor                    *messaging.MessageProcessor
	ProviderRepository                  providerRepo.ProviderRepositoryInterface
	UserProviderRepository              providerRepo.UserProviderRepositoryInterface
	MessageTransactionRepository        providerRepo.MessageTransactionRepositoryInterface
	MessageTransactionHistoryRepository providerRepo.MessageTransactionHistoryRepositoryInterface
}

var (
	loggerInstance *logger.Logger
	loggerOnce     sync.Once
)

func GetLogger() *logger.Logger {
	loggerOnce.Do(func() {
		loggerInstance, _ = logger.NewLogger()
	})
	return loggerInstance
}

// SetupDependencies creates a new application context with all dependencies
func SetupDependencies(loggerInstance *logger.Logger) (*ApplicationContext, error) {
	// Initialize database with logger
	db, err := mysql.InitMySQLDB(loggerInstance)
	if err != nil {
		return nil, err
	}

	// Initialize signal-cli configuration
	signalCliConfigDir := "/home/.local/share/signal-cli/"
	signalCliConfigDirEnv := utils.GetEnv("SIGNAL_CLI_CONFIG_DIR", "")
	if signalCliConfigDirEnv != "" {
		signalCliConfigDir = signalCliConfigDirEnv
		if !strings.HasSuffix(signalCliConfigDirEnv, "/") {
			signalCliConfigDir += "/"
		}
	}

	signalCliConfig := flag.String("signal-cli-config", signalCliConfigDir, "Config directory where signal-cli config is stored")
	attachmentTmpDir := flag.String("attachment-tmp-dir", "/tmp/", "Attachment tmp directory")
	avatarTmpDir := flag.String("avatar-tmp-dir", "/tmp/", "Avatar tmp directory")
	flag.Parse()

	supportsSignalCliNative := "0"
	if _, err := os.Stat("/usr/bin/signal-cli-native"); err == nil {
		supportsSignalCliNative = "1"
	}

	err = os.Setenv("SUPPORTS_NATIVE", supportsSignalCliNative)
	if err != nil {
		loggerInstance.Fatal("Couldn't set env variable: ", zap.Error(err))
	}

	useNative := utils.GetEnv("USE_NATIVE", "")
	if useNative != "" {
		loggerInstance.Warn("The env variable USE_NATIVE is deprecated. Please use the env variable MODE instead")
	}

	signalCliMode := signalClient.Normal
	mode := utils.GetEnv("SIGNAL_MODE", "normal")
	if mode == "normal" {
		signalCliMode = signalClient.Normal
	} else if mode == "json-rpc" {
		signalCliMode = signalClient.JsonRpc
	} else if mode == "native" {
		signalCliMode = signalClient.Native
	}

	if useNative != "" {
		_, modeEnvVariableSet := os.LookupEnv("MODE")
		if modeEnvVariableSet {
			loggerInstance.Fatal("You have both the USE_NATIVE and the MODE env variable set. Please remove the deprecated env variable USE_NATIVE!")
		}
	}

	if useNative == "1" || signalCliMode == signalClient.Native {
		if supportsSignalCliNative == "0" {
			loggerInstance.Error("signal-cli-native is not support on this system...falling back to signal-cli")
			signalCliMode = signalClient.Normal
		}
	}

	if signalCliMode == signalClient.JsonRpc {
		_, autoReceiveScheduleEnvVariableSet := os.LookupEnv("AUTO_RECEIVE_SCHEDULE")
		if autoReceiveScheduleEnvVariableSet {
			loggerInstance.Fatal("Env variable AUTO_RECEIVE_SCHEDULE can't be used with mode json-rpc")
		}

		_, signalCliCommandTimeoutEnvVariableSet := os.LookupEnv("SIGNAL_CLI_CMD_TIMEOUT")
		if signalCliCommandTimeoutEnvVariableSet {
			loggerInstance.Fatal("Env variable SIGNAL_CLI_CMD_TIMEOUT can't be used with mode json-rpc")
		}
	}

	webhookUrl := utils.GetEnv("RECEIVE_WEBHOOK_URL", "")
	if webhookUrl != "" && signalCliMode != signalClient.JsonRpc {
		log.Fatal("Env variable RECEIVE_WEBHOOK_URL can't be used with mode json-rpc!")
	}

	jsonRpc2ClientConfigPathPath := *signalCliConfig + "/jsonrpc2.yml"
	signalCliApiConfigPath := *signalCliConfig + "/api-config.yml"

	// Create the signal client directly for backward compatibility
	signalClientInstance := signalClient.NewSignalClient(*signalCliConfig, *attachmentTmpDir, *avatarTmpDir, signalCliMode, jsonRpc2ClientConfigPathPath, signalCliApiConfigPath, webhookUrl, loggerInstance)
	err = signalClientInstance.Init()
	if err != nil {
		log.Fatal("Couldn't init Signal Client: ", err.Error())
	}

	// Initialize JWT service (manages its own configuration)
	jwtService := security.NewJWTService()

	// Initialize LDAP service with configuration from environment variables
	ldapEnabled := utils.GetEnv("LDAP_ENABLED", "false") == "true"
	ldapConfig := security.LDAPConfig{
		URL:          utils.GetEnv("LDAP_URL", ""),
		BindDN:       utils.GetEnv("LDAP_BIND_DN", ""),
		BindPassword: utils.GetEnv("LDAP_BIND_PASSWORD", ""),
		BaseDN:       utils.GetEnv("LDAP_BASE_DN", ""),
		UserFilter:   utils.GetEnv("LDAP_USER_FILTER", "(uid=%s)"),
		Enabled:      ldapEnabled,
		TLSEnabled:   utils.GetEnv("LDAP_TLS_ENABLED", "false") == "true",
		Attributes:   strings.Split(utils.GetEnv("LDAP_ATTRIBUTES", "uid,mail,givenName,sn"), ","),
	}
	ldapService := security.NewLDAPService(ldapConfig, loggerInstance)
	loggerInstance.Info("LDAP authentication " + map[bool]string{true: "enabled", false: "disabled"}[ldapEnabled])

	// Initialize Azure AD service with configuration from environment variables
	azureADEnabled := utils.GetEnv("AZURE_AD_ENABLED", "false") == "true"
	azureADConfig := security.AzureADConfig{
		TenantID:     utils.GetEnv("AZURE_AD_TENANT_ID", ""),
		ClientID:     utils.GetEnv("AZURE_AD_CLIENT_ID", ""),
		ClientSecret: utils.GetEnv("AZURE_AD_CLIENT_SECRET", ""),
		RedirectURI:  utils.GetEnv("AZURE_AD_REDIRECT_URI", ""),
		Scopes:       strings.Split(utils.GetEnv("AZURE_AD_SCOPES", "openid,profile,email"), ","),
		Enabled:      azureADEnabled,
	}
	azureADService := security.NewAzureADService(azureADConfig, loggerInstance)
	loggerInstance.Info("Azure AD authentication " + map[bool]string{true: "enabled", false: "disabled"}[azureADEnabled])

	validator := helper.NewValidator(loggerInstance)
	commonService := common.NewCommonService(validator)

	// Initialize repositories with logger
	userRepo := user.NewUserRepository(db, loggerInstance)
	providerRepository := providerRepo.NewProviderRepository(db, loggerInstance)
	userProviderRepository := providerRepo.NewUserProviderRepository(db, loggerInstance)
	messageTransactionRepository := providerRepo.NewMessageTransactionRepository(db, loggerInstance)
	messageTransactionHistoryRepository := providerRepo.NewMessageTransactionHistoryRepository(db, loggerInstance)

	// Initialize use cases with logger
	authUC := authUseCase.NewAuthUseCase(userRepo, jwtService, ldapService, azureADService, loggerInstance)
	userUC := userUseCase.NewUserUseCase(userRepo, loggerInstance)

	// Create message processor with 100 worker goroutines
	messageProcessor := messaging.NewMessageProcessor(
		signalClientInstance,
		providerRepository,
		userProviderRepository,
		messageTransactionRepository,
		messageTransactionHistoryRepository,
		loggerInstance,
		100, // 100 worker goroutines
	)

	// Initialize message use case
	messageUC := messageUseCase.NewMessageUseCase(
		providerRepository,
		userProviderRepository,
		messageTransactionRepository,
		messageProcessor,
		userRepo,
		loggerInstance,
	)

	// Initialize controllers with logger
	authController := authController.NewAuthController(authUC, loggerInstance)
	userController := userController.NewUserController(userUC, loggerInstance)
	signalClientController := signalController.NewSignalController(signalClientInstance, commonService, loggerInstance)
	sendController := sendController.NewSendController(
		commonService,
		messageUC,
		loggerInstance,
	)

	var wsMutex sync.Mutex
	var stopSignalReceive = make(chan struct{})
	go handleSignalReceive(signalClientInstance, os.Getenv("SIGNAL_FROM_NUMBER"), stopSignalReceive, &wsMutex, loggerInstance)

	return &ApplicationContext{
		DB:                                  db,
		Logger:                              loggerInstance,
		AuthController:                      authController,
		UserController:                      userController,
		SignalController:                    signalClientController,
		SendController:                      sendController,
		JWTService:                          jwtService,
		LDAPService:                         ldapService,
		AzureADService:                      azureADService,
		CommonService:                       commonService,
		UserRepository:                      userRepo,
		AuthUseCase:                         authUC,
		UserUseCase:                         userUC,
		MessageProcessor:                    messageProcessor,
		ProviderRepository:                  providerRepository,
		UserProviderRepository:              userProviderRepository,
		MessageTransactionRepository:        messageTransactionRepository,
		MessageTransactionHistoryRepository: messageTransactionHistoryRepository,
	}, nil
}

func handleSignalReceive(signalClient *signalClient.SignalClient, number string, stop chan struct{}, wsMutex *sync.Mutex, loggerInstance *logger.Logger) {
	receiveChannel, channelUuid, err := signalClient.GetReceiveChannel()
	if err != nil {
		loggerInstance.Error("Couldn't get receive channel: ", zap.Error(err))
		return
	}

	for {
		select {
		case <-stop:
			signalClient.RemoveReceiveChannel(channelUuid)
			return
		case msg := <-receiveChannel:
			var data string = string(msg.Params)
			var err error = nil
			if msg.Err.Code != 0 {
				err = errors.New(msg.Err.Message)
			}

			if err == nil {
				if data != "" {
					type Response struct {
						Account string `json:"account"`
					}
					var response Response
					err = json.Unmarshal([]byte(data), &response)
					if err != nil {
						loggerInstance.Error(fmt.Sprintf("Couldn't parse message %s", data), zap.Error(err))
						continue
					}

					if response.Account == number {
						wsMutex.Lock()
						loggerInstance.Debug("Received message from self: " + data)
						wsMutex.Unlock()
					}
				}
			} else {
				wsMutex.Lock()
				loggerInstance.Error(fmt.Sprintf("Received error message: %s", data), zap.Error(err))
				wsMutex.Unlock()
			}
		}
	}
}

// NewTestApplicationContext creates an application context for testing with mocked dependencies
func NewTestApplicationContext(
	mockUserRepo user.UserRepositoryInterface,
	mockJWTService security.IJWTService,
	mockLDAPService security.ILDAPService,
	mockAzureADService security.IAzureADService,
	loggerInstance *logger.Logger,
) *ApplicationContext {
	// Initialize use cases with mocked repositories and logger
	authUC := authUseCase.NewAuthUseCase(mockUserRepo, mockJWTService, mockLDAPService, mockAzureADService, loggerInstance)
	userUC := userUseCase.NewUserUseCase(mockUserRepo, loggerInstance)

	// Initialize controllers with logger
	authController := authController.NewAuthController(authUC, loggerInstance)
	userController := userController.NewUserController(userUC, loggerInstance)

	return &ApplicationContext{
		AuthController: authController,
		UserController: userController,
		JWTService:     mockJWTService,
		LDAPService:    mockLDAPService,
		AzureADService: mockAzureADService,
		UserRepository: mockUserRepo,
		AuthUseCase:    authUC,
		UserUseCase:    userUC,
	}
}
