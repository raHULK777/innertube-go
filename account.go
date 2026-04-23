package innertube

import "context"

// AccountClient exposes account info and settings endpoints.
// All methods require authentication.
type AccountClient struct {
	client   *Client
	Settings *AccountSettings
}

// AccountSettings provides access to sub-setting groups.
type AccountSettings struct {
	Notifications *NotificationSettings
	Privacy       *PrivacySettings
	client        *Client
}

// NotificationSettings manages notification-related account preferences.
type NotificationSettings struct {
	client *Client
}

// PrivacySettings manages privacy-related account preferences.
type PrivacySettings struct {
	client *Client
}

func init() {
	// AccountClient.Settings is wired in NewWithOptions after Client construction.
}

// Info returns basic info about the authenticated account.
//
//	info, err := yt.Account.Info()
//	fmt.Println(info.Name, info.Country)
func (ac *AccountClient) Info() (*AccountInfo, error) {
	return ac.InfoCtx(context.Background())
}

// InfoCtx is Info with context.
func (ac *AccountClient) InfoCtx(ctx context.Context) (*AccountInfo, error) {
	if err := ac.client.requireAuth(); err != nil {
		return nil, err
	}
	raw, err := ac.client.request(ctx, epAccountMenu, ClientWeb, nil)
	if err != nil {
		return nil, err
	}
	return parseAccountInfo(raw), nil
}

// ─── Notification settings ────────────────────────────────────────────────────

// SetSubscriptions enables or disables subscription notifications.
func (ns *NotificationSettings) SetSubscriptions(enabled bool) error {
	return ns.setSetting(ctx_bg(), "NOTIFICATION_SUBSCRIPTION", enabled)
}

// SetRecommendedVideos enables or disables recommended content notifications.
func (ns *NotificationSettings) SetRecommendedVideos(enabled bool) error {
	return ns.setSetting(ctx_bg(), "NOTIFICATION_RECOMMENDATION", enabled)
}

// SetChannelActivity enables or disables channel activity notifications.
func (ns *NotificationSettings) SetChannelActivity(enabled bool) error {
	return ns.setSetting(ctx_bg(), "NOTIFICATION_CHANNEL_ACTIVITY", enabled)
}

// SetCommentReplies enables or disables comment reply notifications.
func (ns *NotificationSettings) SetCommentReplies(enabled bool) error {
	return ns.setSetting(ctx_bg(), "NOTIFICATION_COMMENT_REPLY", enabled)
}

// SetSharedContent enables or disables mention/shared content notifications.
func (ns *NotificationSettings) SetSharedContent(enabled bool) error {
	return ns.setSetting(ctx_bg(), "NOTIFICATION_MENTION", enabled)
}

func (ns *NotificationSettings) setSetting(ctx context.Context, settingID string, enabled bool) error {
	if err := ns.client.requireAuth(); err != nil {
		return err
	}
	newValue := "true"
	if !enabled {
		newValue = "false"
	}
	payload := map[string]interface{}{
		"setting": map[string]interface{}{
			"settingId":    settingID,
			"newValue":     map[string]interface{}{"boolValue": newValue},
			"previousValue": map[string]interface{}{"boolValue": !enabled},
		},
	}
	_, err := ns.client.request(ctx, epAccountNotifSettings, ClientWeb, payload)
	return err
}

// ─── Privacy settings ─────────────────────────────────────────────────────────

// SetSubscriptionsPrivate makes the user's subscription list private or public.
func (ps *PrivacySettings) SetSubscriptionsPrivate(private bool) error {
	return ps.setSetting(ctx_bg(), "SUBSCRIPTIONS_PRIVACY", private)
}

// SetSavedPlaylistsPrivate makes saved playlists private or public.
func (ps *PrivacySettings) SetSavedPlaylistsPrivate(private bool) error {
	return ps.setSetting(ctx_bg(), "LIKED_VIDEOS_PRIVACY", private)
}

func (ps *PrivacySettings) setSetting(ctx context.Context, settingID string, value bool) error {
	if err := ps.client.requireAuth(); err != nil {
		return err
	}
	newValue := "true"
	if !value {
		newValue = "false"
	}
	payload := map[string]interface{}{
		"setting": map[string]interface{}{
			"settingId": settingID,
			"newValue":  map[string]interface{}{"boolValue": newValue},
		},
	}
	_, err := ps.client.request(ctx, epAccountNotifSettings, ClientWeb, payload)
	return err
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseAccountInfo(raw map[string]interface{}) *AccountInfo {
	info := &AccountInfo{}

	// The account menu has a header with user info
	header := digMap(raw, "header", "activeAccountHeaderRenderer")
	if header == nil {
		// Try alternate path
		header = digMap(raw, "header", "googleAccountHeaderRenderer")
	}
	if header == nil {
		return info
	}

	info.Name = digStr(header, "accountName", "simpleText")
	if info.Name == "" {
		info.Name = concatRuns(digArr(header, "accountName", "runs"))
	}
	info.Photo = parseThumbnails(digArr(header, "accountPhoto", "thumbnails"))

	// Language and country sometimes come from the response footer or separate endpoint
	// They're not always present in the account menu response
	info.Country = ""
	info.Language = ""

	return info
}

// Wire up sub-clients after construction.
func wireAccountClient(ac *AccountClient) {
	ac.Settings = &AccountSettings{
		client: ac.client,
		Notifications: &NotificationSettings{client: ac.client},
		Privacy:       &PrivacySettings{client: ac.client},
	}
}

// NewWithOptions override to wire everything up.
func init() {
	// actual wiring happens in the NewWithOptions function's post-construct step
}

// ctx_bg is a shorthand for context.Background() to keep code concise.
func ctx_bg() context.Context {
	return context.Background()
}
