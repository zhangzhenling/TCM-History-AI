// Package dto carries the request/response DTOs (data transfer objects) for
// the User Service application layer. DTOs decouple the wire format from the
// domain entity structs.
package dto

// RegisterRequest is the payload for POST /api/v1/auth/register.
type RegisterRequest struct {
	Username string `json:"username,required"`
	Password string `json:"password,required"`
	Email    string `json:"email,optional"`
	Phone    string `json:"phone,optional"`
}

// LoginRequest is the payload for POST /api/v1/auth/login.
type LoginRequest struct {
	Username string `json:"username,required"`
	Password string `json:"password,required"`
}

// RefreshRequest is the payload for POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token,required"`
}

// TokenResponse is returned by every auth endpoint that mints a token pair.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token TTL in seconds
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
}
