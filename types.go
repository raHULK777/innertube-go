package innertube

import "time"

// ─────────────────────────────────────────────
// Common shared types
// ─────────────────────────────────────────────

// Thumbnail represents a single thumbnail image.
type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Duration holds video/track duration in multiple formats.
type Duration struct {
	Seconds         int    `json:"seconds"`
	SimpleText      string `json:"simple_text"`       // e.g. "3:45"
	AccessibilityLabel string `json:"accessibility_label"` // e.g. "3 minutes, 45 seconds"
}

// Channel holds basic channel info embedded in other responses.
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Badge represents a video or channel badge (e.g. "4K", "Live", "Verified").
type Badge struct {
	Text  string `json:"text"`
	Style string `json:"style"`
}

// ActionResult is returned by interaction methods.
type ActionResult struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
}

// ─────────────────────────────────────────────
// Search types
// ─────────────────────────────────────────────

// SearchResult holds results from a YouTube or YouTube Music search.
type SearchResult struct {
	Query            string        `json:"query"`
	CorrectedQuery   string        `json:"corrected_query,omitempty"`
	EstimatedResults int64         `json:"estimated_results"`
	Videos           []VideoResult `json:"videos,omitempty"`
	Continuation     string        `json:"continuation,omitempty"`
}

// VideoResult is a single video item in a search/feed response.
type VideoResult struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Channel     Channel   `json:"channel"`
	Metadata    VideoMeta `json:"metadata"`
}

// VideoMeta holds supplementary info for VideoResult.
type VideoMeta struct {
	ViewCount     string    `json:"view_count"`
	ShortViewText string    `json:"short_view_count_text"`
	Thumbnails    []Thumbnail `json:"thumbnails"`
	Duration      Duration  `json:"duration"`
	Published     string    `json:"published"`
	Badges        []Badge   `json:"badges,omitempty"`
	OwnerBadges   []Badge   `json:"owner_badges,omitempty"`
	IsLive        bool      `json:"is_live"`
}

// SearchSuggestion is a single autocomplete suggestion.
type SearchSuggestion struct {
	Text     string `json:"text"`
	BoldText string `json:"bold_text,omitempty"`
}

// MusicSearchResult holds YouTube Music search results.
type MusicSearchResult struct {
	Query          string              `json:"query"`
	CorrectedQuery string              `json:"corrected_query,omitempty"`
	Results        MusicSearchSections `json:"results"`
}

// MusicSearchSections contains all categorized YTMusic results.
type MusicSearchSections struct {
	TopResult          []interface{}        `json:"top_result,omitempty"`
	Songs              []MusicTrack         `json:"songs,omitempty"`
	Videos             []MusicVideo         `json:"videos,omitempty"`
	Albums             []MusicAlbum         `json:"albums,omitempty"`
	FeaturedPlaylists  []MusicPlaylist      `json:"featured_playlists,omitempty"`
	CommunityPlaylists []MusicPlaylist      `json:"community_playlists,omitempty"`
	Artists            []MusicArtist        `json:"artists,omitempty"`
}

type MusicTrack struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Artist     string      `json:"artist"`
	Album      string      `json:"album"`
	Duration   string      `json:"duration"`
	Thumbnails []Thumbnail `json:"thumbnails"`
}

type MusicVideo struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Views      string      `json:"views"`
	Duration   string      `json:"duration"`
	Thumbnails []Thumbnail `json:"thumbnails"`
}

type MusicAlbum struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Year       string      `json:"year"`
	Thumbnails []Thumbnail `json:"thumbnails"`
}

type MusicPlaylist struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	ChannelID  string `json:"channel_id"`
	TotalItems int    `json:"total_items"`
}

type MusicArtist struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Subscribers string      `json:"subscribers"`
	Thumbnails  []Thumbnail `json:"thumbnails"`
}

// ─────────────────────────────────────────────
// Video / Player types
// ─────────────────────────────────────────────

// VideoDetails holds full metadata for a video.
type VideoDetails struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Thumbnail   Thumbnail     `json:"thumbnail"`
	Metadata    VideoFullMeta `json:"metadata"`

	// client reference for chained calls
	client *Client
}

// VideoFullMeta holds rich metadata extracted from the player response.
type VideoFullMeta struct {
	Embed struct {
		IframeURL      string `json:"iframe_url"`
		FlashURL       string `json:"flash_url"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
		FlashSecureURL string `json:"flash_secure_url"`
	} `json:"embed"`

	Likes    int     `json:"likes"`
	Dislikes int     `json:"dislikes"` // no longer provided by YouTube natively
	ViewCount        int64   `json:"view_count"`
	AverageRating    float64 `json:"average_rating"`
	LengthSeconds    int     `json:"length_seconds"`
	ChannelID        string  `json:"channel_id"`
	ChannelURL       string  `json:"channel_url"`
	ExternalChannelID string `json:"external_channel_id"`
	AllowRatings     bool    `json:"allow_ratings"`
	IsLiveContent    bool    `json:"is_live_content"`
	IsFamilySafe     bool    `json:"is_family_safe"`
	IsUnlisted       bool    `json:"is_unlisted"`
	IsPrivate        bool    `json:"is_private"`
	IsLiked          bool    `json:"is_liked"`
	IsDisliked       bool    `json:"is_disliked"`
	IsSubscribed     bool    `json:"is_subscribed"`
	SubscriberCount  string  `json:"subscriber_count"`
	LikeInfo         struct {
		Count         int    `json:"count"`
		ShortCountText string `json:"short_count_text"`
	} `json:"like_info"`
	PublishDateText              string   `json:"publish_date_text"`
	HasYPCMetadata              bool     `json:"has_ypc_metadata"`
	Category                    string   `json:"category"`
	ChannelName                 string   `json:"channel_name"`
	PublishDate                 string   `json:"publish_date"`
	UploadDate                  string   `json:"upload_date"`
	Keywords                    []string `json:"keywords"`
	NotificationPreference      string   `json:"current_notification_preference"`
}

// GetComments is a convenience method on VideoDetails that fetches comments.
func (v *VideoDetails) GetComments() (*CommentsResult, error) {
	return v.client.GetComments(v.ID)
}

// StreamingData holds all available formats for a video.
type StreamingData struct {
	SelectedFormat Format   `json:"selected_format"`
	Formats        []Format `json:"formats"`
	ExpiresIn      int      `json:"expires_in_seconds,omitempty"`
}

// Format represents a single audio/video stream format.
type Format struct {
	ITag              int    `json:"itag"`
	MimeType          string `json:"mime_type"`
	Bitrate           int    `json:"bitrate"`
	AverageBitrate    int    `json:"average_bitrate,omitempty"`
	Width             int    `json:"width,omitempty"`
	Height            int    `json:"height,omitempty"`
	InitRange         Range  `json:"init_range,omitempty"`
	IndexRange        Range  `json:"index_range,omitempty"`
	LastModified      string `json:"last_modified"`
	ContentLength     string `json:"content_length"`
	Quality           string `json:"quality"`
	QualityLabel      string `json:"quality_label,omitempty"`
	ProjectionType    string `json:"projection_type"`
	FPS               int    `json:"fps,omitempty"`
	HighReplication   bool   `json:"high_replication,omitempty"`
	AudioQuality      string `json:"audio_quality,omitempty"`
	ApproxDurationMs  string `json:"approx_duration_ms"`
	AudioSampleRate   string `json:"audio_sample_rate,omitempty"`
	AudioChannels     int    `json:"audio_channels,omitempty"`
	LoudnessDB        float64 `json:"loudness_db,omitempty"`
	URL               string `json:"url"`
	HasAudio          bool   `json:"has_audio"`
	HasVideo          bool   `json:"has_video"`
}

// Range holds start/end byte offsets for init/index segments.
type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// StreamOptions controls which format to select.
type StreamOptions struct {
	// Format: "mp4", "webm", "audio" — defaults to "mp4"
	Format string

	// Quality: "144p", "240p", "360p", "480p", "720p", "1080p", "1440p", "2160p" — defaults to "360p"
	Quality string

	// Type: "video", "audio", "videoandaudio" — defaults to "videoandaudio"
	Type string
}

// ─────────────────────────────────────────────
// Comments types
// ─────────────────────────────────────────────

// CommentsResult holds a page of comments.
type CommentsResult struct {
	Comments     []Comment `json:"comments"`
	CommentCount string    `json:"comment_count,omitempty"`
	Continuation string    `json:"continuation,omitempty"`

	client  *Client
	videoID string
}

// GetContinuation fetches the next page of comments.
func (r *CommentsResult) GetContinuation() (*CommentsResult, error) {
	if r.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return r.client.getCommentsContinuation(r.videoID, r.Continuation)
}

// Comment is a single comment.
type Comment struct {
	ID       string        `json:"id"`
	Text     string        `json:"text"`
	Author   CommentAuthor `json:"author"`
	Metadata CommentMeta   `json:"metadata"`

	client  *Client
	videoID string
}

// Like likes this comment.
func (c *Comment) Like() error {
	return c.client.likeComment(c.videoID, c.ID, c.Metadata.CommentAction)
}

// Dislike dislikes this comment.
func (c *Comment) Dislike() error {
	return c.client.dislikeComment(c.videoID, c.ID, c.Metadata.DislikeAction)
}

// Reply posts a reply to this comment.
func (c *Comment) Reply(text string) error {
	return c.client.replyToComment(c.videoID, c.ID, text)
}

// GetReplies fetches replies to this comment.
func (c *Comment) GetReplies() (*CommentsResult, error) {
	return c.client.getCommentReplies(c.videoID, c.ID, c.Metadata.ReplyAction)
}

// CommentAuthor holds author info for a comment.
type CommentAuthor struct {
	Name      string      `json:"name"`
	Thumbnail []Thumbnail `json:"thumbnail"`
	ChannelID string      `json:"channel_id"`
}

// CommentMeta holds metadata for a comment.
type CommentMeta struct {
	Published      string `json:"published"`
	IsLiked        bool   `json:"is_liked"`
	IsDisliked     bool   `json:"is_disliked"`
	IsPinned       bool   `json:"is_pinned"`
	IsChannelOwner bool   `json:"is_channel_owner"`
	LikeCount      int    `json:"like_count"`
	ReplyCount     int    `json:"reply_count"`

	// Internal action endpoints for interactions
	CommentAction string `json:"-"`
	DislikeAction string `json:"-"`
	ReplyAction   string `json:"-"`
}

// ─────────────────────────────────────────────
// Feed types (Home, History, Subscriptions)
// ─────────────────────────────────────────────

// HomeFeed holds home feed videos.
type HomeFeed struct {
	Videos       []VideoResult `json:"videos"`
	Continuation string        `json:"continuation,omitempty"`
	client       *Client
}

// GetContinuation fetches the next page of the home feed.
func (h *HomeFeed) GetContinuation() (*HomeFeed, error) {
	if h.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return h.client.getHomeFeedContinuation(h.Continuation)
}

// HistoryFeed holds watch history grouped by date.
type HistoryFeed struct {
	Items        []HistoryGroup `json:"items"`
	Continuation string         `json:"continuation,omitempty"`
	client       *Client
}

// GetContinuation fetches the next page of history.
func (h *HistoryFeed) GetContinuation() (*HistoryFeed, error) {
	if h.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return h.client.getHistoryContinuation(h.Continuation)
}

// HistoryGroup is a date-grouped set of watched videos.
type HistoryGroup struct {
	Date   string        `json:"date"`
	Videos []VideoResult `json:"videos"`
}

// SubscriptionsFeed holds subscriptions feed grouped by date.
type SubscriptionsFeed struct {
	Items        []SubscriptionGroup `json:"items"`
	Continuation string              `json:"continuation,omitempty"`
	client       *Client
}

// GetContinuation fetches the next page of the subscriptions feed.
func (s *SubscriptionsFeed) GetContinuation() (*SubscriptionsFeed, error) {
	if s.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return s.client.getSubscriptionsFeedContinuation(s.Continuation)
}

// SubscriptionGroup is a date-grouped set of subscription videos.
type SubscriptionGroup struct {
	Date   string        `json:"date"`
	Videos []VideoResult `json:"videos"`
}

// ─────────────────────────────────────────────
// Notification types
// ─────────────────────────────────────────────

// NotificationsResult holds a page of notifications.
type NotificationsResult struct {
	Items        []Notification `json:"items"`
	Continuation string         `json:"continuation,omitempty"`
	client       *Client
}

// GetContinuation fetches the next page of notifications.
func (n *NotificationsResult) GetContinuation() (*NotificationsResult, error) {
	if n.Continuation == "" {
		return nil, ErrNoContinuation
	}
	return n.client.getNotificationsContinuation(n.Continuation)
}

// Notification is a single YouTube notification.
type Notification struct {
	Title            string    `json:"title"`
	SentTime         string    `json:"sent_time"`
	ChannelName      string    `json:"channel_name"`
	ChannelThumbnail Thumbnail `json:"channel_thumbnail"`
	VideoThumbnail   Thumbnail `json:"video_thumbnail"`
	VideoURL         string    `json:"video_url"`
	Read             bool      `json:"read"`
	NotificationID   string    `json:"notification_id"`
}

// ─────────────────────────────────────────────
// Playlist types
// ─────────────────────────────────────────────

// Playlist holds playlist metadata and items.
type Playlist struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	TotalItems  interface{}    `json:"total_items"` // string (YT) or int (Music)
	LastUpdated string         `json:"last_updated,omitempty"`
	Views       string         `json:"views,omitempty"`
	Duration    string         `json:"duration,omitempty"`
	Year        string         `json:"year,omitempty"`
	Items       []PlaylistItem `json:"items"`
	Continuation string        `json:"continuation,omitempty"`
}

// PlaylistItem is a single video in a playlist.
type PlaylistItem struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Duration   Duration    `json:"duration"`
	Thumbnails []Thumbnail `json:"thumbnails"`
	SetVideoID string      `json:"set_video_id,omitempty"`
}

// ─────────────────────────────────────────────
// Account types
// ─────────────────────────────────────────────

// AccountInfo holds basic Google account info.
type AccountInfo struct {
	Name     string      `json:"name"`
	Photo    []Thumbnail `json:"photo"`
	Country  string      `json:"country"`
	Language string      `json:"language"`
}

// NotificationPreference options.
const (
	NotifAll          = "ALL"
	NotifNone         = "NONE"
	NotifPersonalized = "PERSONALIZED"
)

// ─────────────────────────────────────────────
// Live chat types
// ─────────────────────────────────────────────

// LiveChat provides access to a live chat stream.
type LiveChat struct {
	videoID         string
	continuation    string
	client          *Client
	stopCh          chan struct{}
	chatUpdateCh    chan ChatMessage
	metaUpdateCh    chan LiveMetadata
}

// ChatMessage is a single live chat message.
type ChatMessage struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	Author    ChatAuthor  `json:"author"`
	Timestamp time.Time   `json:"timestamp"`
	Amount    string      `json:"amount,omitempty"` // for Superchats
	Color     string      `json:"color,omitempty"`  // Superchat color
	IsMember  bool        `json:"is_member"`
	deleteToken string    // internal
}

// Delete deletes this chat message (if you sent it or are a moderator).
func (m *ChatMessage) Delete(lc *LiveChat) error {
	return lc.client.deleteLiveChatMessage(lc.videoID, m.deleteToken)
}

// ChatAuthor holds info about who sent a chat message.
type ChatAuthor struct {
	Name      string      `json:"name"`
	ChannelID string      `json:"channel_id"`
	Thumbnail []Thumbnail `json:"thumbnails"`
	BadgeText string      `json:"badge_text,omitempty"`
}

// LiveMetadata holds real-time stats about a livestream.
type LiveMetadata struct {
	ViewerCount string `json:"viewer_count"`
	LikeCount   string `json:"like_count"`
	Title       string `json:"title"`
}

// OnChatUpdate registers a handler called for each new chat message.
func (lc *LiveChat) OnChatUpdate(fn func(ChatMessage)) {
	go func() {
		for msg := range lc.chatUpdateCh {
			fn(msg)
		}
	}()
}

// OnMetadataUpdate registers a handler called when stream stats update.
func (lc *LiveChat) OnMetadataUpdate(fn func(LiveMetadata)) {
	go func() {
		for meta := range lc.metaUpdateCh {
			fn(meta)
		}
	}()
}

// Stop stops polling the live chat.
func (lc *LiveChat) Stop() {
	close(lc.stopCh)
}

// SendMessage sends a message to the live chat. Returns the sent message.
func (lc *LiveChat) SendMessage(text string) (*ChatMessage, error) {
	return lc.client.sendLiveChatMessage(lc.videoID, text, lc.continuation)
}

// ─────────────────────────────────────────────
// Download types
// ─────────────────────────────────────────────

// DownloadProgress holds state for an in-progress download.
type DownloadProgress struct {
	Percentage    float64
	Downloaded    int64
	Total         int64
	DownloadedMB  float64
	TotalMB       float64
	Speed         float64 // bytes/sec
	ETA           time.Duration
}

// ─────────────────────────────────────────────
// Auth types
// ─────────────────────────────────────────────

// Credentials holds authentication info.
type Credentials struct {
	// For OAuth2
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`

	// For cookie auth
	Cookies string `json:"cookies,omitempty"`

	// Visitor data from YouTube session
	VisitorData string `json:"visitor_data,omitempty"`
}

// IsExpired reports whether the OAuth token has expired.
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt.Add(-60 * time.Second))
}
