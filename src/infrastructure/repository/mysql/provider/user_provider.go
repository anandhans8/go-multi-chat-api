package provider

import (
	"encoding/json"
	"time"

	domainErrors "go-multi-chat-api/src/domain/errors"
	domainProvider "go-multi-chat-api/src/domain/provider"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserProvider is the database model for user providers
type UserProvider struct {
	ID         int       `gorm:"primaryKey"`
	UserID     int       `gorm:"column:user_id;index"`
	ProviderID int       `gorm:"column:provider_id;index"`
	Priority   int       `gorm:"column:priority"`
	Config     string    `gorm:"column:config;type:text"`
	Status     bool      `gorm:"column:status"`
	CreatedAt  time.Time `gorm:"autoCreateTime:mili"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime:mili"`
}

func (UserProvider) TableName() string {
	return "user_providers"
}

var ColumnsUserProviderMapping = map[string]string{
	"id":         "id",
	"userID":     "user_id",
	"providerID": "provider_id",
	"priority":   "priority",
	"config":     "config",
	"status":     "status",
	"createdAt":  "created_at",
	"updatedAt":  "updated_at",
}

// UserProviderRepositoryInterface defines the interface for user provider repository operations
type UserProviderRepositoryInterface interface {
	GetUserProviders(userID int) (*[]domainProvider.UserProvider, error)
	Create(userProviderDomain *domainProvider.UserProvider) (*domainProvider.UserProvider, error)
	GetByID(id int) (*domainProvider.UserProvider, error)
	Update(id int, userProviderMap map[string]interface{}) (*domainProvider.UserProvider, error)
	Delete(id int) error
	GetUserProvidersByPriority(userID int) (*[]domainProvider.UserProvider, error)
}

type UserProviderRepository struct {
	DB     *gorm.DB
	Logger *logger.Logger
}

func NewUserProviderRepository(db *gorm.DB, loggerInstance *logger.Logger) UserProviderRepositoryInterface {
	return &UserProviderRepository{DB: db, Logger: loggerInstance}
}

func (r *UserProviderRepository) GetUserProviders(userID int) (*[]domainProvider.UserProvider, error) {
	var userProviders []UserProvider
	if err := r.DB.Where("user_id = ?", userID).Find(&userProviders).Error; err != nil {
		r.Logger.Error("Error getting user providers", zap.Error(err), zap.Int("userID", userID))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved user providers", zap.Int("userID", userID), zap.Int("count", len(userProviders)))
	return userProviderArrayToDomainMapper(&userProviders), nil
}

func (r *UserProviderRepository) Create(userProviderDomain *domainProvider.UserProvider) (*domainProvider.UserProvider, error) {
	r.Logger.Info("Creating new user provider", zap.Int("userID", userProviderDomain.UserID), zap.Int("providerID", userProviderDomain.ProviderID))
	userProviderRepository := userProviderFromDomainMapper(userProviderDomain)
	txDb := r.DB.Create(userProviderRepository)
	err := txDb.Error
	if err != nil {
		r.Logger.Error("Error creating user provider", zap.Error(err), zap.Int("userID", userProviderDomain.UserID))
		byteErr, _ := json.Marshal(err)
		var newError domainErrors.GormErr
		errUnmarshal := json.Unmarshal(byteErr, &newError)
		if errUnmarshal != nil {
			return &domainProvider.UserProvider{}, errUnmarshal
		}
		switch newError.Number {
		case 1062:
			err = domainErrors.NewAppErrorWithType(domainErrors.ResourceAlreadyExists)
			return &domainProvider.UserProvider{}, err
		default:
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
	}
	r.Logger.Info("Successfully created user provider", zap.Int("userID", userProviderDomain.UserID), zap.Int("id", userProviderRepository.ID))
	return userProviderRepository.toDomainMapper(), err
}

func (r *UserProviderRepository) GetByID(id int) (*domainProvider.UserProvider, error) {
	var userProvider UserProvider
	err := r.DB.Where("id = ?", id).First(&userProvider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.Logger.Warn("User provider not found", zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.NotFound)
		} else {
			r.Logger.Error("Error getting user provider by ID", zap.Error(err), zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
		return &domainProvider.UserProvider{}, err
	}
	r.Logger.Info("Successfully retrieved user provider by ID", zap.Int("id", id))
	return userProvider.toDomainMapper(), nil
}

func (r *UserProviderRepository) Update(id int, userProviderMap map[string]interface{}) (*domainProvider.UserProvider, error) {
	var userProviderObj UserProvider
	userProviderObj.ID = id

	// Map JSON field names to DB column names
	updateData := make(map[string]interface{})
	for k, v := range userProviderMap {
		if column, ok := ColumnsUserProviderMapping[k]; ok {
			updateData[column] = v
		} else {
			updateData[k] = v
		}
	}

	err := r.DB.Model(&userProviderObj).
		Select("user_id", "provider_id", "priority", "config", "status").
		Updates(updateData).Error
	if err != nil {
		r.Logger.Error("Error updating user provider", zap.Error(err), zap.Int("id", id))
		byteErr, _ := json.Marshal(err)
		var newError domainErrors.GormErr
		errUnmarshal := json.Unmarshal(byteErr, &newError)
		if errUnmarshal != nil {
			return &domainProvider.UserProvider{}, errUnmarshal
		}
		switch newError.Number {
		case 1062:
			return &domainProvider.UserProvider{}, domainErrors.NewAppErrorWithType(domainErrors.ResourceAlreadyExists)
		default:
			return &domainProvider.UserProvider{}, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
	}
	if err := r.DB.Where("id = ?", id).First(&userProviderObj).Error; err != nil {
		r.Logger.Error("Error retrieving updated user provider", zap.Error(err), zap.Int("id", id))
		return &domainProvider.UserProvider{}, err
	}
	r.Logger.Info("Successfully updated user provider", zap.Int("id", id))
	return userProviderObj.toDomainMapper(), nil
}

func (r *UserProviderRepository) Delete(id int) error {
	tx := r.DB.Delete(&UserProvider{}, id)
	if tx.Error != nil {
		r.Logger.Error("Error deleting user provider", zap.Error(tx.Error), zap.Int("id", id))
		return domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	if tx.RowsAffected == 0 {
		r.Logger.Warn("User provider not found for deletion", zap.Int("id", id))
		return domainErrors.NewAppErrorWithType(domainErrors.NotFound)
	}
	r.Logger.Info("Successfully deleted user provider", zap.Int("id", id))
	return nil
}

func (r *UserProviderRepository) GetUserProvidersByPriority(userID int) (*[]domainProvider.UserProvider, error) {
	var userProviders []UserProvider
	if err := r.DB.Where("user_id = ? AND status = ?", userID, true).Order("priority ASC").Find(&userProviders).Error; err != nil {
		r.Logger.Error("Error getting user providers by priority", zap.Error(err), zap.Int("userID", userID))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved user providers by priority", zap.Int("userID", userID), zap.Int("count", len(userProviders)))
	return userProviderArrayToDomainMapper(&userProviders), nil
}

// Mappers
func (up *UserProvider) toDomainMapper() *domainProvider.UserProvider {
	return &domainProvider.UserProvider{
		ID:         up.ID,
		UserID:     up.UserID,
		ProviderID: up.ProviderID,
		Priority:   up.Priority,
		Config:     up.Config,
		Status:     up.Status,
		CreatedAt:  up.CreatedAt,
		UpdatedAt:  up.UpdatedAt,
	}
}

func userProviderFromDomainMapper(up *domainProvider.UserProvider) *UserProvider {
	return &UserProvider{
		ID:         up.ID,
		UserID:     up.UserID,
		ProviderID: up.ProviderID,
		Priority:   up.Priority,
		Config:     up.Config,
		Status:     up.Status,
		CreatedAt:  up.CreatedAt,
		UpdatedAt:  up.UpdatedAt,
	}
}

func userProviderArrayToDomainMapper(userProviders *[]UserProvider) *[]domainProvider.UserProvider {
	userProvidersDomain := make([]domainProvider.UserProvider, len(*userProviders))
	for i, userProvider := range *userProviders {
		userProvidersDomain[i] = *userProvider.toDomainMapper()
	}
	return &userProvidersDomain
}
