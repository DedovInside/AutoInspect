package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

const (
	yandexAuthURL     = "https://oauth.yandex.ru/authorize"
	yandexTokenURL    = "https://oauth.yandex.ru/token"
	yandexUserInfoURL = "https://login.yandex.ru/info"
)

type YandexOAuthClient struct {
	config     *oauth2.Config
	httpClient *http.Client
}

type YandexProfile struct {
	ID           string `json:"id"`
	Login        string `json:"login"`
	DefaultEmail string `json:"default_email"`
}

func NewYandexOAuthClient(clientID, clientSecret, redirectURL string) *YandexOAuthClient {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil
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
			Scopes: []string{"login:email", "login:info"},
		},
		httpClient: http.DefaultClient,
	}
}

func (c *YandexOAuthClient) AuthCodeURL(state string) string {
	return c.config.AuthCodeURL(state)
}

func (c *YandexOAuthClient) ExchangeCodeAndFetchProfile(ctx context.Context, code string) (*YandexProfile, error) {
	token, err := c.config.Exchange(ctx, code)

	if err != nil {
		return nil, fmt.Errorf("exchange yandex code: %w", err)
	}

	values := url.Values{}
	values.Set("format", "json")
	values.Set("jwt_secret", "")
	values.Set("oauth_token", token.AccessToken)

	endpoint := yandexUserInfoURL + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

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
