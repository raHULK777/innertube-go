package innertube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Google OAuth2 client ID used by YouTube TV — allows device-flow auth
	// without registering your own OAuth app.
	oauthClientID     = "861556708454-d6dlm3lh05idd8npek18k6be8ba3oc68.apps.googleusercontent.com"
	oauthClientSecret = "SboVhoG9s0rNafixCSGGKXAT"
	oauthScope        = "https://www.googleapis.com/auth/youtube"

	oauthDeviceCodeURL = "https://oauth2.googleapis.com/device/code"
	oauthTokenURL      = "https://oauth2.googleapis.com/token"

	pollInterval = 5 * time.Second
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
}

// oauthDeviceFlow executes the full OAuth2 device authorization flow.
// It calls onCode with the URL/code for the user to visit, then polls
// until the user approves or ctx is cancelled.
func oauthDeviceFlow(ctx context.Context, hc *http.Client, onCode func(url, code string)) (*Credentials, error) {
	// Step 1: request device code
	dc, err := requestDeviceCode(ctx, hc)
	if err != nil {
		return nil, fmt.Errorf("innertube/oauth: get device code: %w", err)
	}

	// Notify caller of the code to show the user
	onCode(dc.VerificationURL, dc.UserCode)

	interval := time.Duration(dc.Interval) * time.Second
	if interval < pollInterval {
		interval = pollInterval
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	// Step 2: poll for token
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		tok, err := pollToken(ctx, hc, dc.DeviceCode)
		if err != nil {
			return nil, fmt.Errorf("innertube/oauth: poll token: %w", err)
		}
		if tok == nil {
			// Still pending
			continue
		}
		return tok, nil
	}

	return nil, fmt.Errorf("innertube/oauth: authorization timed out")
}

func requestDeviceCode(ctx context.Context, hc *http.Client) (*deviceCodeResponse, error) {
	payload := map[string]string{
		"client_id": oauthClientID,
		"scope":     oauthScope,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthDeviceCodeURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dc deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, err
	}
	return &dc, nil
}

// pollToken returns nil, nil if the user hasn't authorized yet (authorization_pending).
func pollToken(ctx context.Context, hc *http.Client, deviceCode string) (*Credentials, error) {
	payload := map[string]string{
		"client_id":     oauthClientID,
		"client_secret": oauthClientSecret,
		"device_code":   deviceCode,
		"grant_type":    "urn:ietf:params:oauth:grant-type:device_code",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tok tokenResponse
	if err := json.Unmarshal(rawBody, &tok); err != nil {
		return nil, err
	}

	if tok.Error == "authorization_pending" || tok.Error == "slow_down" {
		return nil, nil
	}
	if tok.Error != "" {
		return nil, fmt.Errorf("innertube/oauth: token error: %s", tok.Error)
	}

	creds := &Credentials{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Scope:        tok.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}
	return creds, nil
}

// refreshOAuthToken uses the refresh token to get a new access token.
func refreshOAuthToken(ctx context.Context, hc *http.Client, refreshToken string) (*Credentials, error) {
	payload := map[string]string{
		"client_id":     oauthClientID,
		"client_secret": oauthClientSecret,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrTokenRefreshFailed, tok.Error)
	}

	creds := &Credentials{
		AccessToken:  tok.AccessToken,
		RefreshToken: refreshToken, // reuse existing refresh token
		TokenType:    tok.TokenType,
		Scope:        tok.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}
	return creds, nil
}

// ensureValidToken refreshes the OAuth token if it's expired.
func (c *Client) ensureValidToken(ctx context.Context) error {
	c.mu.RLock()
	creds := c.credentials
	c.mu.RUnlock()

	if creds == nil || creds.RefreshToken == "" || !creds.IsExpired() {
		return nil
	}

	fresh, err := refreshOAuthToken(ctx, c.httpClient, creds.RefreshToken)
	if err != nil {
		return err
	}

	// Preserve visitor data
	fresh.VisitorData = creds.VisitorData
	fresh.Cookies = creds.Cookies

	c.mu.Lock()
	c.credentials = fresh
	c.mu.Unlock()

	return nil
}

// GetCredentials returns a copy of the current credentials (for persistence).
func (c *Client) GetCredentials() *Credentials {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.credentials == nil {
		return nil
	}
	cp := *c.credentials
	return &cp
}

// SetCredentials replaces the current credentials (for restoring from storage).
func (c *Client) SetCredentials(creds *Credentials) {
	c.mu.Lock()
	c.credentials = creds
	c.mu.Unlock()
}
