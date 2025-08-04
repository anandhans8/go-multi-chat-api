package signal

import ds "go-multi-chat-api/src/infrastructure/datastructs"

type MessageRequest struct {
	Type       string   `json:"type" binding:"required"`
	Message    string   `json:"message" binding:"required"`
	Recipients []string `json:"recipients" binding:"required"`
}

type SendMessage struct {
	Number            string              `json:"number" binding:"required"`
	Recipients        []string            `json:"recipients"`
	Recipient         string              `json:"recipient" swaggerignore:"true"` //some REST API consumers (like the Synology NAS) do not support an array as recipients, so we provide this string parameter here as backup. In order to not confuse anyone, the parameter won't be exposed in the Swagger UI (most users are fine with the recipients parameter).
	Message           string              `json:"message"`
	Base64Attachments []string            `json:"base64_attachments" example:"<BASE64 ENCODED DATA>,data:<MIME-TYPE>;base64<comma><BASE64 ENCODED DATA>,data:<MIME-TYPE>;filename=<FILENAME>;base64<comma><BASE64 ENCODED DATA>"`
	Sticker           string              `json:"sticker"`
	Mentions          []ds.MessageMention `json:"mentions"`
	QuoteTimestamp    *int64              `json:"quote_timestamp"`
	QuoteAuthor       *string             `json:"quote_author"`
	QuoteMessage      *string             `json:"quote_message"`
	QuoteMentions     []ds.MessageMention `json:"quote_mentions"`
	TextMode          *string             `json:"text_mode" enums:"normal,styled"`
	EditTimestamp     *int64              `json:"edit_timestamp"`
	NotifySelf        *bool               `json:"notify_self"`
	LinkPreview       *ds.LinkPreviewType `json:"link_preview"`
	ViewOnce          *bool               `json:"view_once"`
}

type SendMessageResponse struct {
	Timestamp string `json:"timestamp"`
}

type Error struct {
	Msg string `json:"error"`
}

type SendMessageError struct {
	Msg             string   `json:"error"`
	ChallengeTokens []string `json:"challenge_tokens,omitempty"`
	Account         string   `json:"account"`
}

type RegisterNumberRequest struct {
	UseVoice bool   `json:"use_voice"`
	Captcha  string `json:"captcha"`
}

type VerifyNumberSettings struct {
	Pin string `json:"pin"`
}
