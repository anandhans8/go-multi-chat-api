package signal_client

import (
	domainSignal "go-multi-chat-api/src/domain/signal"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"time"

	"go.uber.org/zap"
)

// Repository implements the domainSignal.ISignalService interface
type Repository struct {
	client *SignalClient
	Logger *logger.Logger
}

// GetClient returns the underlying SignalClient
func (r *Repository) GetClient() *SignalClient {
	return r.client
}

// NewSignalRepository creates a new Repository
func NewSignalRepository(signalCliConfig string, attachmentTmpDir string, avatarTmpDir string, signalCliMode SignalCliMode, jsonRpc2ClientConfigPath string, signalCliApiConfigPath string, receiveWebhookUrl string, loggerInstance *logger.Logger) domainSignal.ISignalService {
	client := NewSignalClient(signalCliConfig, attachmentTmpDir, avatarTmpDir, signalCliMode, jsonRpc2ClientConfigPath, signalCliApiConfigPath, receiveWebhookUrl, loggerInstance)
	return &Repository{
		client: client,
		Logger: loggerInstance,
	}
}

// RegisterNumber registers a new Signal number
func (r *Repository) RegisterNumber(number string, useVoice bool, captcha string) error {
	r.Logger.Info("Repository: Registering number", zap.String("number", number))
	return r.client.RegisterNumber(number, useVoice, captcha)
}

// VerifyRegisteredNumber verifies a registered Signal number
func (r *Repository) VerifyRegisteredNumber(number string, token string, pin string) error {
	r.Logger.Info("Repository: Verifying registered number", zap.String("number", number))
	return r.client.VerifyRegisteredNumber(number, token, pin)
}

// UnregisterNumber unregisters a Signal number
func (r *Repository) UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error {
	r.Logger.Info("Repository: Unregistering number", zap.String("number", number))
	return r.client.UnregisterNumber(number, deleteAccount, deleteLocalData)
}

// GetAccounts gets all registered Signal accounts
func (r *Repository) GetAccounts() ([]string, error) {
	r.Logger.Info("Repository: Getting all accounts")
	return r.client.GetAccounts()
}

// Send sends a message via Signal
func (r *Repository) Send(number string, message string, recipients []string, attachments []string, isGroup bool) (*domainSignal.SendResponse, error) {
	r.Logger.Info("Repository: Sending message",
		zap.String("from", number),
		zap.Int("recipientsCount", len(recipients)),
		zap.Int("attachmentsCount", len(attachments)),
		zap.Bool("isGroup", isGroup))

	response, err := r.client.SendV1(number, message, recipients, attachments, isGroup)
	if err != nil {
		return nil, err
	}

	// Convert from internal SendResponse to domain SendResponse
	return &domainSignal.SendResponse{
		Timestamp: response.Timestamp,
	}, nil
}

// Receive receives messages via Signal
func (r *Repository) Receive(number string, timeout int64, ignoreAttachments bool, ignoreStories bool, maxMessages int64, sendReadReceipts bool) (string, error) {
	r.Logger.Info("Repository: Receiving messages", zap.String("number", number))
	return r.client.Receive(number, timeout, ignoreAttachments, ignoreStories, maxMessages, sendReadReceipts)
}

// CreateGroup creates a new Signal group
func (r *Repository) CreateGroup(number string, name string, members []string, description string, editGroupPermission domainSignal.GroupPermission, addMembersPermission domainSignal.GroupPermission, groupLinkState domainSignal.GroupLinkState, expirationTime *int) (string, error) {
	r.Logger.Info("Repository: Creating group",
		zap.String("name", name),
		zap.String("creator", number),
		zap.Int("membersCount", len(members)))

	// Convert domain types to internal types
	internalEditGroupPermission := GroupPermission(editGroupPermission)
	internalAddMembersPermission := GroupPermission(addMembersPermission)
	internalGroupLinkState := GroupLinkState(groupLinkState)

	return r.client.CreateGroup(number, name, members, description, internalEditGroupPermission, internalAddMembersPermission, internalGroupLinkState, expirationTime)
}

// GetGroups gets all Signal groups
func (r *Repository) GetGroups(number string) ([]domainSignal.GroupEntry, error) {
	r.Logger.Info("Repository: Getting all groups", zap.String("number", number))

	groups, err := r.client.GetGroups(number)
	if err != nil {
		return nil, err
	}

	// Convert from internal GroupEntry to domain GroupEntry
	domainGroups := make([]domainSignal.GroupEntry, len(groups))
	for i, group := range groups {
		domainGroups[i] = domainSignal.GroupEntry{
			ID:                group.Id,
			Name:              group.Name,
			Description:       group.Description,
			Members:           group.Members,
			Admins:            group.Admins,
			BlockedMembers:    []string{}, // Not directly available in the internal model
			PendingMembers:    group.PendingInvites,
			RequestingMembers: group.PendingRequests,
			GroupLinkState:    domainSignal.DefaultGroupLinkState, // Not directly available in the internal model
		}
	}

	return domainGroups, nil
}

// GetGroup gets a specific Signal group
func (r *Repository) GetGroup(number string, groupId string) (*domainSignal.GroupEntry, error) {
	r.Logger.Info("Repository: Getting group", zap.String("groupId", groupId))

	group, err := r.client.GetGroup(number, groupId)
	if err != nil {
		return nil, err
	}

	// Convert from internal GroupEntry to domain GroupEntry
	domainGroup := &domainSignal.GroupEntry{
		ID:                group.Id,
		Name:              group.Name,
		Description:       group.Description,
		Members:           group.Members,
		Admins:            group.Admins,
		BlockedMembers:    []string{}, // Not directly available in the internal model
		PendingMembers:    group.PendingInvites,
		RequestingMembers: group.PendingRequests,
		GroupLinkState:    domainSignal.DefaultGroupLinkState, // Not directly available in the internal model
	}

	return domainGroup, nil
}

// UpdateGroup updates a Signal group
func (r *Repository) UpdateGroup(number string, groupId string, avatar *string, description *string, name *string, expirationTime *int, groupLinkState *domainSignal.GroupLinkState) error {
	r.Logger.Info("Repository: Updating group", zap.String("groupId", groupId))

	// Convert domain types to internal types
	var internalGroupLinkState *GroupLinkState
	if groupLinkState != nil {
		internalGroupLinkStateValue := GroupLinkState(*groupLinkState)
		internalGroupLinkState = &internalGroupLinkStateValue
	}

	return r.client.UpdateGroup(number, groupId, avatar, description, name, expirationTime, internalGroupLinkState)
}

// DeleteGroup deletes a Signal group
func (r *Repository) DeleteGroup(number string, groupId string) error {
	r.Logger.Info("Repository: Deleting group", zap.String("groupId", groupId))
	return r.client.DeleteGroup(number, groupId)
}

// AddMembersToGroup adds members to a Signal group
func (r *Repository) AddMembersToGroup(number string, groupId string, members []string) error {
	r.Logger.Info("Repository: Adding members to group",
		zap.String("groupId", groupId),
		zap.Int("membersCount", len(members)))
	return r.client.AddMembersToGroup(number, groupId, members)
}

// RemoveMembersFromGroup removes members from a Signal group
func (r *Repository) RemoveMembersFromGroup(number string, groupId string, members []string) error {
	r.Logger.Info("Repository: Removing members from group",
		zap.String("groupId", groupId),
		zap.Int("membersCount", len(members)))
	return r.client.RemoveMembersFromGroup(number, groupId, members)
}

// AddAdminsToGroup adds admins to a Signal group
func (r *Repository) AddAdminsToGroup(number string, groupId string, admins []string) error {
	r.Logger.Info("Repository: Adding admins to group",
		zap.String("groupId", groupId),
		zap.Int("adminsCount", len(admins)))
	return r.client.AddAdminsToGroup(number, groupId, admins)
}

// RemoveAdminsFromGroup removes admins from a Signal group
func (r *Repository) RemoveAdminsFromGroup(number string, groupId string, admins []string) error {
	r.Logger.Info("Repository: Removing admins from group",
		zap.String("groupId", groupId),
		zap.Int("adminsCount", len(admins)))
	return r.client.RemoveAdminsFromGroup(number, groupId, admins)
}

// ListIdentities lists all Signal identities
func (r *Repository) ListIdentities(number string) (*[]domainSignal.IdentityEntry, error) {
	r.Logger.Info("Repository: Listing identities", zap.String("number", number))

	identities, err := r.client.ListIdentities(number)
	if err != nil {
		return nil, err
	}

	// Convert from internal IdentityEntry to domain IdentityEntry
	domainIdentities := make([]domainSignal.IdentityEntry, len(*identities))
	for i, identity := range *identities {
		// Parse the Added string to time.Time if needed
		var addedTime time.Time
		// Use Status as TrustLevel
		domainIdentities[i] = domainSignal.IdentityEntry{
			Number:         identity.Number,
			TrustLevel:     identity.Status,
			AddedTimestamp: addedTime, // Default zero time
			SafetyNumber:   identity.SafetyNumber,
		}
	}

	return &domainIdentities, nil
}

// TrustIdentity trusts a Signal identity
func (r *Repository) TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error {
	r.Logger.Info("Repository: Trusting identity",
		zap.String("number", number),
		zap.String("numberToTrust", numberToTrust))
	return r.client.TrustIdentity(number, numberToTrust, verifiedSafetyNumber, trustAllKnownKeys)
}

// GetQrCodeLink gets a QR code link for Signal
func (r *Repository) GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error) {
	r.Logger.Info("Repository: Getting QR code link",
		zap.String("deviceName", deviceName),
		zap.Int("qrCodeVersion", qrCodeVersion))
	return r.client.GetQrCodeLink(deviceName, qrCodeVersion)
}
