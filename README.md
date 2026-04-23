# innertube-go

A full-featured Go client for YouTube's private **InnerTube API** — the same
API YouTube itself uses internally across all its clients.

**No API key required.** No scraping. No third-party video libraries.

Heavily inspired by [haxzie/innerTube.js](https://github.com/haxzie/innerTube.js)
and [LuanRT/YouTube.js](https://github.com/LuanRT/YouTube.js), rewritten from
scratch in idiomatic Go.

---

## Features

| Feature | Auth Required |
|---|---|
| Search YouTube videos | No |
| Search YouTube Music (songs, albums, artists, playlists) | No |
| Search suggestions / autocomplete | No |
| Get full video metadata & player info | No |
| Get all streaming format URLs | No |
| Download videos with real-time progress | No |
| Download audio-only or video-only streams | No |
| Get video comments + replies | No |
| Get playlist metadata & items | No |
| Get channel info, videos, search | No |
| Get home feed | No |
| Get lyrics (YouTube Music) | No |
| Fetch live chat in real time | No |
| Send live chat messages | **Yes** |
| Like / Dislike / Remove Like | **Yes** |
| Subscribe / Unsubscribe | **Yes** |
| Post comments & replies | **Yes** |
| Change notification preferences | **Yes** |
| Get watch history | **Yes** |
| Get subscriptions feed | **Yes** |
| Get notifications | **Yes** |
| Account info & settings | **Yes** |
| OAuth2 device-flow sign-in | — |
| Cookie-based sign-in | — |
| Automatic OAuth2 token refresh | — |

---

## Installation

```bash
go get github.com/raHULK777/innertube-go
```

Requires **Go 1.22+**.

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/slick/innertube-go"
)

func main() {
    yt, err := innertube.New()
    if err != nil {
        log.Fatal(err)
    }

    results, err := yt.Search("never gonna give you up")
    if err != nil {
        log.Fatal(err)
    }

    for _, v := range results.Videos[:5] {
        fmt.Printf("%s — %s (%s)\n", v.Title, v.Channel.Name, v.Metadata.Duration.SimpleText)
    }
}
```

---

## Usage

### Creating a Client

```go
// Default client (unauthenticated, English, US region)
yt, err := innertube.New()

// Custom options
yt, err := innertube.NewWithOptions(&innertube.Options{
    Lang:       "ja",        // Japanese results
    Country:    "JP",
    SafeSearch: true,
})
```

---

### Search

```go
// YouTube search
results, err := yt.Search("golang tutorial")

// With filters
results, err := yt.Search("golang tutorial", &innertube.SearchOptions{
    Filter: "video",                  // "video", "channel", "playlist", "movie"
    Sort:   "upload_date",            // "relevance", "upload_date", "view_count", "rating"
})

// YouTube Music search
musicResults, err := yt.SearchMusic("Daft Punk")
fmt.Println(musicResults.Results.Songs)
fmt.Println(musicResults.Results.Albums)
fmt.Println(musicResults.Results.Artists)

// Autocomplete suggestions
suggestions, err := yt.GetSearchSuggestions("never gonna")
// → [{Text: "never gonna give you up"}, ...]

// Next page of results
more, err := yt.SearchContinuation(results.Continuation)
```

---

### Video Details

```go
video, err := yt.GetVideoDetails("dQw4w9WgXcQ")

fmt.Println(video.Title)
fmt.Println(video.Metadata.ViewCount)
fmt.Println(video.Metadata.ChannelName)
fmt.Println(video.Metadata.LengthSeconds)
fmt.Println(video.Metadata.PublishDate)
fmt.Println(video.Metadata.IsLiveContent)
fmt.Println(video.Metadata.Keywords)

// Chained: get comments directly from video
comments, err := video.GetComments()
```

---

### Comments

```go
comments, err := yt.GetComments("VIDEO_ID")

fmt.Println(comments.CommentCount)
for _, c := range comments.Comments {
    fmt.Printf("%s: %s\n", c.Author.Name, c.Text)
    fmt.Printf("  Likes: %d  Replies: %d\n", c.Metadata.LikeCount, c.Metadata.ReplyCount)
}

// Replies
replies, err := comments.Comments[0].GetReplies()

// Next page
more, err := comments.GetContinuation()

// Interact with comments (auth required)
comments.Comments[0].Like()
comments.Comments[0].Reply("Great comment!")
```

---

### Downloading Videos

```go
// Full download with progress
err := yt.Download("VIDEO_ID", &innertube.DownloadOptions{
    StreamOptions: innertube.StreamOptions{
        Quality: "720p",       // "144p", "240p", "360p", "480p", "720p", "1080p"
        Type:    "videoandaudio", // "video", "audio", "videoandaudio"
        Format:  "mp4",        // "mp4", "webm"
    },
    OutputPath: "./video.mp4",
    OnInfo: func(details *innertube.VideoDetails, f innertube.Format, all []innertube.Format) {
        fmt.Printf("Downloading: %s\n", details.Title)
    },
    OnProgress: func(p innertube.DownloadProgress) {
        fmt.Printf("\r%.1f%%  %.1f/%.1fMB  (%.0fKB/s, ETA %v)",
            p.Percentage, p.DownloadedMB, p.TotalMB, p.Speed/1024, p.ETA)
    },
    OnComplete: func() { fmt.Println("\nDone!") },
})

// Audio-only
err := yt.DownloadAudioOnly("VIDEO_ID", "./audio.webm", onProgress)

// Video-only (no audio)
err := yt.DownloadVideoOnly("VIDEO_ID", "1080p", "./video.mp4", onProgress)

// Get direct URL without downloading
url, err := yt.GetDownloadURL("VIDEO_ID", &innertube.StreamOptions{Quality: "360p"})

// List all available formats
formats, err := yt.ListFormats("VIDEO_ID")
for _, f := range formats {
    fmt.Println(innertube.FormatSummary(f))
}

// Get raw streaming data
streamData, err := yt.GetStreamingData("VIDEO_ID", &innertube.StreamOptions{
    Quality: "1080p",
    Type:    "video",
})
// streamData.SelectedFormat.URL  → direct HTTPS URL
// streamData.Formats             → all available formats

// Download to any io.Writer (pipe to stdout, network, etc.)
err := yt.DownloadToWriter(ctx, "VIDEO_ID", opts, os.Stdout)

// Cancellable download
ctx, cancel := context.WithCancel(context.Background())
err := yt.DownloadCtx(ctx, "VIDEO_ID", opts)
cancel() // cancels the download mid-flight
```

---

### Playlists

```go
// YouTube playlist
playlist, err := yt.GetPlaylist("PLxxxxxx")
fmt.Println(playlist.Title)
fmt.Println(playlist.TotalItems)
for _, item := range playlist.Items {
    fmt.Printf("[%s] %s — %s\n", item.ID, item.Title, item.Author)
}

// YouTube Music playlist / album
playlist, err := yt.GetPlaylist("PLxxxxxx", &innertube.SearchOptions{
    Client: innertube.ClientWebMusic,
})
```

---

### Channels

```go
// By channel ID or @handle
channel, err := yt.GetChannel("UCxxxxxx")
channel, err := yt.GetChannel("@MrBeast")

fmt.Println(channel.Name)
fmt.Println(channel.Subscribers)
fmt.Println(channel.IsVerified)

// Channel videos
feed, err := yt.GetChannelVideos("UCxxxxxx")
more, err := feed.GetContinuation()

// Search within a channel
results, err := yt.SearchChannel("UCxxxxxx", "cats")
```

---

### Feeds

```go
// Home feed (recommendations)
feed, err := yt.GetHomeFeed()
more, err := feed.GetContinuation()

// Watch history (auth required)
history, err := yt.GetHistory()
for _, group := range history.Items {
    fmt.Printf("%s: %d videos\n", group.Date, len(group.Videos))
}
more, err := history.GetContinuation()

// Subscriptions feed (auth required)
subs, err := yt.GetSubscriptionsFeed()
more, err := subs.GetContinuation()
```

---

### Notifications (auth required)

```go
notifs, err := yt.GetNotifications()
count, err := yt.GetUnseenNotificationsCount()

for _, n := range notifs.Items {
    fmt.Printf("[%v] %s — %s\n", n.Read, n.ChannelName, n.Title)
}
more, err := notifs.GetContinuation()
```

---

### Live Chat

```go
video, err := yt.GetVideoDetails("LIVE_VIDEO_ID")
lc, err := yt.GetLivechat(video)

lc.OnChatUpdate(func(msg innertube.ChatMessage) {
    fmt.Printf("%s: %s\n", msg.Author.Name, msg.Text)
    if msg.Amount != "" {
        fmt.Printf("  💰 Superchat: %s\n", msg.Amount)
    }
})

lc.OnMetadataUpdate(func(meta innertube.LiveMetadata) {
    fmt.Printf("Viewers: %s\n", meta.ViewerCount)
})

// Send a message (auth required)
sent, err := lc.SendMessage("Hello from Go!")

// Delete a message
sent.Delete(lc)

// Stop polling
lc.Stop()
```

---

### Interactions (all require auth)

```go
yt.Interact.Subscribe("CHANNEL_ID")
yt.Interact.Unsubscribe("CHANNEL_ID")

yt.Interact.Like("VIDEO_ID")
yt.Interact.Dislike("VIDEO_ID")
yt.Interact.RemoveLike("VIDEO_ID")

yt.Interact.Comment("VIDEO_ID", "Great video!")

// Notification preferences: innertube.NotifAll | NotifNone | NotifPersonalized
yt.Interact.ChangeNotificationPreferences("CHANNEL_ID", innertube.NotifAll)
```

---

### Account Settings (auth required)

```go
info, err := yt.Account.Info()
fmt.Println(info.Name, info.Country)

// Notifications
yt.Account.Settings.Notifications.SetSubscriptions(true)
yt.Account.Settings.Notifications.SetRecommendedVideos(true)
yt.Account.Settings.Notifications.SetChannelActivity(true)
yt.Account.Settings.Notifications.SetCommentReplies(true)
yt.Account.Settings.Notifications.SetSharedContent(true)

// Privacy
yt.Account.Settings.Privacy.SetSubscriptionsPrivate(true)
yt.Account.Settings.Privacy.SetSavedPlaylistsPrivate(true)
```

---

### Authentication

#### OAuth2 (recommended — device flow, no browser automation)

```go
import (
    "encoding/json"
    "os"
    "github.com/slick/innertube-go"
)

yt, _ := innertube.New()

// Load saved credentials
if data, err := os.ReadFile("creds.json"); err == nil {
    var creds innertube.Credentials
    if json.Unmarshal(data, &creds) == nil {
        yt.SetCredentials(&creds)
    }
}

// Sign in if needed
if !yt.IsAuthenticated() {
    err := yt.SignInWithOAuth(ctx, func(url, code string) {
        fmt.Printf("Go to %s and enter code: %s\n", url, code)
    })
    // Save credentials
    data, _ := json.MarshalIndent(yt.GetCredentials(), "", "  ")
    os.WriteFile("creds.json", data, 0600)
}
```

Tokens are **automatically refreshed** before they expire — you just need to
persist the `Credentials` struct and reload it on startup.

#### Cookies

```go
// Grab cookies from browser DevTools → Network → Request Headers → Cookie
yt.SignInWithCookies("CONSENT=YES+...; SID=...; HSID=...; SSID=...")
```

---

### YouTube Music — Lyrics

```go
search, err := yt.SearchMusic("Bohemian Rhapsody Queen")
lyrics, err := yt.GetLyrics(search.Results.Songs[0].ID)
fmt.Println(lyrics)
```

---

## Client Types

The library supports multiple YouTube client contexts:

| Constant | Description |
|---|---|
| `ClientWeb` | Standard desktop web client (default) |
| `ClientWebMusic` | YouTube Music web client |
| `ClientAndroid` | Android app — best for direct stream URLs |
| `ClientAndroidMusic` | YouTube Music Android |
| `ClientIOS` | iOS app |
| `ClientTVEmbedded` | TV embedded player — used for player requests |

The library automatically selects the right client for each request type.

---

## Error Handling

```go
var apiErr *innertube.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("API error %d on %s: %s\n", apiErr.StatusCode, apiErr.Endpoint, apiErr.Message)
}

// Sentinel errors
errors.Is(err, innertube.ErrNotAuthenticated)
errors.Is(err, innertube.ErrNoContinuation)
errors.Is(err, innertube.ErrNoStreamFound)
errors.Is(err, innertube.ErrVideoUnavailable)
errors.Is(err, innertube.ErrLiveChatDisabled)
errors.Is(err, innertube.ErrTokenRefreshFailed)
```

---

## Architecture

```
innertube-go/
├── innertube.go     — Client struct, New(), SignIn methods
├── contexts.go      — Client type configs (Web, Music, Android, iOS, TV)
├── request.go       — Core HTTP POST, JSON helpers, response traversal
├── auth.go          — OAuth2 device flow, token refresh, cookie auth
├── types.go         — All public types and structs
├── errors.go        — Sentinel errors and APIError
├── search.go        — Search(), SearchMusic(), GetSearchSuggestions()
├── video.go         — GetVideoDetails(), GetStreamingData(), GetLivechat()
├── comments.go      — GetComments(), replies, interactions
├── feed.go          — GetHomeFeed(), GetHistory(), GetSubscriptionsFeed()
├── notifications.go — GetNotifications(), GetUnseenNotificationsCount()
├── interact.go      — Like, Dislike, Subscribe, Comment, Notifications
├── account.go       — Account.Info(), Settings.Notifications, Settings.Privacy
├── playlist.go      — GetPlaylist(), GetLyrics()
├── channel.go       — GetChannel(), GetChannelVideos(), SearchChannel()
├── livechat.go      — Live chat polling, SendMessage, DeleteMessage
├── download.go      — Download(), DownloadToWriter(), ListFormats()
└── examples/
    ├── search/      — YouTube + Music search demo
    ├── download/    — Download with progress demo
    ├── auth/        — OAuth2 sign-in + interactions demo
    ├── livechat/    — Live chat bot demo
    └── music/       — Video details, comments, playlists, channels demo
```

---

## Disclaimer

This project is not affiliated with, endorsed, or sponsored by YouTube or
Google. All trademarks and brand names are the property of their respective
owners. Use responsibly and in accordance with YouTube's Terms of Service.

## License

MIT
