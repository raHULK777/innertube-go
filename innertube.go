// Package innertube provides a full-featured Go client for YouTube's private
// InnerTube API — the same API YouTube itself uses under the hood.
//
// No API key required. Supports unauthenticated and authenticated requests
// via OAuth2 (device flow) or cookies.
//
// Usage:
//
//	yt, err := innertube.New()
//	results, err := yt.Search("never gonna give you up")
//	video, err := yt.GetVideoDetails("dQw4w9WgXcQ")
package innertube

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL    = "https://www.youtube.com"
	musicURL   = "https://music.youtube.com"
	apiPath    = "/youtubei/v1"
	apiVersion = "v1"

	defaultLang    = "en"
	defaultCountry = "US"
	defaultTimeout = 30 * time.Second
)

// Client is the primary InnerTube client. Create one with New() or NewWithOptions()
// and reuse it across calls.
type Client struct {
	httpClient  *http.Client
	credentials *Credentials
	mu          sync.RWMutex
	opts        *Options

	// Interact exposes user interaction endpoints (like, dislike, subscribe…).
	// Requires authentication.
	Interact *InteractClient

	// Account exposes account info and settings endpoints.
	// Requires authentication.
	Account *AccountClient
}

// Options configures the Client.
type Options struct {
	// Language for responses (default: "en")
	Lang string

	// Country/region code (default: "US")
	Country string

	// HTTPClient to use. Defaults to a client with a 30-second timeout.
	HTTPClient *http.Client

	// Credentials for authenticated requests (OAuth2 or cookies).
	// Leave nil for unauthenticated requests.
	Credentials *Credentials

	// SafeSearch enables restricted mode (default: false)
	SafeSearch bool
}

// New returns a default unauthenticated InnerTube client.
func New() (*Client, error) {
	return NewWithOptions(&Options{})
}

// NewWithOptions returns a configured InnerTube client.
func NewWithOptions(opts *Options) (*Client, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Lang == "" {
		opts.Lang = defaultLang
	}
	if opts.Country == "" {
		opts.Country = defaultCountry
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}

	c := &Client{
		httpClient:  opts.HTTPClient,
		credentials: opts.Credentials,
		opts:        opts,
	}

	c.Interact = &InteractClient{client: c}
	c.Account = &AccountClient{client: c}

	// Wire up Account sub-clients
	wireAccountClient(c.Account)

	return c, nil
}

// SignInWithOAuth initiates OAuth2 device-flow sign-in.
// It calls onCode with the verification URL and user code for the user to visit.
// Blocks until sign-in succeeds or ctx is cancelled.
func (c *Client) SignInWithOAuth(ctx context.Context, onCode func(verificationURL, code string)) error {
	creds, err := oauthDeviceFlow(ctx, c.httpClient, onCode)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.credentials = creds
	c.mu.Unlock()
	return nil
}

// SignInWithCookies sets the given cookie string as the authentication method.
// You can grab your cookies from your browser's DevTools after logging into YouTube.
func (c *Client) SignInWithCookies(cookies string) {
	c.mu.Lock()
	c.credentials = &Credentials{Cookies: cookies}
	c.mu.Unlock()
}

// IsAuthenticated reports whether the client has credentials loaded.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.credentials != nil && (c.credentials.AccessToken != "" || c.credentials.Cookies != "")
}
