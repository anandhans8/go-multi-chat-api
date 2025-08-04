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

// Provider is the database model for providers
type Provider struct {
	ID          int       `gorm:"primaryKey"`
	Name        string    `gorm:"unique"`
	Type        string    `gorm:"column:type"`
	Description string    `gorm:"column:description"`
	Config      string    `gorm:"column:config;type:text"`
	Status      bool      `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"autoCreateTime:mili"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime:mili"`
}

func (Provider) TableName() string {
	return "providers"
}

var ColumnsProviderMapping = map[string]string{
	"id":          "id",
	"name":        "name",
	"type":        "type",
	"description": "description",
	"config":      "config",
	"status":      "status",
	"createdAt":   "created_at",
	"updatedAt":   "updated_at",
}

// ProviderRepositoryInterface defines the interface for provider repository operations
type ProviderRepositoryInterface interface {
	GetAll() (*[]domainProvider.Provider, error)
	Create(providerDomain *domainProvider.Provider) (*domainProvider.Provider, error)
	GetByID(id int) (*domainProvider.Provider, error)
	Update(id int, providerMap map[string]interface{}) (*domainProvider.Provider, error)
	Delete(id int) error
}

type Repository struct {
	DB     *gorm.DB
	Logger *logger.Logger
}

func NewProviderRepository(db *gorm.DB, loggerInstance *logger.Logger) ProviderRepositoryInterface {
	return &Repository{DB: db, Logger: loggerInstance}
}

func (r *Repository) GetAll() (*[]domainProvider.Provider, error) {
	var providers []Provider
	if err := r.DB.Find(&providers).Error; err != nil {
		r.Logger.Error("Error getting all providers", zap.Error(err))
		return nil, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	r.Logger.Info("Successfully retrieved all providers", zap.Int("count", len(providers)))
	return arrayToDomainMapper(&providers), nil
}

func (r *Repository) Create(providerDomain *domainProvider.Provider) (*domainProvider.Provider, error) {
	r.Logger.Info("Creating new provider", zap.String("name", providerDomain.Name))
	providerRepository := fromDomainMapper(providerDomain)
	txDb := r.DB.Create(providerRepository)
	err := txDb.Error
	if err != nil {
		r.Logger.Error("Error creating provider", zap.Error(err), zap.String("name", providerDomain.Name))
		byteErr, _ := json.Marshal(err)
		var newError domainErrors.GormErr
		errUnmarshal := json.Unmarshal(byteErr, &newError)
		if errUnmarshal != nil {
			return &domainProvider.Provider{}, errUnmarshal
		}
		switch newError.Number {
		case 1062:
			err = domainErrors.NewAppErrorWithType(domainErrors.ResourceAlreadyExists)
			return &domainProvider.Provider{}, err
		default:
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
	}
	r.Logger.Info("Successfully created provider", zap.String("name", providerDomain.Name), zap.Int("id", providerRepository.ID))
	return providerRepository.toDomainMapper(), err
}

func (r *Repository) GetByID(id int) (*domainProvider.Provider, error) {
	var provider Provider
	err := r.DB.Where("id = ?", id).First(&provider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.Logger.Warn("Provider not found", zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.NotFound)
		} else {
			r.Logger.Error("Error getting provider by ID", zap.Error(err), zap.Int("id", id))
			err = domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
		return &domainProvider.Provider{}, err
	}
	r.Logger.Info("Successfully retrieved provider by ID", zap.Int("id", id))
	return provider.toDomainMapper(), nil
}

func (r *Repository) Update(id int, providerMap map[string]interface{}) (*domainProvider.Provider, error) {
	var providerObj Provider
	providerObj.ID = id

	// Map JSON field names to DB column names
	updateData := make(map[string]interface{})
	for k, v := range providerMap {
		if column, ok := ColumnsProviderMapping[k]; ok {
			updateData[column] = v
		} else {
			updateData[k] = v
		}
	}

	err := r.DB.Model(&providerObj).
		Select("name", "type", "description", "config", "status").
		Updates(updateData).Error
	if err != nil {
		r.Logger.Error("Error updating provider", zap.Error(err), zap.Int("id", id))
		byteErr, _ := json.Marshal(err)
		var newError domainErrors.GormErr
		errUnmarshal := json.Unmarshal(byteErr, &newError)
		if errUnmarshal != nil {
			return &domainProvider.Provider{}, errUnmarshal
		}
		switch newError.Number {
		case 1062:
			return &domainProvider.Provider{}, domainErrors.NewAppErrorWithType(domainErrors.ResourceAlreadyExists)
		default:
			return &domainProvider.Provider{}, domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
		}
	}
	if err := r.DB.Where("id = ?", id).First(&providerObj).Error; err != nil {
		r.Logger.Error("Error retrieving updated provider", zap.Error(err), zap.Int("id", id))
		return &domainProvider.Provider{}, err
	}
	r.Logger.Info("Successfully updated provider", zap.Int("id", id))
	return providerObj.toDomainMapper(), nil
}

func (r *Repository) Delete(id int) error {
	tx := r.DB.Delete(&Provider{}, id)
	if tx.Error != nil {
		r.Logger.Error("Error deleting provider", zap.Error(tx.Error), zap.Int("id", id))
		return domainErrors.NewAppErrorWithType(domainErrors.UnknownError)
	}
	if tx.RowsAffected == 0 {
		r.Logger.Warn("Provider not found for deletion", zap.Int("id", id))
		return domainErrors.NewAppErrorWithType(domainErrors.NotFound)
	}
	r.Logger.Info("Successfully deleted provider", zap.Int("id", id))
	return nil
}

// Mappers
func (p *Provider) toDomainMapper() *domainProvider.Provider {
	return &domainProvider.Provider{
		ID:          p.ID,
		Name:        p.Name,
		Type:        p.Type,
		Description: p.Description,
		Config:      p.Config,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func fromDomainMapper(p *domainProvider.Provider) *Provider {
	return &Provider{
		ID:          p.ID,
		Name:        p.Name,
		Type:        p.Type,
		Description: p.Description,
		Config:      p.Config,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func arrayToDomainMapper(providers *[]Provider) *[]domainProvider.Provider {
	providersDomain := make([]domainProvider.Provider, len(*providers))
	for i, provider := range *providers {
		providersDomain[i] = *provider.toDomainMapper()
	}
	return &providersDomain
}
