package mysql

import (
	"fmt"
	"os"
	"strings"

	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/repository/mysql/provider"
	"go-multi-chat-api/src/infrastructure/repository/mysql/user"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// loadDatabaseConfig loads database configuration from environment variables
// Returns error if any required environment variable is missing
func loadDatabaseConfig() (DatabaseConfig, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")

	// Check for missing required environment variables
	var missingVars []string
	if host == "" {
		missingVars = append(missingVars, "DB_HOST")
	}
	if port == "" {
		missingVars = append(missingVars, "DB_PORT")
	}
	if user == "" {
		missingVars = append(missingVars, "DB_USER")
	}
	if password == "" {
		missingVars = append(missingVars, "DB_PASSWORD")
	}
	if dbName == "" {
		missingVars = append(missingVars, "DB_NAME")
	}
	if sslMode == "" {
		missingVars = append(missingVars, "DB_SSLMODE")
	}

	if len(missingVars) > 0 {
		return DatabaseConfig{}, fmt.Errorf("missing required database environment variables: %s", strings.Join(missingVars, ", "))
	}

	return DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbName,
		SSLMode:  sslMode,
	}, nil
}

type MySQLRepository struct {
	DB     *gorm.DB
	Logger *logger.Logger
	Auth   AuthService
}

type AuthService interface {
	HashPassword(password string) (string, error)
}

func NewRepository(db *gorm.DB, loggerInstance *logger.Logger) *MySQLRepository {
	return &MySQLRepository{
		DB:     db,
		Logger: loggerInstance,
	}
}

func (r *MySQLRepository) SetLogger(loggerInstance *logger.Logger) {
	r.Logger = loggerInstance
}

func (r *MySQLRepository) SetAuthService(auth AuthService) {
	r.Auth = auth
}

func (r *MySQLRepository) LoadDBConfig() (DatabaseConfig, error) {
	return loadDatabaseConfig()
}

func (c DatabaseConfig) GetDSN() string {
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName)
	return connectionString
}

func (r *MySQLRepository) InitDatabase() error {
	cfg, err := loadDatabaseConfig()
	if err != nil {
		r.Logger.Error("Failed to load database configuration", zap.Error(err))
		return fmt.Errorf("failed to load database configuration: %w", err)
	}

	// Create a GORM logger with zap
	gormZap := logger.NewGormLogger(r.Logger.Log).
		LogMode(gormlogger.Warn) // Silent / Error / Warn / Info

	r.DB, err = gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: gormZap,
	})
	if err != nil {
		r.Logger.Error("Error connecting to the database", zap.Error(err))
		return err
	}

	err = r.MigrateEntitiesGORM()
	if err != nil {
		r.Logger.Error("Error migrating the database", zap.Error(err))
		return err
	}

	err = r.SeedInitialUser()
	if err != nil {
		r.Logger.Error("Error seeding initial user", zap.Error(err))
		return err
	}

	r.Logger.Info("Database connection and migrations successful")
	return nil
}

func (r *MySQLRepository) MigrateEntitiesGORM() error {
	// Import the models to register them with GORM
	userModel := &user.User{}

	// Import provider models
	providerModel := &provider.Provider{}
	userProviderModel := &provider.UserProvider{}
	messageTransactionModel := &provider.MessageTransaction{}
	messageTransactionHistoryModel := &provider.MessageTransactionHistory{}

	// Auto migrate the models to create/update tables
	err := r.DB.AutoMigrate(
		userModel,
		providerModel,
		userProviderModel,
		messageTransactionModel,
		messageTransactionHistoryModel,
	)
	if err != nil {
		r.Logger.Error("Error migrating database entities", zap.Error(err))
		return err
	}

	r.Logger.Info("Database entities migration completed successfully")
	return nil
}

func (r *MySQLRepository) SeedInitialUser() error {
	email := os.Getenv("START_USER_EMAIL")
	pw := os.Getenv("START_USER_PW")
	if email == "" || pw == "" {
		r.Logger.Info("Initial user seed skipped: START_USER_EMAIL or START_USER_PW not set")
		return nil
	}

	// Check if a user already exists
	var existingUser user.User
	err := r.DB.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		r.Logger.Info("Initial user already exists, skipping seed", zap.String("email", email))
		return nil
	}

	// Create an initial user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		r.Logger.Error("Error hashing password for initial user", zap.Error(err))
		return err
	}

	newUser := user.User{
		Email:            email,
		HashPassword:     string(hashedPassword),
		Role:             "admin",
		UserName:         "admin",
		Status:           true,
		MessageRateLimit: 1000, // Example rate limit
	}

	err = r.DB.Create(&newUser).Error
	if err != nil {
		r.Logger.Error("Error creating initial user", zap.Error(err))
		return err
	}

	r.Logger.Info("Initial user created successfully", zap.String("email", email))

	// Create default providers for the initial user
	defaultProviders := []provider.Provider{
		{Name: "Signal", Type: "signal", Status: true, Description: "Signal is a free and open-source messaging app for Android and iOS."},
		{Name: "Teams", Type: "teams", Status: true, Description: "Microsoft Teams is a collaboration app that helps your team stay organized and has conversations all in one place."},
		{Name: "Sms", Type: "sms", Status: true, Description: "SMS is a text messaging service component of most telephone, internet, and mobile device systems."},
		{Name: "Email", Type: "email", Status: true, Description: "Email is a method of exchanging digital messages between people using electronic devices."},
	}

	for _, providerData := range defaultProviders {
		var (
			// Check if the provider already exists
			existingProvider provider.Provider
		)
		err = r.DB.Where("name = ?", providerData.Name).First(&existingProvider).Error
		err = r.DB.Create(&providerData).Error
		if err != nil {
			r.Logger.Error("Error creating default provider", zap.Error(err), zap.String("provider", providerData.Name))
			return err
		}
		r.Logger.Info("Default provider created successfully", zap.String("provider", providerData.Name))
	}

	var signalProvider provider.Provider
	// Check if Signal provider exists
	err = r.DB.Where("name = ?", "Signal").First(&signalProvider).Error
	if err != nil {
		r.Logger.Error("Signal provider not found", zap.Error(err))
		return err
	}

	// Create user-provider associations
	userProvider := provider.UserProvider{
		UserID:     newUser.ID,
		ProviderID: signalProvider.ID,
		Status:     true,
		Priority:   1, // Default priority
	}
	err = r.DB.Create(&userProvider).Error
	if err != nil {
		r.Logger.Error("Error creating user-provider association", zap.Error(err), zap.String("provider", signalProvider.Name))
		return err
	}
	return nil
}

// InitMySQLDB initializes the database connection with logger
func InitMySQLDB(loggerInstance *logger.Logger) (*gorm.DB, error) {
	repo := &MySQLRepository{
		Logger: loggerInstance,
	}

	err := repo.InitDatabase()
	if err != nil {
		return nil, err
	}

	return repo.DB, nil
}
