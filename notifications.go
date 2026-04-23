package innertube

import "context"

// GetNotifications returns the authenticated user's notification feed.
// Requires authentication.
//
//	notifs, err := yt.GetNotifications()
func (c *Client) GetNotifications() (*NotificationsResult, error) {
	return c.GetNotificationsCtx(context.Background())
}

// GetNotificationsCtx is GetNotifications with context.
func (c *Client) GetNotificationsCtx(ctx context.Context) (*NotificationsResult, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"notificationsMenuRequestType": "NOTIFICATIONS_MENU_REQUEST_TYPE_INBOX",
	}
	raw, err := c.request(ctx, epGetNotifications, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseNotifications(raw, c), nil
}

// getNotificationsContinuation fetches the next page of notifications.
func (c *Client) getNotificationsContinuation(token string) (*NotificationsResult, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(context.Background(), epGetNotifications, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseNotifications(raw, c), nil
}

// GetUnseenNotificationsCount returns the count of unread notifications.
// Requires authentication.
//
//	count, err := yt.GetUnseenNotificationsCount()
//	fmt.Printf("You have %d unread notifications\n", count)
func (c *Client) GetUnseenNotificationsCount() (int, error) {
	return c.GetUnseenNotificationsCountCtx(context.Background())
}

// GetUnseenNotificationsCountCtx is GetUnseenNotificationsCount with context.
func (c *Client) GetUnseenNotificationsCountCtx(ctx context.Context) (int, error) {
	if err := c.requireAuth(); err != nil {
		return 0, err
	}
	raw, err := c.request(ctx, epGetUnseenCount, ClientWeb, nil)
	if err != nil {
		return 0, err
	}
	count := int(digFloat(raw, "unseenCount"))
	return count, nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseNotifications(raw map[string]interface{}, c *Client) *NotificationsResult {
	result := &NotificationsResult{client: c}

	actions := digArr(raw, "actions")
	for _, action := range actions {
		am := toMap(action)
		if am == nil {
			continue
		}

		// Continuation token
		if token := digStr(am, "continuationCommand", "token"); token != "" {
			result.Continuation = token
			continue
		}

		// Notification items
		openNotifMenu := digMap(am, "openPopupAction", "popup", "multiPageMenuRenderer")
		if openNotifMenu == nil {
			continue
		}
		for _, section := range digArr(openNotifMenu, "sections") {
			for _, item := range digArr(section, "multiPageMenuNotificationSectionRenderer", "items") {
				notif := parseNotification(item)
				if notif != nil {
					result.Items = append(result.Items, *notif)
				}
			}
		}
	}

	return result
}

func parseNotification(item interface{}) *Notification {
	im := toMap(item)
	if im == nil {
		return nil
	}
	nr := digMap(im, "notificationRenderer")
	if nr == nil {
		return nil
	}

	title := concatRuns(digArr(nr, "shortMessage", "runs"))
	sentTime := digStr(nr, "sentTimeText", "simpleText")
	channelName := firstText(digArr(nr, "contextualMenu", "menuRenderer", "items"))
	notifID := digStr(nr, "notificationId")
	read := !digBool(nr, "unread")
	videoURL := digStr(nr, "navigationEndpoint", "watchEndpoint", "videoId")
	if videoURL != "" {
		videoURL = "https://www.youtube.com/watch?v=" + videoURL
	}

	channelThumbs := parseThumbnails(digArr(nr, "thumbnail", "thumbnails"))
	videoThumbs := parseThumbnails(digArr(nr, "videoThumbnail", "thumbnails"))

	var channelThumb, videoThumb Thumbnail
	if len(channelThumbs) > 0 {
		channelThumb = channelThumbs[0]
	}
	if len(videoThumbs) > 0 {
		videoThumb = videoThumbs[0]
	}

	return &Notification{
		Title:            title,
		SentTime:         sentTime,
		ChannelName:      channelName,
		ChannelThumbnail: channelThumb,
		VideoThumbnail:   videoThumb,
		VideoURL:         videoURL,
		Read:             read,
		NotificationID:   notifID,
	}
}
