package models

type APIResponse struct {
	Success bool              `json:"success"`
	Data    any               `json:"data,omitempty"`
	Message string            `json:"message,omitempty"`
	Error   map[string]string `json:"error,omitempty"`
}
