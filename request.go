package innertube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// endpoint paths
const (
	epSearch                        = "/search"
	epBrowse                        = "/browse"
	epPlayer                        = "/player"
	epNext                          = "/next"
	epGetTranscript                 = "/get_transcript"
	epLiveChatGetMessages           = "/live_chat/get_live_chat"
	epLiveChatSendMessage           = "/live_chat/send_message"
	epLiveChatDeleteMessage         = "/live_chat/delete_live_chat_message"
	epLiveChatGetReplay             = "/live_chat/get_live_chat_replay"
	epLike                          = "/like/like"
	epDislike                       = "/like/dislike"
	epRemoveLike                    = "/like/removelike"
	epSubscribe                     = "/subscription/subscribe"
	epUnsubscribe                   = "/subscription/unsubscribe"
	epCreateComment                 = "/comment/create_comment"
	epCreateCommentReply            = "/comment/create_comment_reply"
	epPerformCommentAction          = "/comment/perform_comment_action"
	epNotifyChangePreference        = "/notification/modify_channel_preference"
	epAccountMenu                   = "/account/account_menu"
	epAccountNotifSettings          = "/account/set_setting"
	epGuide                         = "/guide"
	epGetNotifications              = "/notification/get_notification_menu"
	epGetUnseenCount                = "/notification/get_unseen_count"
)

// request sends a POST request to the InnerTube API and returns the raw JSON body.
func (c *Client) request(
	ctx context.Context,
	endpoint string,
	ct ClientType,
	payload map[string]interface{},
) (map[string]interface{}, error) {

	// Refresh token if needed
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	cfg := clientConfigs[ct]
	if cfg.Name == "" {
		cfg = clientConfigs[ClientWeb]
	}

	// Build URL
	apiURL := fmt.Sprintf("%s%s/%s?key=%s&prettyPrint=false",
		baseURL, apiPath, endpoint[1:], cfg.APIKey)

	// Merge context into payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["context"] = c.buildContext(ct)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("innertube: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("innertube: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("X-YouTube-Client-Name", clientNameInt(ct))
	req.Header.Set("X-YouTube-Client-Version", cfg.Version)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Accept-Language", c.opts.Lang+",en;q=0.9")

	c.mu.RLock()
	creds := c.credentials
	c.mu.RUnlock()

	if creds != nil {
		if creds.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		}
		if creds.Cookies != "" {
			req.Header.Set("Cookie", creds.Cookies)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("innertube: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("innertube: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Message:    string(respBody),
			Endpoint:   endpoint,
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("innertube: unmarshal response: %w", err)
	}

	return result, nil
}

// requestMusic sends a request to the YouTube Music base URL.
func (c *Client) requestMusic(
	ctx context.Context,
	endpoint string,
	payload map[string]interface{},
) (map[string]interface{}, error) {

	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	cfg := clientConfigs[ClientWebMusic]
	apiURL := fmt.Sprintf("%s%s/%s?key=%s&prettyPrint=false",
		musicURL, apiPath, endpoint[1:], cfg.APIKey)

	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["context"] = c.buildContext(ClientWebMusic)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("innertube: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("innertube: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Origin", musicURL)
	req.Header.Set("Referer", musicURL+"/")
	req.Header.Set("Accept-Language", c.opts.Lang+",en;q=0.9")
	req.Header.Set("X-YouTube-Client-Name", "67")
	req.Header.Set("X-YouTube-Client-Version", cfg.Version)

	c.mu.RLock()
	creds := c.credentials
	c.mu.RUnlock()

	if creds != nil {
		if creds.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		}
		if creds.Cookies != "" {
			req.Header.Set("Cookie", creds.Cookies)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("innertube: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("innertube: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Message:    string(respBody),
			Endpoint:   endpoint,
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("innertube: unmarshal response: %w", err)
	}

	return result, nil
}

// get performs a raw GET request and returns the response body.
func (c *Client) get(ctx context.Context, rawURL string, params url.Values) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgentFor(ClientWeb))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// requireAuth returns ErrNotAuthenticated if the client has no credentials.
func (c *Client) requireAuth() error {
	if !c.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	return nil
}

// clientNameInt returns the numeric string name for the X-YouTube-Client-Name header.
func clientNameInt(ct ClientType) string {
	switch ct {
	case ClientWeb:
		return "1"
	case ClientWebMusic:
		return "67"
	case ClientAndroid:
		return "3"
	case ClientAndroidMusic:
		return "21"
	case ClientIOS:
		return "5"
	case ClientTVEmbedded:
		return "85"
	default:
		return "1"
	}
}

// ─── JSON traversal helpers ───────────────────────────────────────────────────

// dig traverses a nested map[string]interface{} using dot-path style keys.
func dig(m interface{}, keys ...string) interface{} {
	cur := m
	for _, key := range keys {
		switch v := cur.(type) {
		case map[string]interface{}:
			cur = v[key]
		default:
			return nil
		}
	}
	return cur
}

// digStr is like dig but returns a string.
func digStr(m interface{}, keys ...string) string {
	v := dig(m, keys...)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// digFloat is like dig but returns a float64.
func digFloat(m interface{}, keys ...string) float64 {
	v := dig(m, keys...)
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// digBool is like dig but returns a bool.
func digBool(m interface{}, keys ...string) bool {
	v := dig(m, keys...)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// digArr is like dig but returns []interface{}.
func digArr(m interface{}, keys ...string) []interface{} {
	v := dig(m, keys...)
	if a, ok := v.([]interface{}); ok {
		return a
	}
	return nil
}

// digMap is like dig but returns map[string]interface{}.
func digMap(m interface{}, keys ...string) map[string]interface{} {
	v := dig(m, keys...)
	if mp, ok := v.(map[string]interface{}); ok {
		return mp
	}
	return nil
}

// firstText extracts the first "text" run from a YouTube runs array.
func firstText(runs []interface{}) string {
	if len(runs) == 0 {
		return ""
	}
	if m, ok := runs[0].(map[string]interface{}); ok {
		if t, ok := m["text"].(string); ok {
			return t
		}
	}
	return ""
}

// concatRuns concatenates all text runs in a YouTube runs array.
func concatRuns(runs []interface{}) string {
	result := ""
	for _, r := range runs {
		if m, ok := r.(map[string]interface{}); ok {
			if t, ok := m["text"].(string); ok {
				result += t
			}
		}
	}
	return result
}

// parseThumbnails extracts thumbnails from a raw thumbnails array.
func parseThumbnails(raw []interface{}) []Thumbnail {
	var out []Thumbnail
	for _, t := range raw {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		th := Thumbnail{
			URL: digStr(m, "url"),
		}
		if w, ok := m["width"].(float64); ok {
			th.Width = int(w)
		}
		if h, ok := m["height"].(float64); ok {
			th.Height = int(h)
		}
		out = append(out, th)
	}
	return out
}

// bestThumbnail returns the highest-resolution thumbnail from a list.
func bestThumbnail(thumbs []Thumbnail) Thumbnail {
	var best Thumbnail
	for _, t := range thumbs {
		if t.Width*t.Height > best.Width*best.Height {
			best = t
		}
	}
	return best
}

// parseDuration parses a YouTube duration object.
func parseDuration(raw map[string]interface{}) Duration {
	d := Duration{}
	if raw == nil {
		return d
	}
	if secs, ok := raw["seconds"].(float64); ok {
		d.Seconds = int(secs)
	}
	if st, ok := raw["simpleText"].(string); ok {
		d.SimpleText = st
	}
	if al, ok := raw["accessibility"].(map[string]interface{}); ok {
		if alp, ok := al["accessibilityData"].(map[string]interface{}); ok {
			d.AccessibilityLabel, _ = alp["label"].(string)
		}
	}
	return d
}

// parseBadges parses a badges array from a video renderer.
func parseBadges(raw []interface{}) []Badge {
	var out []Badge
	for _, b := range raw {
		bm, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		// Try metadataBadgeRenderer
		if mbr, ok := bm["metadataBadgeRenderer"].(map[string]interface{}); ok {
			out = append(out, Badge{
				Text:  digStr(mbr, "label"),
				Style: digStr(mbr, "style"),
			})
		}
	}
	return out
}
