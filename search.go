package innertube

import (
	"context"
	"net/url"
	"strings"
)

// SearchOptions configures a search request.
type SearchOptions struct {
	// Client is the YouTube client to use (ClientWeb or ClientWebMusic).
	// Defaults to ClientWeb.
	Client ClientType

	// Filter controls result type filtering.
	// Possible values: "video", "channel", "playlist", "movie"
	Filter string

	// Sort order: "relevance", "upload_date", "view_count", "rating"
	Sort string
}

// filterParams maps friendly filter names to InnerTube params.
var filterParams = map[string]string{
	"video":    "EgIQAQ%3D%3D",
	"channel":  "EgIQAg%3D%3D",
	"playlist": "EgIQAw%3D%3D",
	"movie":    "EgIQBA%3D%3D",
}

// sortParams maps sort names to InnerTube params.
var sortParams = map[string]string{
	"upload_date": "CAI%3D",
	"view_count":  "CAM%3D",
	"rating":      "CAE%3D",
}

// Search searches YouTube (or YouTube Music) for the given query.
//
//	results, err := yt.Search("never gonna give you up")
//	results, err := yt.Search("Interstellar OST", &SearchOptions{Client: innertube.ClientWebMusic})
func (c *Client) Search(query string, opts ...*SearchOptions) (*SearchResult, error) {
	return c.SearchCtx(context.Background(), query, opts...)
}

// SearchCtx is Search with context.
func (c *Client) SearchCtx(ctx context.Context, query string, opts ...*SearchOptions) (*SearchResult, error) {
	opt := resolveSearchOpts(opts)

	if opt.Client == ClientWebMusic {
		return c.searchMusic(ctx, query)
	}

	payload := map[string]interface{}{
		"query": query,
	}
	if f, ok := filterParams[opt.Filter]; ok {
		payload["params"] = f
	} else if s, ok := sortParams[opt.Sort]; ok {
		payload["params"] = s
	}

	raw, err := c.request(ctx, epSearch, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	return parseSearchResult(raw, query, c), nil
}

// searchMusic performs a YouTube Music search.
func (c *Client) searchMusic(ctx context.Context, query string) (*SearchResult, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	raw, err := c.requestMusic(ctx, epSearch, payload)
	if err != nil {
		return nil, err
	}
	// For music results, we currently return an empty videos list
	// with the raw continuation — callers can use SearchMusic for structured music results.
	return &SearchResult{
		Query: query,
	}, parseRawInto(raw, nil)
}

// SearchMusic performs a YouTube Music search and returns structured music results.
//
//	results, err := yt.SearchMusic("Daft Punk")
func (c *Client) SearchMusic(query string) (*MusicSearchResult, error) {
	return c.SearchMusicCtx(context.Background(), query)
}

// SearchMusicCtx is SearchMusic with context.
func (c *Client) SearchMusicCtx(ctx context.Context, query string) (*MusicSearchResult, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	raw, err := c.requestMusic(ctx, epSearch, payload)
	if err != nil {
		return nil, err
	}
	return parseMusicSearchResult(raw, query), nil
}

// GetSearchSuggestions returns autocomplete suggestions for the given query.
//
//	suggestions, err := yt.GetSearchSuggestions("never gonna")
func (c *Client) GetSearchSuggestions(query string, opts ...*SearchOptions) ([]SearchSuggestion, error) {
	return c.GetSearchSuggestionsCtx(context.Background(), query, opts...)
}

// GetSearchSuggestionsCtx is GetSearchSuggestions with context.
func (c *Client) GetSearchSuggestionsCtx(ctx context.Context, query string, opts ...*SearchOptions) ([]SearchSuggestion, error) {
	opt := resolveSearchOpts(opts)

	suggestURL := "https://suggestqueries-clients6.youtube.com/complete/search"
	if opt.Client == ClientWebMusic {
		suggestURL = "https://music.youtube.com/youtubei/v1/music/get_search_suggestions"
		return c.getMusicSearchSuggestions(ctx, query)
	}

	params := url.Values{
		"client": {"youtube"},
		"hl":     {c.opts.Lang},
		"gl":     {c.opts.Country},
		"ds":     {"yt"},
		"q":      {query},
		"xhr":    {"t"},
	}

	body, err := c.get(ctx, suggestURL, params)
	if err != nil {
		return nil, err
	}

	return parseSuggestions(string(body)), nil
}

// getMusicSearchSuggestions returns search suggestions from YouTube Music.
func (c *Client) getMusicSearchSuggestions(ctx context.Context, query string) ([]SearchSuggestion, error) {
	payload := map[string]interface{}{
		"input": query,
	}
	raw, err := c.requestMusic(ctx, "/music/get_search_suggestions", payload)
	if err != nil {
		return nil, err
	}

	var suggestions []SearchSuggestion
	contents := digArr(raw, "contents")
	for _, c := range contents {
		for _, sc := range digArr(c, "searchSuggestionsSectionRenderer", "contents") {
			scm := toMap(sc)
			if r, ok := scm["searchSuggestionRenderer"].(map[string]interface{}); ok {
				text := concatRuns(digArr(r, "suggestion", "runs"))
				suggestions = append(suggestions, SearchSuggestion{Text: text})
			}
		}
	}
	return suggestions, nil
}

// SearchContinuation fetches additional search results.
//
//	results, err := yt.Search("something")
//	more, err := yt.SearchContinuation(results.Continuation)
func (c *Client) SearchContinuation(token string) (*SearchResult, error) {
	return c.SearchContinuationCtx(context.Background(), token)
}

// SearchContinuationCtx is SearchContinuation with context.
func (c *Client) SearchContinuationCtx(ctx context.Context, token string) (*SearchResult, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}
	raw, err := c.request(ctx, epSearch, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseSearchContinuation(raw, c), nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseSearchResult(raw map[string]interface{}, query string, c *Client) *SearchResult {
	result := &SearchResult{Query: query}

	// Estimated results
	if header := digMap(raw, "header", "searchHeaderRenderer"); header != nil {
		if er := digStr(header, "estimatedResults"); er != "" {
			var n int64
			if _, err := (&strings.Reader{}).Read(nil); err == nil {
				_ = er
			}
			result.EstimatedResults = n
		}
	}

	// Corrected query
	result.CorrectedQuery = digStr(raw, "response", "header", "searchHeaderRenderer", "correctedQuery", "simpleText")

	contents := extractSearchContents(raw)
	for _, item := range contents {
		if v := parseVideoRenderer(item, c); v != nil {
			result.Videos = append(result.Videos, *v)
		}
	}

	// Continuation token
	result.Continuation = extractContinuationToken(raw)

	return result
}

func parseSearchContinuation(raw map[string]interface{}, c *Client) *SearchResult {
	result := &SearchResult{}
	contents := extractSearchContentsContinuation(raw)
	for _, item := range contents {
		if v := parseVideoRenderer(item, c); v != nil {
			result.Videos = append(result.Videos, *v)
		}
	}
	result.Continuation = extractContinuationToken(raw)
	return result
}

func parseMusicSearchResult(raw map[string]interface{}, query string) *MusicSearchResult {
	result := &MusicSearchResult{
		Query: query,
	}
	// Music search sections are under contents > tabbedSearchResultsRenderer > tabs > tabRenderer > content
	tabs := digArr(raw, "contents", "tabbedSearchResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		sections := digArr(tabRenderer, "content", "sectionListRenderer", "contents")
		for _, section := range sections {
			sr := digMap(section, "musicShelfRenderer")
			if sr == nil {
				continue
			}
			title := concatRuns(digArr(sr, "title", "runs"))
			items := digArr(sr, "contents")

			switch strings.ToLower(title) {
			case "songs", "top result":
				for _, item := range items {
					if t := parseMusicTrack(item); t != nil {
						result.Results.Songs = append(result.Results.Songs, *t)
					}
				}
			case "videos":
				for _, item := range items {
					if v := parseMusicVideoItem(item); v != nil {
						result.Results.Videos = append(result.Results.Videos, *v)
					}
				}
			case "albums", "singles":
				for _, item := range items {
					if a := parseMusicAlbum(item); a != nil {
						result.Results.Albums = append(result.Results.Albums, *a)
					}
				}
			case "featured playlists":
				for _, item := range items {
					if p := parseMusicPlaylistItem(item); p != nil {
						result.Results.FeaturedPlaylists = append(result.Results.FeaturedPlaylists, *p)
					}
				}
			case "community playlists":
				for _, item := range items {
					if p := parseMusicPlaylistItem(item); p != nil {
						result.Results.CommunityPlaylists = append(result.Results.CommunityPlaylists, *p)
					}
				}
			case "artists":
				for _, item := range items {
					if a := parseMusicArtistItem(item); a != nil {
						result.Results.Artists = append(result.Results.Artists, *a)
					}
				}
			}
		}
	}
	return result
}

func parseMusicTrack(item interface{}) *MusicTrack {
	r := digMap(item, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}
	id := digStr(r, "playlistItemData", "videoId")
	if id == "" {
		// try overlay
		id = digStr(r, "overlay", "musicItemThumbnailOverlayRenderer", "content",
			"musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
	}
	cols := digArr(r, "flexColumns")
	title := ""
	artist := ""
	album := ""
	duration := ""
	if len(cols) > 0 {
		title = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		runs := digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
		artist = firstText(runs)
		if len(runs) > 2 {
			album = digStr(toMap(runs[2]), "text")
		}
	}
	if len(cols) > 2 {
		duration = concatRuns(digArr(cols[2], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	thumbs := parseThumbnails(digArr(r, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))

	return &MusicTrack{
		ID:         id,
		Title:      title,
		Artist:     artist,
		Album:      album,
		Duration:   duration,
		Thumbnails: thumbs,
	}
}

func parseMusicVideoItem(item interface{}) *MusicVideo {
	r := digMap(item, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}
	id := digStr(r, "playlistItemData", "videoId")
	cols := digArr(r, "flexColumns")
	title := ""
	author := ""
	views := ""
	duration := ""
	if len(cols) > 0 {
		title = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		runs := digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
		author = firstText(runs)
		if len(runs) > 2 {
			views = digStr(toMap(runs[2]), "text")
		}
	}
	if len(cols) > 2 {
		duration = concatRuns(digArr(cols[2], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	thumbs := parseThumbnails(digArr(r, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))
	return &MusicVideo{
		ID: id, Title: title, Author: author, Views: views, Duration: duration, Thumbnails: thumbs,
	}
}

func parseMusicAlbum(item interface{}) *MusicAlbum {
	r := digMap(item, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}
	id := digStr(r, "overlay", "musicItemThumbnailOverlayRenderer", "content",
		"musicPlayButtonRenderer", "playNavigationEndpoint", "watchPlaylistEndpoint", "playlistId")
	cols := digArr(r, "flexColumns")
	title := ""
	author := ""
	year := ""
	if len(cols) > 0 {
		title = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		runs := digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
		author = firstText(runs)
		if len(runs) > 2 {
			year = digStr(toMap(runs[2]), "text")
		}
	}
	thumbs := parseThumbnails(digArr(r, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))
	return &MusicAlbum{ID: id, Title: title, Author: author, Year: year, Thumbnails: thumbs}
}

func parseMusicPlaylistItem(item interface{}) *MusicPlaylist {
	r := digMap(item, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}
	id := digStr(r, "overlay", "musicItemThumbnailOverlayRenderer", "content",
		"musicPlayButtonRenderer", "playNavigationEndpoint", "watchPlaylistEndpoint", "playlistId")
	cols := digArr(r, "flexColumns")
	title := ""
	author := ""
	if len(cols) > 0 {
		title = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		author = firstText(digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	return &MusicPlaylist{ID: id, Title: title, Author: author}
}

func parseMusicArtistItem(item interface{}) *MusicArtist {
	r := digMap(item, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}
	id := digStr(r, "navigationEndpoint", "browseEndpoint", "browseId")
	cols := digArr(r, "flexColumns")
	name := ""
	subscribers := ""
	if len(cols) > 0 {
		name = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		subscribers = firstText(digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	thumbs := parseThumbnails(digArr(r, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))
	return &MusicArtist{ID: id, Name: name, Subscribers: subscribers, Thumbnails: thumbs}
}

func parseSuggestions(raw string) []SearchSuggestion {
	// YouTube returns JSONP-ish: window.google.ac.h(["query",[["suggestion",0,[131]],...],...])
	// We'll parse the inner JSON array.
	var suggestions []SearchSuggestion
	start := strings.Index(raw, ",[")
	if start < 0 {
		return suggestions
	}
	raw = raw[start+1:]
	end := strings.LastIndex(raw, "]")
	if end < 0 {
		return suggestions
	}
	raw = raw[:end+1]

	// raw is now like [["text1",0,...],["text2",0,...],...]
	// quick parse
	entries := strings.Split(raw, `["`)
	for _, e := range entries[1:] {
		idx := strings.Index(e, `"`)
		if idx < 0 {
			continue
		}
		text := e[:idx]
		if text != "" {
			suggestions = append(suggestions, SearchSuggestion{Text: text})
		}
	}
	return suggestions
}

// ─── Shared renderer parsers ─────────────────────────────────────────────────

func parseVideoRenderer(item interface{}, c *Client) *VideoResult {
	m, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}

	vr, ok := m["videoRenderer"].(map[string]interface{})
	if !ok {
		return nil
	}

	id := digStr(vr, "videoId")
	if id == "" {
		return nil
	}

	title := concatRuns(digArr(vr, "title", "runs"))
	if title == "" {
		title = digStr(vr, "title", "simpleText")
	}

	desc := concatRuns(digArr(vr, "descriptionSnippet", "runs"))

	chanID := digStr(vr, "ownerText", "runs", "0", "navigationEndpoint", "browseEndpoint", "browseId")
	chanName := firstText(digArr(vr, "ownerText", "runs"))

	thumbs := parseThumbnails(digArr(vr, "thumbnail", "thumbnails"))

	viewCount := digStr(vr, "viewCountText", "simpleText")
	if viewCount == "" {
		viewCount = concatRuns(digArr(vr, "viewCountText", "runs"))
	}

	shortViewText := digStr(vr, "shortViewCountText", "simpleText")

	published := digStr(vr, "publishedTimeText", "simpleText")

	durRaw := digMap(vr, "lengthText")
	var dur Duration
	if durRaw != nil {
		dur.SimpleText = digStr(vr, "lengthText", "simpleText")
		if al := digMap(vr, "lengthText", "accessibility", "accessibilityData"); al != nil {
			dur.AccessibilityLabel, _ = al["label"].(string)
		}
	}

	badges := parseBadges(digArr(vr, "badges"))
	ownerBadges := parseBadges(digArr(vr, "ownerBadges"))

	isLive := false
	for _, b := range badges {
		if strings.ToLower(b.Style) == "live" || strings.EqualFold(b.Text, "live") {
			isLive = true
		}
	}

	return &VideoResult{
		ID:          id,
		URL:         "https://www.youtube.com/watch?v=" + id,
		Title:       title,
		Description: desc,
		Channel: Channel{
			ID:   chanID,
			Name: chanName,
			URL:  "https://www.youtube.com/channel/" + chanID,
		},
		Metadata: VideoMeta{
			ViewCount:     viewCount,
			ShortViewText: shortViewText,
			Thumbnails:    thumbs,
			Duration:      dur,
			Published:     published,
			Badges:        badges,
			OwnerBadges:   ownerBadges,
			IsLive:        isLive,
		},
	}
}

// extractSearchContents pulls the video renderer items from a search response.
func extractSearchContents(raw map[string]interface{}) []interface{} {
	// Path: contents > twoColumnSearchResultsRenderer > primaryContents > sectionListRenderer > contents
	sections := digArr(raw,
		"contents", "twoColumnSearchResultsRenderer",
		"primaryContents", "sectionListRenderer", "contents")

	var items []interface{}
	for _, s := range sections {
		sm, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		ir := digArr(sm, "itemSectionRenderer", "contents")
		items = append(items, ir...)
	}
	return items
}

// extractSearchContentsContinuation handles continuation response structure.
func extractSearchContentsContinuation(raw map[string]interface{}) []interface{} {
	sections := digArr(raw, "onResponseReceivedCommands")
	var items []interface{}
	for _, s := range sections {
		sm := toMap(s)
		appendAction := digMap(sm, "appendContinuationItemsAction")
		if appendAction == nil {
			continue
		}
		for _, item := range digArr(appendAction, "continuationItems") {
			ir := digArr(item, "itemSectionRenderer", "contents")
			items = append(items, ir...)
		}
	}
	return items
}

// extractContinuationToken extracts the continuation token from any response.
func extractContinuationToken(raw map[string]interface{}) string {
	// Search in common locations
	// 1. Contents > ... > continuationItemRenderer > continuationEndpoint
	sections := digArr(raw,
		"contents", "twoColumnSearchResultsRenderer",
		"primaryContents", "sectionListRenderer", "contents")
	for _, s := range sections {
		if token := findContinuationInSection(s); token != "" {
			return token
		}
	}
	// 2. onResponseReceivedCommands
	for _, cmd := range digArr(raw, "onResponseReceivedCommands") {
		items := digArr(cmd, "appendContinuationItemsAction", "continuationItems")
		for _, item := range items {
			if token := findContinuationInSection(item); token != "" {
				return token
			}
		}
	}
	return ""
}

func findContinuationInSection(s interface{}) string {
	if token := digStr(s,
		"continuationItemRenderer", "continuationEndpoint", "continuationCommand", "token"); token != "" {
		return token
	}
	return ""
}

func resolveSearchOpts(opts []*SearchOptions) *SearchOptions {
	if len(opts) == 0 || opts[0] == nil {
		return &SearchOptions{Client: ClientWeb}
	}
	o := *opts[0]
	if o.Client == "" {
		o.Client = ClientWeb
	}
	return &o
}

func toMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func parseRawInto(_ map[string]interface{}, _ interface{}) error {
	return nil
}
