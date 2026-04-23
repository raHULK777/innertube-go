package innertube

import "context"

// GetHomeFeed returns the YouTube home feed (recommendations).
// Authentication is not required but improves results.
//
//	feed, err := yt.GetHomeFeed()
//	more, err := feed.GetContinuation()
func (c *Client) GetHomeFeed() (*HomeFeed, error) {
	return c.GetHomeFeedCtx(context.Background())
}

// GetHomeFeedCtx is GetHomeFeed with context.
func (c *Client) GetHomeFeedCtx(ctx context.Context) (*HomeFeed, error) {
	payload := map[string]interface{}{
		"browseId": "FEwhat_to_watch",
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseHomeFeed(raw, c), nil
}

// getHomeFeedContinuation fetches the next page of the home feed.
func (c *Client) getHomeFeedContinuation(token string) (*HomeFeed, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(context.Background(), epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseHomeFeedContinuation(raw, c), nil
}

// GetHistory returns the authenticated user's watch history.
// Requires authentication.
//
//	history, err := yt.GetHistory()
func (c *Client) GetHistory() (*HistoryFeed, error) {
	return c.GetHistoryCtx(context.Background())
}

// GetHistoryCtx is GetHistory with context.
func (c *Client) GetHistoryCtx(ctx context.Context) (*HistoryFeed, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"browseId": "FEhistory",
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseHistoryFeed(raw, c), nil
}

// getHistoryContinuation fetches the next page of history.
func (c *Client) getHistoryContinuation(token string) (*HistoryFeed, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(context.Background(), epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseHistoryFeedContinuation(raw, c), nil
}

// GetSubscriptionsFeed returns the authenticated user's subscriptions feed.
// Requires authentication.
//
//	feed, err := yt.GetSubscriptionsFeed()
func (c *Client) GetSubscriptionsFeed() (*SubscriptionsFeed, error) {
	return c.GetSubscriptionsFeedCtx(context.Background())
}

// GetSubscriptionsFeedCtx is GetSubscriptionsFeed with context.
func (c *Client) GetSubscriptionsFeedCtx(ctx context.Context) (*SubscriptionsFeed, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"browseId": "FEsubscriptions",
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseSubscriptionsFeed(raw, c), nil
}

// getSubscriptionsFeedContinuation fetches the next page.
func (c *Client) getSubscriptionsFeedContinuation(token string) (*SubscriptionsFeed, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(context.Background(), epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseSubscriptionsFeedContinuation(raw, c), nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseHomeFeed(raw map[string]interface{}, c *Client) *HomeFeed {
	feed := &HomeFeed{client: c}

	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil || !digBool(tabRenderer, "selected") {
			continue
		}
		sections := digArr(tabRenderer, "content", "richGridRenderer", "contents")
		for _, section := range sections {
			sm := toMap(section)
			if sm == nil {
				continue
			}
			if token := digStr(sm, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			if v := parseRichItemRenderer(section, c); v != nil {
				feed.Videos = append(feed.Videos, *v)
			}
		}
	}
	return feed
}

func parseHomeFeedContinuation(raw map[string]interface{}, c *Client) *HomeFeed {
	feed := &HomeFeed{client: c}
	for _, cmd := range digArr(raw, "onResponseReceivedActions") {
		for _, item := range digArr(cmd, "appendContinuationItemsAction", "continuationItems") {
			im := toMap(item)
			if im == nil {
				continue
			}
			if token := digStr(im, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			if v := parseRichItemRenderer(item, c); v != nil {
				feed.Videos = append(feed.Videos, *v)
			}
		}
	}
	return feed
}

func parseHistoryFeed(raw map[string]interface{}, c *Client) *HistoryFeed {
	feed := &HistoryFeed{client: c}

	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		sections := digArr(tabRenderer, "content", "sectionListRenderer", "contents")
		for _, section := range sections {
			sm := toMap(section)
			if sm == nil {
				continue
			}
			if token := digStr(sm, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			group := parseHistoryGroup(section, c)
			if group != nil {
				feed.Items = append(feed.Items, *group)
			}
		}
	}
	return feed
}

func parseHistoryFeedContinuation(raw map[string]interface{}, c *Client) *HistoryFeed {
	feed := &HistoryFeed{client: c}
	for _, cmd := range digArr(raw, "onResponseReceivedActions") {
		for _, item := range digArr(cmd, "appendContinuationItemsAction", "continuationItems") {
			im := toMap(item)
			if im == nil {
				continue
			}
			if token := digStr(im, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			group := parseHistoryGroup(item, c)
			if group != nil {
				feed.Items = append(feed.Items, *group)
			}
		}
	}
	return feed
}

func parseHistoryGroup(section interface{}, c *Client) *HistoryGroup {
	sm := toMap(section)
	if sm == nil {
		return nil
	}
	isr := digMap(sm, "itemSectionRenderer")
	if isr == nil {
		return nil
	}

	group := &HistoryGroup{}

	contents := digArr(isr, "contents")
	for _, content := range contents {
		cm := toMap(content)
		if cm == nil {
			continue
		}
		// Date header
		if header := digMap(cm, "videoRenderer"); header == nil {
			if title := digStr(cm, "shelfRenderer", "title", "runs", "0", "text"); title != "" {
				group.Date = title
			}
		}
		if v := parseVideoRenderer(content, c); v != nil {
			group.Videos = append(group.Videos, *v)
		}
	}

	if len(group.Videos) == 0 {
		return nil
	}
	return group
}

func parseSubscriptionsFeed(raw map[string]interface{}, c *Client) *SubscriptionsFeed {
	feed := &SubscriptionsFeed{client: c}

	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		sections := digArr(tabRenderer, "content", "sectionListRenderer", "contents")
		for _, section := range sections {
			sm := toMap(section)
			if sm == nil {
				continue
			}
			if token := digStr(sm, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			group := parseSubscriptionsGroup(section, c)
			if group != nil {
				feed.Items = append(feed.Items, *group)
			}
		}
	}
	return feed
}

func parseSubscriptionsFeedContinuation(raw map[string]interface{}, c *Client) *SubscriptionsFeed {
	feed := &SubscriptionsFeed{client: c}
	for _, cmd := range digArr(raw, "onResponseReceivedActions") {
		for _, item := range digArr(cmd, "appendContinuationItemsAction", "continuationItems") {
			im := toMap(item)
			if im == nil {
				continue
			}
			if token := digStr(im, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				feed.Continuation = token
				continue
			}
			group := parseSubscriptionsGroup(item, c)
			if group != nil {
				feed.Items = append(feed.Items, *group)
			}
		}
	}
	return feed
}

func parseSubscriptionsGroup(section interface{}, c *Client) *SubscriptionGroup {
	sm := toMap(section)
	if sm == nil {
		return nil
	}
	isr := digMap(sm, "itemSectionRenderer")
	if isr == nil {
		return nil
	}

	group := &SubscriptionGroup{}

	for _, content := range digArr(isr, "contents") {
		// Date shelf
		if shelf := digMap(content, "shelfRenderer"); shelf != nil {
			group.Date = digStr(shelf, "title", "runs", "0", "text")
			for _, item := range digArr(shelf, "content", "gridRenderer", "items") {
				if v := parseGridVideoRenderer(item, c); v != nil {
					group.Videos = append(group.Videos, *v)
				}
			}
		}
	}

	if len(group.Videos) == 0 {
		return nil
	}
	return group
}

// parseRichItemRenderer handles the richItemRenderer format used in home feed.
func parseRichItemRenderer(item interface{}, c *Client) *VideoResult {
	im := toMap(item)
	if im == nil {
		return nil
	}
	content := digMap(im, "richItemRenderer", "content")
	if content == nil {
		return nil
	}
	return parseVideoRenderer(content, c)
}

// parseGridVideoRenderer handles the gridVideoRenderer format.
func parseGridVideoRenderer(item interface{}, c *Client) *VideoResult {
	im := toMap(item)
	if im == nil {
		return nil
	}
	gvr := digMap(im, "gridVideoRenderer")
	if gvr == nil {
		return nil
	}

	id := digStr(gvr, "videoId")
	if id == "" {
		return nil
	}

	title := digStr(gvr, "title", "simpleText")
	if title == "" {
		title = concatRuns(digArr(gvr, "title", "runs"))
	}
	chanName := digStr(gvr, "shortBylineText", "runs", "0", "text")
	chanID := digStr(gvr, "shortBylineText", "runs", "0",
		"navigationEndpoint", "browseEndpoint", "browseId")
	viewCount := digStr(gvr, "viewCountText", "simpleText")
	published := digStr(gvr, "publishedTimeText", "simpleText")
	thumbs := parseThumbnails(digArr(gvr, "thumbnail", "thumbnails"))

	return &VideoResult{
		ID:    id,
		URL:   "https://www.youtube.com/watch?v=" + id,
		Title: title,
		Channel: Channel{
			ID:   chanID,
			Name: chanName,
			URL:  "https://www.youtube.com/channel/" + chanID,
		},
		Metadata: VideoMeta{
			ViewCount:  viewCount,
			Published:  published,
			Thumbnails: thumbs,
		},
	}
}
