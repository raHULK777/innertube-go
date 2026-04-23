package main

import (
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

	// Search for a live stream
	fmt.Println("Searching for live streams...")
	results, err := yt.Search("lofi hip hop radio live", &innertube.SearchOptions{
		Filter: "video",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Find a live video
	var liveVideoID string
	for _, v := range results.Videos {
		if v.Metadata.IsLive {
			liveVideoID = v.ID
			fmt.Printf("Found live stream: %s\n  URL: %s\n\n", v.Title, v.URL)
			break
		}
	}

	if liveVideoID == "" {
		// Fallback: use a known live stream ID (replace with a real one)
		fmt.Println("No live video found in search results. Please provide a live video ID.")
		fmt.Print("Video ID: ")
		fmt.Scan(&liveVideoID)
	}

	// Fetch video details
	video, err := yt.GetVideoDetails(liveVideoID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Connecting to live chat for: %s\n\n", video.Title)

	// Get live chat
	lc, err := yt.GetLivechat(video)
	if err != nil {
		log.Fatal(err)
	}

	// Handle chat messages
	lc.OnChatUpdate(func(msg innertube.ChatMessage) {
		prefix := ""
		if msg.IsMember {
			prefix = "[Member] "
		}
		if msg.Amount != "" {
			prefix = fmt.Sprintf("[Superchat %s] ", msg.Amount)
		}
		if msg.Author.BadgeText != "" {
			prefix = fmt.Sprintf("[%s] ", msg.Author.BadgeText)
		}

		fmt.Printf("%s%s: %s\n", prefix, msg.Author.Name, msg.Text)

		// Auto-reply bot example
		if msg.Text == "!hello" {
			if yt.IsAuthenticated() {
				sent, err := lc.SendMessage("Hello " + msg.Author.Name + "! 👋")
				if err != nil {
					fmt.Printf("Send failed: %v\n", err)
				} else {
					fmt.Printf("→ Replied: %s\n", sent.Text)
				}
			}
		}
	})

	// Handle live metadata updates
	lc.OnMetadataUpdate(func(meta innertube.LiveMetadata) {
		fmt.Printf("[LIVE] Viewers: %s\n", meta.ViewerCount)
	})

	fmt.Println("Listening to live chat... (Ctrl+C to stop)")

	// Wait for interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	lc.Stop()
	fmt.Println("\nStopped.")
}
