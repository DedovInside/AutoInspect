package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"golang.org/x/oauth2"
)

const (
	yandexAuthURL = "https://oauth.yandex.ru/authorize"
	// #nosec
	yandexTokenURL    = "https://oauth.yandex.ru/token"
	yandexUserInfoURL = "https://login.yandex.ru/info"
	yandexAvatarURL   = "https://avatars.yandex.net/get-yapic/%s/islands-200"
)

type YandexOAuthClient struct {
	config     *oauth2.Config
	httpClient *http.Client
}

type YandexProfile struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DefaultEmail    string `json:"default_email"`
	DefaultAvatarID string `json:"default_avatar_id"`
	IsAvatarEmpty   bool   `json:"is_avatar_empty"`
}

func NewYandexOAuthClient(clientID, clientSecret, redirectURL string, timeout time.Duration) (*YandexOAuthClient, error) {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, domain.ErrInvalidOAuthConfig
	}

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &YandexOAuthClient{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  yandexAuthURL,
				TokenURL: yandexTokenURL,
			},
			Scopes: []string{"login:email", "login:info", "login:avatar"},
		},
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func (p *YandexProfile) AvatarURL() *string {
	if p == nil || p.IsAvatarEmpty || strings.TrimSpace(p.DefaultAvatarID) == "" {
		return nil
	}

	url := fmt.Sprintf(yandexAvatarURL, p.DefaultAvatarID)
	return &url
}

func (c *YandexOAuthClient) AuthCodeURL(state string) string {
	return c.config.AuthCodeURL(state)
}

func (c *YandexOAuthClient) ExchangeCodeAndFetchProfile(ctx context.Context, code string) (*YandexProfile, error) {
	token, err := c.config.Exchange(ctx, code)

	if err != nil {
		return nil, fmt.Errorf("exchange yandex code: %w", err)
	}

	endpoint := yandexUserInfoURL + "?format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)

	if err != nil {
		return nil, fmt.Errorf("create yandex profile request: %w", err)
	}

	req.Header.Set("Authorization", "OAuth "+token.AccessToken)

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("request yandex profile: %w", err)
	}

	defer func() { _ = resp.Body.Close() }() // !

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("yandex profile bad status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	profile := &YandexProfile{}

	if err := json.NewDecoder(resp.Body).Decode(profile); err != nil {
		return nil, fmt.Errorf("decode yandex profile: %w", err)
	}

	if profile.ID == "" {
		return nil, fmt.Errorf("empty yandex profile id")
	}

	return profile, nil
}
