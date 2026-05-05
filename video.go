package innertube

import (
	"context"
	"fmt"
	"strings"
)

// GetVideoDetails retrieves full video metadata for the given video ID.
//
//	video, err := yt.GetVideoDetails("dQw4w9WgXcQ")
//	fmt.Println(video.Title, video.Metadata.ViewCount)
func (c *Client) GetVideoDetails(videoID string) (*VideoDetails, error) {
	return c.GetVideoDetailsCtx(context.Background(), videoID)
}

// GetVideoDetailsCtx is GetVideoDetails with context.
func (c *Client) GetVideoDetailsCtx(ctx context.Context, videoID string) (*VideoDetails, error) {
	payload := map[string]interface{}{
		"videoId":        videoID,
		"racyCheckOk":    true,
		"contentCheckOk": true,
		"playbackContext": map[string]interface{}{
			"contentPlaybackContext": map[string]interface{}{
				"signatureTimestamp": 19369,
			},
		},
	}

	// Try clients in order — YouTube periodically blocks specific client versions
	var raw map[string]interface{}
	var lastErr error
	for _, ct := range []ClientType{ClientTVEmbedded, ClientAndroid, ClientIOS, ClientWeb} {
		raw, lastErr = c.request(ctx, epPlayer, ct, payload)
		if lastErr != nil {
			continue
		}
		status := digStr(raw, "playabilityStatus", "status")
		if status == "OK" || status == "LIVE_STREAM_OFFLINE" {
			break
		}
		reason := digStr(raw, "playabilityStatus", "reason")
		lastErr = fmt.Errorf("%w: %s", ErrVideoUnavailable, reason)
	}
	if lastErr != nil {
		return nil, lastErr
	}

	// Also fetch /next for additional metadata (likes, comments, related)
	nextPayload := map[string]interface{}{
		"videoId": videoID,
	}
	nextRaw, err := c.request(ctx, epNext, ClientWeb, nextPayload)
	if err != nil {
		nextRaw = nil
	}

	return parseVideoDetails(videoID, raw, nextRaw, c)
}

// GetStreamingData retrieves the deciphered streaming URLs for a video.
//
//	data, err := yt.GetStreamingData("dQw4w9WgXcQ", &innertube.StreamOptions{
//	    Quality: "720p",
//	    Type:    "videoandaudio",
//	})
func (c *Client) GetStreamingData(videoID string, opts *StreamOptions) (*StreamingData, error) {
	return c.GetStreamingDataCtx(context.Background(), videoID, opts)
}

// GetStreamingDataCtx is GetStreamingData with context.
// GetStreamingDataCtx is GetStreamingData with context.
func (c *Client) GetStreamingDataCtx(ctx context.Context, videoID string, opts *StreamOptions) (*StreamingData, error) {
	if opts == nil {
		opts = &StreamOptions{}
	}
	opts.applyDefaults()

	payload := map[string]interface{}{
		"videoId":        videoID,
		"racyCheckOk":    true,
		"contentCheckOk": true,
		"playbackContext": map[string]interface{}{
			"contentPlaybackContext": map[string]interface{}{
				"signatureTimestamp": 19369,
			},
		},
	}

	// Try Android first (direct URLs, no cipher), fall back to iOS then TV embedded
	var raw map[string]interface{}
	var lastErr error
	for _, ct := range []ClientType{ClientAndroid, ClientIOS, ClientTVEmbedded} {
		raw, lastErr = c.request(ctx, epPlayer, ct, payload)
		if lastErr != nil {
			continue
		}
		status := digStr(raw, "playabilityStatus", "status")
		if status == "OK" || status == "LIVE_STREAM_OFFLINE" {
			lastErr = nil
			break
		}
		reason := digStr(raw, "playabilityStatus", "reason")
		lastErr = fmt.Errorf("%w: %s", ErrVideoUnavailable, reason)
	}
	if lastErr != nil {
		return nil, lastErr
	}

	formats, err := parseFormats(raw)
	if err != nil {
		return nil, err
	}

	// Select the best matching format
	selected, err := selectFormat(formats, opts)
	if err != nil {
		return nil, err
	}

	return &StreamingData{
		SelectedFormat: selected,
		Formats:        formats,
	}, nil
}

// GetLivechat returns a LiveChat instance for a live or premiering video.
// Start listening via lc.OnChatUpdate and lc.OnMetadataUpdate.
//
//	lc, err := yt.GetLivechat(video)
//	lc.OnChatUpdate(func(msg innertube.ChatMessage) {
//	    fmt.Printf("%s: %s\n", msg.Author.Name, msg.Text)
//	})
func (c *Client) GetLivechat(video *VideoDetails) (*LiveChat, error) {
	if !video.Metadata.IsLiveContent {
		return nil, ErrLiveChatDisabled
	}

	// Fetch the live chat continuation token from /next
	payload := map[string]interface{}{
		"videoId": video.ID,
	}
	raw, err := c.request(context.Background(), epNext, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	token := extractLiveChatContinuation(raw)
	if token == "" {
		return nil, ErrLiveChatDisabled
	}

	lc := &LiveChat{
		videoID:      video.ID,
		continuation: token,
		client:       c,
		stopCh:       make(chan struct{}),
		chatUpdateCh: make(chan ChatMessage, 100),
		metaUpdateCh: make(chan LiveMetadata, 10),
	}

	go lc.poll()
	return lc, nil
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseVideoDetails(videoID string, playerRaw, nextRaw map[string]interface{}, c *Client) (*VideoDetails, error) {
	if status := digStr(playerRaw, "playabilityStatus", "status"); status == "ERROR" || status == "LOGIN_REQUIRED" {
		reason := digStr(playerRaw, "playabilityStatus", "reason")
		return nil, fmt.Errorf("%w: %s", ErrVideoUnavailable, reason)
	}

	vd := digMap(playerRaw, "videoDetails")
	if vd == nil {
		return nil, ErrVideoUnavailable
	}

	thumbs := parseThumbnails(digArr(vd, "thumbnail", "thumbnails"))
	var bestThumb Thumbnail
	if len(thumbs) > 0 {
		bestThumb = bestThumbnail(thumbs)
	}

	mi := digMap(playerRaw, "microformat", "playerMicroformatRenderer")

	details := &VideoDetails{
		ID:          videoID,
		Title:       digStr(vd, "title"),
		Description: digStr(vd, "shortDescription"),
		Thumbnail:   bestThumb,
		client:      c,
	}

	viewCount := int64(digFloat(vd, "viewCount"))
	lengthSecs := int(digFloat(vd, "lengthSeconds"))
	chanID := digStr(vd, "channelId")
	chanName := digStr(vd, "author")
	keywords := parseStringArray(digArr(vd, "keywords"))
	isLive := digBool(vd, "isLiveContent")
	isPrivate := digBool(vd, "isPrivate")
	isFamilySafe := true
	if mi != nil {
		isFamilySafe = digBool(mi, "isFamilySafe")
	}
	isUnlisted := false
	if mi != nil {
		isUnlisted = digBool(mi, "isUnlisted")
	}
	category := ""
	publishDate := ""
	uploadDate := ""
	if mi != nil {
		category = digStr(mi, "category")
		publishDate = digStr(mi, "publishDate")
		uploadDate = digStr(mi, "uploadDate")
	}

	details.Metadata = VideoFullMeta{
		ViewCount:     viewCount,
		LengthSeconds: lengthSecs,
		ChannelID:     chanID,
		ChannelURL:    "https://www.youtube.com/channel/" + chanID,
		ChannelName:   chanName,
		IsLiveContent: isLive,
		IsPrivate:     isPrivate,
		IsFamilySafe:  isFamilySafe,
		IsUnlisted:    isUnlisted,
		Keywords:      keywords,
		Category:      category,
		PublishDate:   publishDate,
		UploadDate:    uploadDate,
	}

	// Parse embed info
	if mi != nil {
		embedURL := digStr(mi, "embed", "iframeUrl")
		embedW := int(digFloat(mi, "embed", "width"))
		embedH := int(digFloat(mi, "embed", "height"))
		details.Metadata.Embed.IframeURL = embedURL
		details.Metadata.Embed.Width = embedW
		details.Metadata.Embed.Height = embedH
	}

	// Merge data from /next response if available
	if nextRaw != nil {
		mergeNextData(&details.Metadata, nextRaw)
	}

	return details, nil
}

func mergeNextData(meta *VideoFullMeta, nextRaw map[string]interface{}) {
	// Primary info renderer
	pir := digMap(nextRaw,
		"contents", "twoColumnWatchNextResults", "results", "results",
		"contents", "0", "videoPrimaryInfoRenderer")

	// Subscriber count, like button
	if pir != nil {
		// Likes
		topActions := digArr(pir, "videoActions", "menuRenderer", "topLevelButtons")
		for _, btn := range topActions {
			lbvr := digMap(btn, "segmentedLikeDislikeButtonViewModel")
			if lbvr == nil {
				continue
			}
			likeCount := digStr(lbvr, "likeButtonViewModel", "likeButtonViewModel",
				"toggleButtonViewModel", "toggleButtonViewModel",
				"defaultButtonViewModel", "buttonViewModel", "title")
			meta.LikeInfo.ShortCountText = likeCount
		}
		meta.IsLiked = false // only available when authenticated
	}

	// Secondary info renderer (channel, subscribe button)
	sir := digMap(nextRaw,
		"contents", "twoColumnWatchNextResults", "results", "results",
		"contents", "1", "videoSecondaryInfoRenderer")
	if sir != nil {
		owner := digMap(sir, "owner", "videoOwnerRenderer")
		if owner != nil {
			subText := digStr(owner, "subscriberCountText", "simpleText")
			if subText == "" {
				subText = concatRuns(digArr(owner, "subscriberCountText", "runs"))
			}
			meta.SubscriberCount = subText
			meta.ExternalChannelID = digStr(owner, "navigationEndpoint", "browseEndpoint", "browseId")
		}
		meta.IsSubscribed = digBool(sir, "subscribeButton", "subscribeButtonRenderer", "subscribed")
	}
}

// ─── Formats ──────────────────────────────────────────────────────────────────

func parseFormats(raw map[string]interface{}) ([]Format, error) {
	sd := digMap(raw, "streamingData")
	if sd == nil {
		return nil, fmt.Errorf("innertube: no streamingData in player response")
	}

	var formats []Format

	// Combined formats (audio+video, usually lower quality)
	for _, f := range digArr(sd, "formats") {
		if fm := parseFormatItem(f, true, true); fm != nil {
			formats = append(formats, *fm)
		}
	}

	// Adaptive formats (separate audio or video streams)
	for _, f := range digArr(sd, "adaptiveFormats") {
		fm := parseFormatItem(f, false, false)
		if fm == nil {
			continue
		}
		mimeType := strings.ToLower(fm.MimeType)
		fm.HasVideo = strings.HasPrefix(mimeType, "video/")
		fm.HasAudio = strings.HasPrefix(mimeType, "audio/")
		formats = append(formats, *fm)
	}

	return formats, nil
}

func parseFormatItem(item interface{}, hasAudio, hasVideo bool) *Format {
	m, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}

	url := digStr(m, "url")
	// If URL is empty, try signatureCipher (needs decipher — caller must handle)
	if url == "" {
		cipher := digStr(m, "signatureCipher")
		if cipher == "" {
			cipher = digStr(m, "cipher")
		}
		if cipher != "" {
			url = extractURLFromCipher(cipher)
		}
	}

	f := &Format{
		ITag:             int(digFloat(m, "itag")),
		MimeType:         digStr(m, "mimeType"),
		Bitrate:          int(digFloat(m, "bitrate")),
		AverageBitrate:   int(digFloat(m, "averageBitrate")),
		Width:            int(digFloat(m, "width")),
		Height:           int(digFloat(m, "height")),
		LastModified:     digStr(m, "lastModified"),
		ContentLength:    digStr(m, "contentLength"),
		Quality:          digStr(m, "quality"),
		QualityLabel:     digStr(m, "qualityLabel"),
		ProjectionType:   digStr(m, "projectionType"),
		FPS:              int(digFloat(m, "fps")),
		HighReplication:  digBool(m, "highReplication"),
		AudioQuality:     digStr(m, "audioQuality"),
		ApproxDurationMs: digStr(m, "approxDurationMs"),
		AudioSampleRate:  digStr(m, "audioSampleRate"),
		AudioChannels:    int(digFloat(m, "audioChannels")),
		LoudnessDB:       digFloat(m, "loudnessDb"),
		URL:              url,
		HasAudio:         hasAudio,
		HasVideo:         hasVideo,
	}

	if ir := digMap(m, "initRange"); ir != nil {
		f.InitRange = Range{Start: digStr(ir, "start"), End: digStr(ir, "end")}
	}
	if ix := digMap(m, "indexRange"); ix != nil {
		f.IndexRange = Range{Start: digStr(ix, "start"), End: digStr(ix, "end")}
	}

	return f
}

// extractURLFromCipher parses the URL from a signatureCipher query string.
func extractURLFromCipher(cipher string) string {
	// signatureCipher looks like: url=...&s=...&sp=sig
	// For the Android client, URLs are usually direct. This is a fallback.
	for _, part := range strings.Split(cipher, "&") {
		if strings.HasPrefix(part, "url=") {
			decoded := strings.TrimPrefix(part, "url=")
			// Basic URL decoding
			decoded = strings.ReplaceAll(decoded, "%3A", ":")
			decoded = strings.ReplaceAll(decoded, "%2F", "/")
			decoded = strings.ReplaceAll(decoded, "%3F", "?")
			decoded = strings.ReplaceAll(decoded, "%3D", "=")
			decoded = strings.ReplaceAll(decoded, "%26", "&")
			return decoded
		}
	}
	return ""
}

// selectFormat picks the best matching format given StreamOptions.
func selectFormat(formats []Format, opts *StreamOptions) (Format, error) {
	wantVideo := opts.Type == "video" || opts.Type == "videoandaudio"
	wantAudio := opts.Type == "audio" || opts.Type == "videoandaudio"

	var candidates []Format
	for _, f := range formats {
		if wantVideo && wantAudio && !(f.HasVideo && f.HasAudio) {
			continue
		}
		if opts.Type == "video" && !f.HasVideo {
			continue
		}
		if opts.Type == "audio" && !f.HasAudio {
			continue
		}
		if opts.Format != "" && !strings.Contains(strings.ToLower(f.MimeType), opts.Format) {
			continue
		}
		if f.URL == "" {
			continue
		}
		candidates = append(candidates, f)
	}

	if len(candidates) == 0 {
		// Fall back: return any format
		for _, f := range formats {
			if f.URL != "" {
				return f, nil
			}
		}
		return Format{}, ErrNoStreamFound
	}

	// Find exact quality match
	for _, f := range candidates {
		if opts.Quality != "" && strings.EqualFold(f.QualityLabel, opts.Quality) {
			return f, nil
		}
	}

	// Fallback: highest quality
	best := candidates[0]
	for _, f := range candidates[1:] {
		if f.Height > best.Height {
			best = f
		}
	}
	return best, nil
}

func (o *StreamOptions) applyDefaults() {
	if o.Format == "" {
		o.Format = "mp4"
	}
	if o.Quality == "" {
		o.Quality = "360p"
	}
	if o.Type == "" {
		o.Type = "videoandaudio"
	}
}

func parseStringArray(raw []interface{}) []string {
	var out []string
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func extractLiveChatContinuation(raw map[string]interface{}) string {
	contents := digArr(raw,
		"contents", "twoColumnWatchNextResults", "conversationBar",
		"liveChatRenderer", "continuations")
	for _, c := range contents {
		token := digStr(c, "reloadContinuationData", "continuation")
		if token != "" {
			return token
		}
	}
	return ""
}
