package innertube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadOptions configures a video download.
type DownloadOptions struct {
	StreamOptions

	// Range to download (byte range). Leave zero-value to download entire file.
	Range *ByteRange

	// Output path to write to. If empty, returns a ReadCloser instead.
	OutputPath string

	// OnProgress is called periodically with download progress.
	OnProgress func(DownloadProgress)

	// OnInfo is called once when download info is available (format selected).
	OnInfo func(videoDetails *VideoDetails, format Format, allFormats []Format)

	// OnStart is called when the download starts.
	OnStart func()

	// OnComplete is called when the download finishes.
	OnComplete func()
}

// ByteRange specifies a byte range for partial downloads.
type ByteRange struct {
	Start int64
	End   int64
}

// Download downloads a video to a file, streaming with progress callbacks.
//
//	err := yt.Download("dQw4w9WgXcQ", &innertube.DownloadOptions{
//	    StreamOptions: innertube.StreamOptions{
//	        Quality: "720p",
//	        Type:    "videoandaudio",
//	    },
//	    OutputPath: "./video.mp4",
//	    OnProgress: func(p innertube.DownloadProgress) {
//	        fmt.Printf("\r%.1f%%  %.1fMB / %.1fMB", p.Percentage, p.DownloadedMB, p.TotalMB)
//	    },
//	})
func (c *Client) Download(videoID string, opts *DownloadOptions) error {
	return c.DownloadCtx(context.Background(), videoID, opts)
}

// DownloadCtx is Download with context. Cancel the context to abort.
func (c *Client) DownloadCtx(ctx context.Context, videoID string, opts *DownloadOptions) error {
	if opts == nil {
		opts = &DownloadOptions{}
	}
	opts.StreamOptions.applyDefaults()

	// Fetch video details
	details, err := c.GetVideoDetailsCtx(ctx, videoID)
	if err != nil {
		return fmt.Errorf("download: get details: %w", err)
	}

	// Get streaming data
	streamData, err := c.GetStreamingDataCtx(ctx, videoID, &opts.StreamOptions)
	if err != nil {
		return fmt.Errorf("download: get streaming data: %w", err)
	}

	if opts.OnInfo != nil {
		opts.OnInfo(details, streamData.SelectedFormat, streamData.Formats)
	}

	if opts.OnStart != nil {
		opts.OnStart()
	}

	// Open output file
	out, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("download: create file: %w", err)
	}
	defer out.Close()

	err = c.streamToWriter(ctx, streamData.SelectedFormat.URL, out, opts)
	if err != nil {
		return err
	}

	if opts.OnComplete != nil {
		opts.OnComplete()
	}

	return nil
}

// DownloadToWriter downloads a video and writes it to the given writer.
// This allows piping directly to stdout, a network socket, etc.
//
//	err := yt.DownloadToWriter(ctx, "dQw4w9WgXcQ", opts, os.Stdout)
func (c *Client) DownloadToWriter(ctx context.Context, videoID string, opts *DownloadOptions, w io.Writer) error {
	if opts == nil {
		opts = &DownloadOptions{}
	}
	opts.StreamOptions.applyDefaults()

	streamData, err := c.GetStreamingDataCtx(ctx, videoID, &opts.StreamOptions)
	if err != nil {
		return fmt.Errorf("download: get streaming data: %w", err)
	}

	if opts.OnStart != nil {
		opts.OnStart()
	}

	return c.streamToWriter(ctx, streamData.SelectedFormat.URL, w, opts)
}

// GetDownloadURL returns the direct download URL for a video without downloading it.
//
//	url, err := yt.GetDownloadURL("dQw4w9WgXcQ", &innertube.StreamOptions{Quality: "360p"})
func (c *Client) GetDownloadURL(videoID string, opts *StreamOptions) (string, error) {
	if opts == nil {
		opts = &StreamOptions{}
	}
	opts.applyDefaults()

	streamData, err := c.GetStreamingData(videoID, opts)
	if err != nil {
		return "", err
	}
	return streamData.SelectedFormat.URL, nil
}

// streamToWriter performs the actual HTTP download and writes to w with progress.
func (c *Client) streamToWriter(ctx context.Context, dlURL string, w io.Writer, opts *DownloadOptions) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return fmt.Errorf("download: build request: %w", err)
	}

	req.Header.Set("User-Agent", userAgentFor(ClientWeb))
	req.Header.Set("Referer", baseURL+"/")

	if opts.Range != nil {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", opts.Range.Start, opts.Range.End))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download: server returned %s", resp.Status)
	}

	total := resp.ContentLength
	var downloaded int64
	startTime := time.Now()

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			written, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("download: write: %w", writeErr)
			}
			downloaded += int64(written)

			if opts.OnProgress != nil && total > 0 {
				elapsed := time.Since(startTime).Seconds()
				speed := 0.0
				if elapsed > 0 {
					speed = float64(downloaded) / elapsed
				}
				eta := time.Duration(0)
				if speed > 0 && total > 0 {
					remaining := float64(total-downloaded) / speed
					eta = time.Duration(remaining) * time.Second
				}
				opts.OnProgress(DownloadProgress{
					Percentage:   float64(downloaded) / float64(total) * 100,
					Downloaded:   downloaded,
					Total:        total,
					DownloadedMB: float64(downloaded) / (1024 * 1024),
					TotalMB:      float64(total) / (1024 * 1024),
					Speed:        speed,
					ETA:          eta,
				})
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("download: read: %w", readErr)
		}
	}

	return nil
}

// DownloadAudioOnly downloads only the audio track of a video.
//
//	err := yt.DownloadAudioOnly("dQw4w9WgXcQ", "./audio.webm", nil)
func (c *Client) DownloadAudioOnly(videoID, outputPath string, onProgress func(DownloadProgress)) error {
	return c.Download(videoID, &DownloadOptions{
		StreamOptions: StreamOptions{
			Type:   "audio",
			Format: "webm",
		},
		OutputPath: outputPath,
		OnProgress: onProgress,
	})
}

// DownloadVideoOnly downloads only the video track (no audio) of a video.
//
//	err := yt.DownloadVideoOnly("dQw4w9WgXcQ", "1080p", "./video.mp4", nil)
func (c *Client) DownloadVideoOnly(videoID, quality, outputPath string, onProgress func(DownloadProgress)) error {
	return c.Download(videoID, &DownloadOptions{
		StreamOptions: StreamOptions{
			Quality: quality,
			Type:    "video",
			Format:  "mp4",
		},
		OutputPath: outputPath,
		OnProgress: onProgress,
	})
}

// ListFormats returns all available streaming formats for a video without downloading.
//
//	formats, err := yt.ListFormats("dQw4w9WgXcQ")
//	for _, f := range formats {
//	    fmt.Printf("[%d] %s  %s  audio=%v video=%v\n", f.ITag, f.QualityLabel, f.MimeType, f.HasAudio, f.HasVideo)
//	}
func (c *Client) ListFormats(videoID string) ([]Format, error) {
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
	raw, err := c.request(context.Background(), epPlayer, ClientAndroid, payload)
	if err != nil {
		return nil, err
	}
	return parseFormats(raw)
}

// FormatSummary returns a human-readable summary string for a format.
//
//	fmt.Println(innertube.FormatSummary(format))
//	// → [22] video/mp4; codecs="avc1.64001F, mp4a.40.2" | 720p | audio+video | 2.3 MB/s
func FormatSummary(f Format) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("[%d]", f.ITag))
	parts = append(parts, f.MimeType)

	if f.QualityLabel != "" {
		parts = append(parts, f.QualityLabel)
	} else if f.AudioQuality != "" {
		parts = append(parts, strings.ToLower(f.AudioQuality))
	}

	kind := ""
	if f.HasAudio && f.HasVideo {
		kind = "audio+video"
	} else if f.HasAudio {
		kind = "audio only"
	} else if f.HasVideo {
		kind = "video only"
	}
	if kind != "" {
		parts = append(parts, kind)
	}

	if f.Bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%d kbps", f.Bitrate/1000))
	}

	return strings.Join(parts, " | ")
}
