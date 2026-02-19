package dto

type UserHistoryPagination struct {
	Data       []UserHistoryResponse `json:"data"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	Sort       string                `json:"sort"`
	OrderBy    string                `json:"order_by"`
	TotalCount int64                 `json:"total_count"`
}

type UserHistoryIDRequest struct {
	ID int `json:"id" binding:"required,numeric,min=1" uri:"id" label:"ID"`
}

type UserHistoryRequest struct {
	Page    int    `json:"page" binding:"omitempty,numeric,min=1,max=100"`
	Limit   int    `json:"limit" binding:"omitempty,numeric,min=10,max=100"`
	Sort    string `json:"sort" binding:"omitempty"`
	OrderBy string `json:"order_by" binding:"omitempty"`
}

type UserHistoryResponse struct {
	ID            int `json:"id"`
	UserID        string `json:"user_id"`
	ActionType    string `json:"action_type"`
	OldValue      string `json:"old_value"`
	NewValue      string `json:"new_value"`
	Reason        string `json:"reason"`
	ChangedBy     string `json:"changed_by"`
	RevokeToken   string `json:"revoke_token"`
	RevokeExpires string `json:"revoke_expires"`
	IPAddress     string `json:"ip_address"`
	UserAgent     string `json:"user_agent"`
	ChangedAt     string `json:"changed_at"`
	DeletedAt     string `json:"deleted_at"`
}
