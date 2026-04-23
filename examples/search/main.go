package main

import (
	"fmt"
	"log"

	"github.com/raHULK777/innertube-go"
)

func main() {
	// Create a client — no API key needed
	yt, err := innertube.New()
	if err != nil {
		log.Fatal(err)
	}

	// ─── YouTube Search ────────────────────────────────────────────────────

	fmt.Println("=== YouTube Search ===")
	results, err := yt.Search("never gonna give you up")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Query: %q  (~%d results)\n\n", results.Query, results.EstimatedResults)
	for i, v := range results.Videos {
		if i >= 5 {
			break
		}
		fmt.Printf("[%d] %s\n    Channel: %s\n    Views: %s  Duration: %s  Published: %s\n    URL: %s\n\n",
			i+1,
			v.Title,
			v.Channel.Name,
			v.Metadata.ViewCount,
			v.Metadata.Duration.SimpleText,
			v.Metadata.Published,
			v.URL,
		)
	}

	// ─── Fetch next page ──────────────────────────────────────────────────

	if results.Continuation != "" {
		fmt.Println("Fetching next page of results...")
		more, err := yt.SearchContinuation(results.Continuation)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Got %d more videos\n\n", len(more.Videos))
	}

	// ─── Search Suggestions ───────────────────────────────────────────────

	fmt.Println("=== Search Suggestions ===")
	suggestions, err := yt.GetSearchSuggestions("never gonna")
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range suggestions {
		fmt.Printf("  → %s\n", s.Text)
	}

	// ─── YouTube Music Search ─────────────────────────────────────────────

	fmt.Println("\n=== YouTube Music Search ===")
	musicResults, err := yt.SearchMusic("Daft Punk")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Songs (%d):\n", len(musicResults.Results.Songs))
	for i, s := range musicResults.Results.Songs {
		if i >= 3 {
			break
		}
		fmt.Printf("  %s — %s (%s)\n", s.Title, s.Artist, s.Duration)
	}

	fmt.Printf("\nAlbums (%d):\n", len(musicResults.Results.Albums))
	for i, a := range musicResults.Results.Albums {
		if i >= 3 {
			break
		}
		fmt.Printf("  %s by %s (%s)\n", a.Title, a.Author, a.Year)
	}

	fmt.Printf("\nArtists (%d):\n", len(musicResults.Results.Artists))
	for i, a := range musicResults.Results.Artists {
		if i >= 3 {
			break
		}
		fmt.Printf("  %s (%s subscribers)\n", a.Name, a.Subscribers)
	}
}
