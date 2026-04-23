package innertube

import "errors"

// Sentinel errors returned by the library.
var (
	// ErrNotAuthenticated is returned when an authenticated endpoint is called without credentials.
	ErrNotAuthenticated = errors.New("innertube: not authenticated — call SignInWithOAuth or SignInWithCookies first")

	// ErrNoContinuation is returned when GetContinuation() is called but no continuation token exists.
	ErrNoContinuation = errors.New("innertube: no continuation token available")

	// ErrNoStreamFound is returned when no format matches the requested StreamOptions.
	ErrNoStreamFound = errors.New("innertube: no stream found matching the requested options")

	// ErrVideoUnavailable is returned when a video cannot be accessed (private, deleted, etc.)
	ErrVideoUnavailable = errors.New("innertube: video is unavailable")

	// ErrLiveChatDisabled is returned when a video has no live chat.
	ErrLiveChatDisabled = errors.New("innertube: live chat is not available for this video")

	// ErrTokenRefreshFailed is returned when an OAuth2 token refresh fails.
	ErrTokenRefreshFailed = errors.New("innertube: failed to refresh OAuth2 token")

	// ErrCipherNotFound is returned when the player's cipher function cannot be found.
	ErrCipherNotFound = errors.New("innertube: could not locate signature cipher in player JS")
)

// APIError is returned when the InnerTube API responds with an error status.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
	Endpoint   string
}

func (e *APIError) Error() string {
	return "innertube: API error " + e.Status + " on " + e.Endpoint + ": " + e.Message
}
