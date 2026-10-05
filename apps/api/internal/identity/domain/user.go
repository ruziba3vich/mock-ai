package domain

import "time"

type User struct {
	ID            string    `json:"id"`
	DisplayName   string    `json:"display_name"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Email         string    `json:"email,omitempty"`
	EmailVerified bool      `json:"email_verified"`
	Roles         []string  `json:"roles"`
	CreatedAt     time.Time `json:"created_at"`
}

type ExternalProfile struct {
	Provider      string
	Subject       string
	DisplayName   string
	AvatarURL     string
	Email         string
	EmailVerified bool
	RawProfile    []byte
}
