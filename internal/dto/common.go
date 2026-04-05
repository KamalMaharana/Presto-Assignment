package dto

type BaseResponse struct {
	StatusCode int         `json:"status_code"`
	RequestID  string      `json:"request_id"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id"`
}

type MessageResponse struct {
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
