package innertube

import "context"

// ChannelInfo holds metadata about a YouTube channel.
type ChannelInfo struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Handle      string      `json:"handle"`
	Description string      `json:"description"`
	Subscribers string      `json:"subscribers"`
	VideoCount  string      `json:"video_count"`
	ViewCount   string      `json:"view_count"`
	Country     string      `json:"country"`
	URL         string      `json:"url"`
	Thumbnails  []Thumbnail `json:"thumbnails"`
	Banner      []Thumbnail `json:"banner"`
	IsVerified  bool        `json:"is_verified"`
}

// ChannelFeed holds a channel's videos tab content.
type ChannelFeed struct {
	ChannelID    string        `json:"channel_id"`
	Videos       []VideoResult `json:"videos"`
	Continuation string        `json:"continuation,omitempty"`
	client       *Client
}

// GetContinuation fetches more videos from a channel feed.
func (f *ChannelFeed) GetContinuation() (*ChannelFeed, error) {
	if f.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return f.client.getChannelFeedContinuation(f.ChannelID, f.Continuation)
}

// GetChannel fetches the info and metadata for a YouTube channel.
//
//	channel, err := yt.GetChannel("UCxxxxxx")
//	// or by handle:
//	channel, err := yt.GetChannel("@MrBeast")
func (c *Client) GetChannel(channelID string) (*ChannelInfo, error) {
	return c.GetChannelCtx(context.Background(), channelID)
}

// GetChannelCtx is GetChannel with context.
func (c *Client) GetChannelCtx(ctx context.Context, channelID string) (*ChannelInfo, error) {
	browseID := channelID
	// Handle @handle format
	if len(channelID) > 0 && channelID[0] == '@' {
		browseID = channelID
	}

	payload := map[string]interface{}{
		"browseId": browseID,
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseChannelInfo(raw, channelID), nil
}

// GetChannelVideos fetches the videos tab of a YouTube channel.
//
//	feed, err := yt.GetChannelVideos("UCxxxxxx")
//	more, err := feed.GetContinuation()
func (c *Client) GetChannelVideos(channelID string) (*ChannelFeed, error) {
	return c.GetChannelVideosCtx(context.Background(), channelID)
}

// GetChannelVideosCtx is GetChannelVideos with context.
func (c *Client) GetChannelVideosCtx(ctx context.Context, channelID string) (*ChannelFeed, error) {
	payload := map[string]interface{}{
		"browseId": channelID,
		"params":   "EgZ2aWRlb3PyBgQKAjoA", // videos tab param
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseChannelFeed(raw, channelID, c), nil
}

// getChannelFeedContinuation fetches more videos from a channel.
func (c *Client) getChannelFeedContinuation(channelID, token string) (*ChannelFeed, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(context.Background(), epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseChannelFeedContinuation(raw, channelID, c), nil
}

// SearchChannel searches within a specific channel.
//
//	results, err := yt.SearchChannel("UCxxxxxx", "cats")
func (c *Client) SearchChannel(channelID, query string) (*SearchResult, error) {
	return c.SearchChannelCtx(context.Background(), channelID, query)
}

// SearchChannelCtx is SearchChannel with context.
func (c *Client) SearchChannelCtx(ctx context.Context, channelID, query string) (*SearchResult, error) {
	payload := map[string]interface{}{
		"browseId": channelID,
		"params":   "EgZzZWFyY2jyBgQKADoA", // search tab param
		"query":    query,
	}
	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseSearchResult(raw, query, c), nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseChannelInfo(raw map[string]interface{}, channelID string) *ChannelInfo {
	info := &ChannelInfo{
		ID:  channelID,
		URL: "https://www.youtube.com/channel/" + channelID,
	}

	// Header — c4TabbedHeaderRenderer (main channel page)
	header := digMap(raw, "header", "c4TabbedHeaderRenderer")
	if header != nil {
		info.Name = digStr(header, "title")
		info.Handle = digStr(header, "channelHandleText", "runs", "0", "text")
		info.Subscribers = digStr(header, "subscriberCountText", "simpleText")
		if info.Subscribers == "" {
			info.Subscribers = concatRuns(digArr(header, "subscriberCountText", "runs"))
		}
		info.Thumbnails = parseThumbnails(digArr(header, "avatar", "thumbnails"))
		info.Banner = parseThumbnails(digArr(header, "banner", "thumbnails"))

		// Verified badge
		for _, badge := range digArr(header, "badges") {
			if digStr(badge, "metadataBadgeRenderer", "style") == "BADGE_STYLE_TYPE_VERIFIED" {
				info.IsVerified = true
			}
		}
	}

	// Metadata from channel metadata renderer
	metadata := digMap(raw, "metadata", "channelMetadataRenderer")
	if metadata != nil {
		if info.Name == "" {
			info.Name = digStr(metadata, "title")
		}
		info.Description = digStr(metadata, "description")
		if info.ID == channelID {
			info.ID = digStr(metadata, "externalId")
		}
		info.Country = digStr(metadata, "country")
	}

	// About tab for video/view counts
	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		if digStr(tabRenderer, "title") == "About" {
			// Extract video count and view count from about section
			for _, section := range digArr(tabRenderer, "content", "sectionListRenderer", "contents") {
				for _, row := range digArr(section, "itemSectionRenderer", "contents",
					"0", "channelAboutFullMetadataRenderer", "viewCountText", "simpleText") {
					info.ViewCount, _ = row.(string)
				}
			}
		}
	}

	return info
}

func parseChannelFeed(raw map[string]interface{}, channelID string, c *Client) *ChannelFeed {
	feed := &ChannelFeed{ChannelID: channelID, client: c}

	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil || !digBool(tabRenderer, "selected") {
			continue
		}

		// Rich grid (newer channel layout)
		sections := digArr(tabRenderer, "content", "richGridRenderer", "contents")
		if len(sections) > 0 {
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
			return feed
		}

		// Fallback: section list renderer
		for _, section := range digArr(tabRenderer, "content", "sectionListRenderer", "contents") {
			for _, item := range digArr(section, "itemSectionRenderer", "contents",
				"0", "gridRenderer", "items") {
				if token := digStr(item, "continuationItemRenderer",
					"continuationEndpoint", "continuationCommand", "token"); token != "" {
					feed.Continuation = token
					continue
				}
				if v := parseGridVideoRenderer(item, c); v != nil {
					feed.Videos = append(feed.Videos, *v)
				}
			}
		}
	}

	return feed
}

func parseChannelFeedContinuation(raw map[string]interface{}, channelID string, c *Client) *ChannelFeed {
	feed := &ChannelFeed{ChannelID: channelID, client: c}
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
