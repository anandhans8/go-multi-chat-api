package signal

import (
	domainSignal "go-multi-chat-api/src/domain/signal"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"go.uber.org/zap"
)

// ISignalUseCase defines the interface for signal use cases
type ISignalUseCase interface {
	// Account operations
	RegisterNumber(number string, useVoice bool, captcha string) error
	VerifyRegisteredNumber(number string, token string, pin string) error
	UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error
	GetAccounts() ([]string, error)

	// Messaging operations
	Send(number string, message string, recipients []string, attachments []string, isGroup bool) (*domainSignal.SendResponse, error)
	Receive(number string, timeout int64, ignoreAttachments bool, ignoreStories bool, maxMessages int64, sendReadReceipts bool) (string, error)

	// Group operations
	CreateGroup(number string, name string, members []string, description string, editGroupPermission domainSignal.GroupPermission, addMembersPermission domainSignal.GroupPermission, groupLinkState domainSignal.GroupLinkState, expirationTime *int) (string, error)
	GetGroups(number string) ([]domainSignal.GroupEntry, error)
	GetGroup(number string, groupId string) (*domainSignal.GroupEntry, error)
	UpdateGroup(number string, groupId string, avatar *string, description *string, name *string, expirationTime *int, groupLinkState *domainSignal.GroupLinkState) error
	DeleteGroup(number string, groupId string) error
	AddMembersToGroup(number string, groupId string, members []string) error
	RemoveMembersFromGroup(number string, groupId string, members []string) error
	AddAdminsToGroup(number string, groupId string, admins []string) error
	RemoveAdminsFromGroup(number string, groupId string, admins []string) error

	// Identity operations
	ListIdentities(number string) (*[]domainSignal.IdentityEntry, error)
	TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error

	// QR code operations
	GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error)
}

// SignalUseCase implements the ISignalUseCase interface
type SignalUseCase struct {
	signalService domainSignal.ISignalService
	Logger        *logger.Logger
}

// NewSignalUseCase creates a new SignalUseCase
func NewSignalUseCase(signalService domainSignal.ISignalService, loggerInstance *logger.Logger) ISignalUseCase {
	return &SignalUseCase{
		signalService: signalService,
		Logger:        loggerInstance,
	}
}

// RegisterNumber registers a new Signal number
func (s *SignalUseCase) RegisterNumber(number string, useVoice bool, captcha string) error {
	s.Logger.Info("Registering number", zap.String("number", number))
	return s.signalService.RegisterNumber(number, useVoice, captcha)
}

// VerifyRegisteredNumber verifies a registered Signal number
func (s *SignalUseCase) VerifyRegisteredNumber(number string, token string, pin string) error {
	s.Logger.Info("Verifying registered number", zap.String("number", number))
	return s.signalService.VerifyRegisteredNumber(number, token, pin)
}

// UnregisterNumber unregisters a Signal number
func (s *SignalUseCase) UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error {
	s.Logger.Info("Unregistering number", zap.String("number", number))
	return s.signalService.UnregisterNumber(number, deleteAccount, deleteLocalData)
}

// GetAccounts gets all registered Signal accounts
func (s *SignalUseCase) GetAccounts() ([]string, error) {
	s.Logger.Info("Getting all accounts")
	return s.signalService.GetAccounts()
}

// Send sends a message via Signal
func (s *SignalUseCase) Send(number string, message string, recipients []string, attachments []string, isGroup bool) (*domainSignal.SendResponse, error) {
	s.Logger.Info("Sending message",
		zap.String("from", number),
		zap.Int("recipientsCount", len(recipients)),
		zap.Int("attachmentsCount", len(attachments)),
		zap.Bool("isGroup", isGroup))
	return s.signalService.Send(number, message, recipients, attachments, isGroup)
}

// Receive receives messages via Signal
func (s *SignalUseCase) Receive(number string, timeout int64, ignoreAttachments bool, ignoreStories bool, maxMessages int64, sendReadReceipts bool) (string, error) {
	s.Logger.Info("Receiving messages", zap.String("number", number))
	return s.signalService.Receive(number, timeout, ignoreAttachments, ignoreStories, maxMessages, sendReadReceipts)
}

// CreateGroup creates a new Signal group
func (s *SignalUseCase) CreateGroup(number string, name string, members []string, description string, editGroupPermission domainSignal.GroupPermission, addMembersPermission domainSignal.GroupPermission, groupLinkState domainSignal.GroupLinkState, expirationTime *int) (string, error) {
	s.Logger.Info("Creating group",
		zap.String("name", name),
		zap.String("creator", number),
		zap.Int("membersCount", len(members)))
	return s.signalService.CreateGroup(number, name, members, description, editGroupPermission, addMembersPermission, groupLinkState, expirationTime)
}

// GetGroups gets all Signal groups
func (s *SignalUseCase) GetGroups(number string) ([]domainSignal.GroupEntry, error) {
	s.Logger.Info("Getting all groups", zap.String("number", number))
	return s.signalService.GetGroups(number)
}

// GetGroup gets a specific Signal group
func (s *SignalUseCase) GetGroup(number string, groupId string) (*domainSignal.GroupEntry, error) {
	s.Logger.Info("Getting group", zap.String("groupId", groupId))
	return s.signalService.GetGroup(number, groupId)
}

// UpdateGroup updates a Signal group
func (s *SignalUseCase) UpdateGroup(number string, groupId string, avatar *string, description *string, name *string, expirationTime *int, groupLinkState *domainSignal.GroupLinkState) error {
	s.Logger.Info("Updating group", zap.String("groupId", groupId))
	return s.signalService.UpdateGroup(number, groupId, avatar, description, name, expirationTime, groupLinkState)
}

// DeleteGroup deletes a Signal group
func (s *SignalUseCase) DeleteGroup(number string, groupId string) error {
	s.Logger.Info("Deleting group", zap.String("groupId", groupId))
	return s.signalService.DeleteGroup(number, groupId)
}

// AddMembersToGroup adds members to a Signal group
func (s *SignalUseCase) AddMembersToGroup(number string, groupId string, members []string) error {
	s.Logger.Info("Adding members to group",
		zap.String("groupId", groupId),
		zap.Int("membersCount", len(members)))
	return s.signalService.AddMembersToGroup(number, groupId, members)
}

// RemoveMembersFromGroup removes members from a Signal group
func (s *SignalUseCase) RemoveMembersFromGroup(number string, groupId string, members []string) error {
	s.Logger.Info("Removing members from group",
		zap.String("groupId", groupId),
		zap.Int("membersCount", len(members)))
	return s.signalService.RemoveMembersFromGroup(number, groupId, members)
}

// AddAdminsToGroup adds admins to a Signal group
func (s *SignalUseCase) AddAdminsToGroup(number string, groupId string, admins []string) error {
	s.Logger.Info("Adding admins to group",
		zap.String("groupId", groupId),
		zap.Int("adminsCount", len(admins)))
	return s.signalService.AddAdminsToGroup(number, groupId, admins)
}

// RemoveAdminsFromGroup removes admins from a Signal group
func (s *SignalUseCase) RemoveAdminsFromGroup(number string, groupId string, admins []string) error {
	s.Logger.Info("Removing admins from group",
		zap.String("groupId", groupId),
		zap.Int("adminsCount", len(admins)))
	return s.signalService.RemoveAdminsFromGroup(number, groupId, admins)
}

// ListIdentities lists all Signal identities
func (s *SignalUseCase) ListIdentities(number string) (*[]domainSignal.IdentityEntry, error) {
	s.Logger.Info("Listing identities", zap.String("number", number))
	return s.signalService.ListIdentities(number)
}

// TrustIdentity trusts a Signal identity
func (s *SignalUseCase) TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error {
	s.Logger.Info("Trusting identity",
		zap.String("number", number),
		zap.String("numberToTrust", numberToTrust))
	return s.signalService.TrustIdentity(number, numberToTrust, verifiedSafetyNumber, trustAllKnownKeys)
}

// GetQrCodeLink gets a QR code link for Signal
func (s *SignalUseCase) GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error) {
	s.Logger.Info("Getting QR code link",
		zap.String("deviceName", deviceName),
		zap.Int("qrCodeVersion", qrCodeVersion))
	return s.signalService.GetQrCodeLink(deviceName, qrCodeVersion)
}
