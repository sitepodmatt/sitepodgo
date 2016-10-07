package webapi

import (
	"time"
)

type SitepodSession struct {
	LastSeen time.Time
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type APIError struct {
	Reason  string
	Message string
}

func (e *APIError) Error() string {
	return e.Reason + ": " + e.Message
}

func NewAPIError(reason string) error {
	err := &APIError{
		Reason:  reason,
		Message: reason,
	}
	return err
}
