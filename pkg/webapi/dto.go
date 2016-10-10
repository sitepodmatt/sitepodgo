package webapi

import (
	"time"
)

type SitepodSession struct {
	LastSeen      *time.Time `json:"last_seen,omitempty"`
	MaxAge        int        `json:"max_age,omitempty"`
	Authenticated bool       `json:"authenticated?"`
}

type LoginRequest struct {
	Action string `json:"action"`
	Data   struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"data"`
}

type APIError struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
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
