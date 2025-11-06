package models

import "time"

// StatusLog represents a historical health check result
type StatusLog struct {
	ID           string    `json:"id" db:"id"`
	ServiceID    string    `json:"service_id" db:"service_id"`
	Status       string    `json:"status" db:"status"`               // StatusOnline, StatusOffline, or StatusUnknown
	ResponseTime *int      `json:"response_time" db:"response_time"` // Response time in milliseconds (nil if check failed)
	ErrorMessage *string   `json:"error_message" db:"error_message"` // Error details if check failed (nil if successful)
	CheckedAt    time.Time `json:"checked_at" db:"checked_at"`
}

// StatusLogResponse is the safe status log data to return to clients
type StatusLogResponse struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
	Status       string    `json:"status"`
	ResponseTime *int      `json:"response_time,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// ToResponse converts StatusLog to StatusLogResponse
func (sl *StatusLog) ToResponse() StatusLogResponse {
	return StatusLogResponse{
		ID:           sl.ID,
		ServiceID:    sl.ServiceID,
		Status:       sl.Status,
		ResponseTime: sl.ResponseTime,
		ErrorMessage: sl.ErrorMessage,
		CheckedAt:    sl.CheckedAt,
	}
}
