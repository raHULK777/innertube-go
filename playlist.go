package innertube

import (
	"context"
	"strings"
)

// GetPlaylist retrieves a playlist's metadata and items.
//
//	playlist, err := yt.GetPlaylist("PLxxxxxx")
//	// For YouTube Music:
//	playlist, err := yt.GetPlaylist("PLxxxxxx", &innertube.SearchOptions{Client: innertube.ClientWebMusic})
func (c *Client) GetPlaylist(playlistID string, opts ...*SearchOptions) (*Playlist, error) {
	return c.GetPlaylistCtx(context.Background(), playlistID, opts...)
}

// GetPlaylistCtx is GetPlaylist with context.
func (c *Client) GetPlaylistCtx(ctx context.Context, playlistID string, opts ...*SearchOptions) (*Playlist, error) {
	opt := resolveSearchOpts(opts)

	if opt.Client == ClientWebMusic {
		return c.getMusicPlaylist(ctx, playlistID)
	}

	payload := map[string]interface{}{
		"browseId": "VL" + playlistID,
		"params":   "wgYCCAA%3D",
	}

	raw, err := c.request(ctx, epBrowse, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	return parsePlaylist(raw, playlistID), nil
}

// getMusicPlaylist fetches a YouTube Music playlist.
func (c *Client) getMusicPlaylist(ctx context.Context, playlistID string) (*Playlist, error) {
	payload := map[string]interface{}{
		"browseId": "VL" + playlistID,
	}
	raw, err := c.requestMusic(ctx, epBrowse, payload)
	if err != nil {
		return nil, err
	}
	return parseMusicPlaylist(raw, playlistID), nil
}

// GetLyrics retrieves lyrics for a YouTube Music track.
// Pass the video/song ID from a music search result.
//
//	search, _ := yt.SearchMusic("Bohemian Rhapsody")
//	lyrics, err := yt.GetLyrics(search.Results.Songs[0].ID)
func (c *Client) GetLyrics(videoID string) (string, error) {
	return c.GetLyricsCtx(context.Background(), videoID)
}

// GetLyricsCtx is GetLyrics with context.
func (c *Client) GetLyricsCtx(ctx context.Context, videoID string) (string, error) {
	// First get the browse ID for lyrics
	payload := map[string]interface{}{
		"videoId": videoID,
	}
	raw, err := c.requestMusic(ctx, epNext, payload)
	if err != nil {
		return "", err
	}

	// Find the lyrics browse endpoint
	lyricsBrowseID := findLyricsBrowseID(raw)
	if lyricsBrowseID == "" {
		return "", nil
	}

	// Browse to the lyrics page
	lyricsPayload := map[string]interface{}{
		"browseId": lyricsBrowseID,
	}
	lyricsRaw, err := c.requestMusic(ctx, epBrowse, lyricsPayload)
	if err != nil {
		return "", err
	}

	return parseLyrics(lyricsRaw), nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parsePlaylist(raw map[string]interface{}, playlistID string) *Playlist {
	pl := &Playlist{ID: playlistID}

	sidebar := digMap(raw, "sidebar", "playlistSidebarRenderer")
	if sidebar != nil {
		for _, item := range digArr(sidebar, "items") {
			primary := digMap(item, "playlistSidebarPrimaryInfoRenderer")
			if primary != nil {
				pl.Title = concatRuns(digArr(primary, "title", "runs"))
				stats := digArr(primary, "stats")
				if len(stats) >= 1 {
					pl.TotalItems = concatRuns(digArr(stats[0], "runs"))
					if pl.TotalItems == "" {
						pl.TotalItems = digStr(toMap(stats[0]), "simpleText")
					}
				}
				if len(stats) >= 2 {
					pl.Views = digStr(toMap(stats[1]), "simpleText")
				}
				if len(stats) >= 3 {
					pl.LastUpdated = digStr(toMap(stats[2]), "simpleText")
					if pl.LastUpdated == "" {
						pl.LastUpdated = concatRuns(digArr(stats[2], "runs"))
					}
				}
			}
			secondary := digMap(item, "playlistSidebarSecondaryInfoRenderer")
			if secondary != nil {
				pl.Description = concatRuns(digArr(secondary, "playlistDescriptionText", "runs"))
			}
		}
	}

	// Playlist header (alternate path)
	if pl.Title == "" {
		if header := digMap(raw, "header", "playlistHeaderRenderer"); header != nil {
			pl.Title = concatRuns(digArr(header, "title", "runs"))
			if pl.Title == "" {
				pl.Title = digStr(header, "title", "simpleText")
			}
		}
	}

	// Videos from the main content area
	tabs := digArr(raw, "contents", "twoColumnBrowseResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		sections := digArr(tabRenderer, "content", "sectionListRenderer", "contents")
		for _, section := range sections {
			if pvr := digMap(section, "itemSectionRenderer"); pvr != nil {
				for _, content := range digArr(pvr, "contents") {
					if plvr := digMap(content, "playlistVideoListRenderer"); plvr != nil {
						for _, videoItem := range digArr(plvr, "contents") {
							vi := parsePlaylistVideoRenderer(videoItem)
							if vi != nil {
								pl.Items = append(pl.Items, *vi)
							}
						}
						// Continuation
						for _, ci := range digArr(plvr, "contents") {
							if token := digStr(ci, "continuationItemRenderer",
								"continuationEndpoint", "continuationCommand", "token"); token != "" {
								pl.Continuation = token
							}
						}
					}
				}
			}
		}
	}

	return pl
}

func parseMusicPlaylist(raw map[string]interface{}, playlistID string) *Playlist {
	pl := &Playlist{ID: playlistID}

	header := digMap(raw, "header", "musicDetailHeaderRenderer")
	if header != nil {
		pl.Title = digStr(header, "title", "runs", "0", "text")
		pl.Description = digStr(header, "description", "runs", "0", "text")
		subtitles := digArr(header, "subtitle", "runs")
		for i, s := range subtitles {
			text := digStr(toMap(s), "text")
			switch i {
			case 2:
				pl.Year = text
			case 4:
				pl.Duration = text
			}
		}
		// Item count from secondSubtitle
		count := digStr(header, "secondSubtitle", "runs", "0", "text")
		pl.TotalItems = count
	}

	// Music playlist items
	contents := digArr(raw, "contents", "singleColumnBrowseResultsRenderer", "tabs")
	for _, tab := range contents {
		sections := digArr(tab, "tabRenderer", "content", "sectionListRenderer", "contents")
		for _, section := range sections {
			for _, item := range digArr(section, "musicShelfRenderer", "contents") {
				vi := parseMusicPlaylistItem2(item)
				if vi != nil {
					pl.Items = append(pl.Items, *vi)
				}
			}
		}
	}

	return pl
}

func parsePlaylistVideoRenderer(item interface{}) *PlaylistItem {
	im := toMap(item)
	if im == nil {
		return nil
	}
	pvr := digMap(im, "playlistVideoRenderer")
	if pvr == nil {
		return nil
	}

	id := digStr(pvr, "videoId")
	if id == "" {
		return nil
	}

	title := concatRuns(digArr(pvr, "title", "runs"))
	if title == "" {
		title = digStr(pvr, "title", "simpleText")
	}

	author := firstText(digArr(pvr, "shortBylineText", "runs"))
	thumbs := parseThumbnails(digArr(pvr, "thumbnail", "thumbnails"))
	setVideoID := digStr(pvr, "setVideoId")

	dur := Duration{}
	if lt := digMap(pvr, "lengthText"); lt != nil {
		dur.SimpleText = digStr(lt, "simpleText")
		if al := digMap(lt, "accessibility", "accessibilityData"); al != nil {
			dur.AccessibilityLabel, _ = al["label"].(string)
		}
	}

	return &PlaylistItem{
		ID:         id,
		Title:      title,
		Author:     author,
		Duration:   dur,
		Thumbnails: thumbs,
		SetVideoID: setVideoID,
	}
}

func parseMusicPlaylistItem2(item interface{}) *PlaylistItem {
	im := toMap(item)
	if im == nil {
		return nil
	}
	r := digMap(im, "musicResponsiveListItemRenderer")
	if r == nil {
		return nil
	}

	id := digStr(r, "playlistItemData", "videoId")
	cols := digArr(r, "flexColumns")
	title := ""
	author := ""
	duration := ""
	if len(cols) > 0 {
		title = concatRuns(digArr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 1 {
		author = firstText(digArr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}
	if len(cols) > 2 {
		duration = concatRuns(digArr(cols[2], "musicResponsiveListItemFlexColumnRenderer", "text", "runs"))
	}

	thumbs := parseThumbnails(digArr(r, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"))

	return &PlaylistItem{
		ID:         id,
		Title:      title,
		Author:     author,
		Duration:   Duration{SimpleText: duration},
		Thumbnails: thumbs,
	}
}

func findLyricsBrowseID(raw map[string]interface{}) string {
	tabs := digArr(raw, "contents", "singleColumnMusicWatchNextResultsRenderer",
		"tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs")
	for _, tab := range tabs {
		tabRenderer := digMap(tab, "tabRenderer")
		if tabRenderer == nil {
			continue
		}
		title := digStr(tabRenderer, "title")
		if !strings.EqualFold(title, "lyrics") {
			continue
		}
		return digStr(tabRenderer, "endpoint", "browseEndpoint", "browseId")
	}
	return ""
}

func parseLyrics(raw map[string]interface{}) string {
	return concatRuns(digArr(raw,
		"contents", "sectionListRenderer", "contents", "0",
		"musicDescriptionShelfRenderer", "description", "runs"))
}
