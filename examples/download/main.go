package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raHULK777/innertube-go"
)

func main() {
	yt, err := innertube.New()
	if err != nil {
		log.Fatal(err)
	}

	videoID := "dQw4w9WgXcQ" // Rick Astley — Never Gonna Give You Up

	// ─── List available formats ───────────────────────────────────────────

	fmt.Println("=== Available Formats ===")
	formats, err := yt.ListFormats(videoID)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range formats {
		fmt.Println(" ", innertube.FormatSummary(f))
	}
	fmt.Println()

	// ─── Get streaming URL without downloading ────────────────────────────

	url, err := yt.GetDownloadURL(videoID, &innertube.StreamOptions{
		Quality: "360p",
		Type:    "videoandaudio",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Direct URL (360p, video+audio):\n%s\n\n", url)

	// ─── Download with progress ───────────────────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl-C to cancel gracefully
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nCancelling download...")
		cancel()
	}()

	fmt.Println("=== Downloading ===")
	err = yt.DownloadCtx(ctx, videoID, &innertube.DownloadOptions{
		StreamOptions: innertube.StreamOptions{
			Quality: "360p",
			Type:    "videoandaudio",
			Format:  "mp4",
		},
		OutputPath: "./rick_astley.mp4",
		OnInfo: func(details *innertube.VideoDetails, selected innertube.Format, all []innertube.Format) {
			fmt.Printf("Title: %s\n", details.Title)
			fmt.Printf("Channel: %s\n", details.Metadata.ChannelName)
			fmt.Printf("Selected format: %s\n\n", innertube.FormatSummary(selected))
		},
		OnStart: func() {
			fmt.Print("Downloading: ")
		},
		OnProgress: func(p innertube.DownloadProgress) {
			fmt.Printf("\rDownloading: %.1f%%  %.2f / %.2f MB  (%.0f KB/s, ETA %v)   ",
				p.Percentage,
				p.DownloadedMB,
				p.TotalMB,
				p.Speed/1024,
				p.ETA.Round(1e9),
			)
		},
		OnComplete: func() {
			fmt.Println("\nDone! Saved to ./rick_astley.mp4")
		},
	})

	if err != nil {
		if err == context.Canceled {
			fmt.Println("Download was cancelled.")
		} else {
			log.Fatal(err)
		}
	}

	// ─── Get streaming data without downloading ───────────────────────────

	fmt.Println("\n=== Streaming Data ===")
	streamData, err := yt.GetStreamingData(videoID, &innertube.StreamOptions{
		Quality: "720p",
		Type:    "video",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n", innertube.FormatSummary(streamData.SelectedFormat))
	fmt.Printf("Total formats available: %d\n", len(streamData.Formats))
}
