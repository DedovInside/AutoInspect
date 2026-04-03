package dto

import "time"

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type OAuthYandexExchangeRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type AuthTokensResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AuthResponse struct {
	Tokens AuthTokensResponse `json:"tokens"`
	User   UserResponse       `json:"user"`
}
