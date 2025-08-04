package send

type MessageRequest struct {
	Type       string   `json:"type" binding:"required"`
	Message    string   `json:"message" binding:"required"`
	Recipients []string `json:"recipients" binding:"required"`
	UserID     int      `json:"user_id" binding:"required"`
}

type MessageResponse struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message,omitempty"`
}

type MessageStatusRequest struct {
	ID int `uri:"id" binding:"required"`
}

type MessageStatusResponse struct {
	ID           int       `json:"id"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	Recipients   string    `json:"recipients"`
	ErrorMessage string    `json:"error_message,omitempty"`
	RetryCount   int       `json:"retry_count"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}
