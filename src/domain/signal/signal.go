package signal

import (
	"time"
)

// Core domain entities

// GroupPermission defines permission levels for group operations
type GroupPermission int

const (
	DefaultGroupPermission GroupPermission = iota + 1
	EveryMember
	OnlyAdmins
)

// GroupLinkState defines the state of group links
type GroupLinkState int

const (
	DefaultGroupLinkState GroupLinkState = iota + 1
	Enabled
	EnabledWithApproval
	Disabled
)

// GroupEntry represents a Signal group
type GroupEntry struct {
	ID              string
	Name            string
	Description     string
	Members         []string
	Admins          []string
	BlockedMembers  []string
	PendingMembers  []string
	RequestingMembers []string
	GroupLinkState  GroupLinkState
}

// IdentityEntry represents a Signal identity
type IdentityEntry struct {
	Number          string
	TrustLevel      string
	AddedTimestamp  time.Time
	SafetyNumber    string
}

// SendResponse represents a response from a send operation
type SendResponse struct {
	Timestamp int64
}

// SearchResultEntry represents a search result
type SearchResultEntry struct {
	Number string
	UUID   string
}

// ISignalService defines the interface for signal service operations
type ISignalService interface {
	// Account operations
	RegisterNumber(number string, useVoice bool, captcha string) error
	VerifyRegisteredNumber(number string, token string, pin string) error
	UnregisterNumber(number string, deleteAccount bool, deleteLocalData bool) error
	GetAccounts() ([]string, error)
	
	// Messaging operations
	Send(number string, message string, recipients []string, attachments []string, isGroup bool) (*SendResponse, error)
	Receive(number string, timeout int64, ignoreAttachments bool, ignoreStories bool, maxMessages int64, sendReadReceipts bool) (string, error)
	
	// Group operations
	CreateGroup(number string, name string, members []string, description string, editGroupPermission GroupPermission, addMembersPermission GroupPermission, groupLinkState GroupLinkState, expirationTime *int) (string, error)
	GetGroups(number string) ([]GroupEntry, error)
	GetGroup(number string, groupId string) (*GroupEntry, error)
	UpdateGroup(number string, groupId string, avatar *string, description *string, name *string, expirationTime *int, groupLinkState *GroupLinkState) error
	DeleteGroup(number string, groupId string) error
	AddMembersToGroup(number string, groupId string, members []string) error
	RemoveMembersFromGroup(number string, groupId string, members []string) error
	AddAdminsToGroup(number string, groupId string, admins []string) error
	RemoveAdminsFromGroup(number string, groupId string, admins []string) error
	
	// Identity operations
	ListIdentities(number string) (*[]IdentityEntry, error)
	TrustIdentity(number string, numberToTrust string, verifiedSafetyNumber *string, trustAllKnownKeys *bool) error
	
	// QR code operations
	GetQrCodeLink(deviceName string, qrCodeVersion int) ([]byte, error)
}