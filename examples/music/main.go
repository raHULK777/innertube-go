package main

import (
	"fmt"
	"log"

	"github.com/raHULK777/innertube-go"
)

func main() {
	yt, err := innertube.New()
	if err != nil {
		log.Fatal(err)
	}

	// ─── Video Details ────────────────────────────────────────────────────

	fmt.Println("=== Video Details ===")
	video, err := yt.GetVideoDetails("dQw4w9WgXcQ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Title:       %s\n", video.Title)
	fmt.Printf("Channel:     %s (%s)\n", video.Metadata.ChannelName, video.Metadata.ChannelID)
	fmt.Printf("Views:       %d\n", video.Metadata.ViewCount)
	fmt.Printf("Duration:    %ds\n", video.Metadata.LengthSeconds)
	fmt.Printf("Published:   %s\n", video.Metadata.PublishDate)
	fmt.Printf("Category:    %s\n", video.Metadata.Category)
	fmt.Printf("Subscribers: %s\n", video.Metadata.SubscriberCount)
	fmt.Printf("IsLive:      %v\n", video.Metadata.IsLiveContent)
	fmt.Printf("Keywords:    %v\n\n", video.Metadata.Keywords[:min(5, len(video.Metadata.Keywords))])

	// ─── Comments ─────────────────────────────────────────────────────────

	fmt.Println("=== Comments ===")
	comments, err := video.GetComments()
	if err != nil {
		log.Printf("Comments error: %v\n", err)
	} else {
		fmt.Printf("Total comments: %s\n", comments.CommentCount)
		for i, c := range comments.Comments {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s: %s\n", c.Author.Name, truncate(c.Text, 80))
			fmt.Printf("    Likes: %d  Replies: %d  Pinned: %v\n", c.Metadata.LikeCount, c.Metadata.ReplyCount, c.Metadata.IsPinned)
		}

		// Get replies to first comment
		if len(comments.Comments) > 0 && comments.Comments[0].Metadata.ReplyCount > 0 {
			fmt.Println("\n  Replies to first comment:")
			replies, err := comments.Comments[0].GetReplies()
			if err == nil {
				for i, r := range replies.Comments {
					if i >= 2 {
						break
					}
					fmt.Printf("    → %s: %s\n", r.Author.Name, truncate(r.Text, 60))
				}
			}
		}

		// Continuation
		if comments.Continuation != "" {
			moreComments, err := comments.GetContinuation()
			if err == nil {
				fmt.Printf("\nNext page: %d more comments\n", len(moreComments.Comments))
			}
		}
	}

	// ─── Playlist ─────────────────────────────────────────────────────────

	fmt.Println("\n=== Playlist ===")
	playlist, err := yt.GetPlaylist("PLbpi6ZahtOH6Ar_3GPy3workV6Q8uLM4_")
	if err != nil {
		log.Printf("Playlist error: %v\n", err)
	} else {
		fmt.Printf("Title:       %s\n", playlist.Title)
		fmt.Printf("Description: %s\n", truncate(playlist.Description, 60))
		fmt.Printf("Total items: %v\n", playlist.TotalItems)
		fmt.Printf("Views:       %s\n", playlist.Views)
		fmt.Printf("Updated:     %s\n", playlist.LastUpdated)
		fmt.Printf("Items (%d loaded):\n", len(playlist.Items))
		for i, item := range playlist.Items {
			if i >= 3 {
				break
			}
			fmt.Printf("  [%d] %s — %s (%s)\n", i+1, item.Title, item.Author, item.Duration.SimpleText)
		}
	}

	// ─── Channel ──────────────────────────────────────────────────────────

	fmt.Println("\n=== Channel Info ===")
	channel, err := yt.GetChannel("UCuAXFkgsw1L7xaCfnd5JJOw")
	if err != nil {
		log.Printf("Channel error: %v\n", err)
	} else {
		fmt.Printf("Name:        %s\n", channel.Name)
		fmt.Printf("Handle:      %s\n", channel.Handle)
		fmt.Printf("Subscribers: %s\n", channel.Subscribers)
		fmt.Printf("Verified:    %v\n", channel.IsVerified)
		fmt.Printf("Description: %s\n", truncate(channel.Description, 80))
	}

	fmt.Println("\n=== Channel Videos ===")
	feed, err := yt.GetChannelVideos("UCuAXFkgsw1L7xaCfnd5JJOw")
	if err != nil {
		log.Printf("Channel videos error: %v\n", err)
	} else {
		fmt.Printf("Videos loaded: %d\n", len(feed.Videos))
		for i, v := range feed.Videos {
			if i >= 3 {
				break
			}
			fmt.Printf("  - %s (%s views)\n", v.Title, v.Metadata.ViewCount)
		}
	}

	// ─── Home Feed ────────────────────────────────────────────────────────

	fmt.Println("\n=== Home Feed (first 5) ===")
	homeFeed, err := yt.GetHomeFeed()
	if err != nil {
		log.Printf("Home feed error: %v\n", err)
	} else {
		for i, v := range homeFeed.Videos {
			if i >= 5 {
				break
			}
			fmt.Printf("  - %s\n    %s  |  %s views\n",
				v.Title, v.Channel.Name, v.Metadata.ViewCount)
		}
	}

	// ─── Lyrics (YouTube Music) ───────────────────────────────────────────

	fmt.Println("\n=== Lyrics ===")
	musicSearch, err := yt.SearchMusic("Bohemian Rhapsody Queen")
	if err == nil && len(musicSearch.Results.Songs) > 0 {
		lyrics, err := yt.GetLyrics(musicSearch.Results.Songs[0].ID)
		if err != nil {
			log.Printf("Lyrics error: %v\n", err)
		} else if lyrics != "" {
			fmt.Println(truncate(lyrics, 300))
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
