package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/raHULK777/innertube-go"
)

const credsFile = "./yt_credentials.json"

func main() {	yt, err := innertube.New()
	if err != nil {
		log.Fatal(err)
	}

	// ─── Load existing credentials if available ───────────────────────────
	if data, err := os.ReadFile(credsFile); err == nil {
		var creds innertube.Credentials
		if json.Unmarshal(data, &creds) == nil {
			yt.SetCredentials(&creds)
			fmt.Println("Loaded existing credentials from", credsFile)
		}
	}

	// ─── Sign in via OAuth2 device flow ──────────────────────────────────
	if !yt.IsAuthenticated() {
		fmt.Println("Not signed in. Starting OAuth2 device flow...")
		err = yt.SignInWithOAuth(context.Background(),
			func(verificationURL, code string) {
				fmt.Printf("\nPlease open this URL in your browser:\n  %s\n\nThen enter this code: %s\n\nWaiting...\n",
					verificationURL, code)
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Successfully signed in!")

		// Save credentials for next time
		if creds := yt.GetCredentials(); creds != nil {
			data, _ := json.MarshalIndent(creds, "", "  ")
			os.WriteFile(credsFile, data, 0600)
			fmt.Println("Credentials saved to", credsFile)
		}
	}

	// Alternative: sign in with cookies
	// yt.SignInWithCookies("COOKIE=value; SID=value; HSID=value; ...")

	// ─── Account info ─────────────────────────────────────────────────────
	fmt.Println("\n=== Account Info ===")
	info, err := yt.Account.Info()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Country: %s\n", info.Country)

	// ─── Subscriptions feed ───────────────────────────────────────────────
	fmt.Println("\n=== Subscriptions Feed ===")
	subsFeed, err := yt.GetSubscriptionsFeed()
	if err != nil {
		log.Fatal(err)
	}
	for i, group := range subsFeed.Items {
		if i >= 2 {
			break
		}
		fmt.Printf("  %s (%d videos)\n", group.Date, len(group.Videos))
		for j, v := range group.Videos {
			if j >= 2 {
				break
			}
			fmt.Printf("    - %s  (%s)\n", v.Title, v.Channel.Name)
		}
	}

	// ─── Watch history ────────────────────────────────────────────────────
	fmt.Println("\n=== Watch History ===")
	history, err := yt.GetHistory()
	if err != nil {
		log.Fatal(err)
	}
	for i, group := range history.Items {
		if i >= 2 {
			break
		}
		fmt.Printf("  %s (%d videos)\n", group.Date, len(group.Videos))
	}

	// ─── Notifications ────────────────────────────────────────────────────
	fmt.Println("\n=== Notifications ===")
	notifs, err := yt.GetNotifications()
	if err != nil {
		log.Fatal(err)
	}
	count, _ := yt.GetUnseenNotificationsCount()
	fmt.Printf("Unread: %d\n", count)
	for i, n := range notifs.Items {
		if i >= 5 {
			break
		}
		status := "read"
		if !n.Read {
			status = "UNREAD"
		}
		fmt.Printf("  [%s] %s — %s\n", status, n.ChannelName, n.Title)
	}

	// ─── Interactions ─────────────────────────────────────────────────────
	videoID := "dQw4w9WgXcQ"
	channelID := "UCuAXFkgsw1L7xaCfnd5JJOw" // Rick Astley's channel

	fmt.Printf("\n=== Interacting with video %s ===\n", videoID)

	// Like
	if _, err := yt.Interact.Like(videoID); err != nil {
		fmt.Printf("Like failed: %v\n", err)
	} else {
		fmt.Println("Liked ✓")
	}

	// Comment
	if _, err := yt.Interact.Comment(videoID, "Great song! 🎵"); err != nil {
		fmt.Printf("Comment failed: %v\n", err)
	} else {
		fmt.Println("Commented ✓")
	}

	// Subscribe
	if _, err := yt.Interact.Subscribe(channelID); err != nil {
		fmt.Printf("Subscribe failed: %v\n", err)
	} else {
		fmt.Println("Subscribed ✓")
	}

	// Change notification preferences
	if _, err := yt.Interact.ChangeNotificationPreferences(channelID, innertube.NotifAll); err != nil {
		fmt.Printf("Notification pref failed: %v\n", err)
	} else {
		fmt.Println("Notifications set to ALL ✓")
	}

	// ─── Account settings ─────────────────────────────────────────────────
	fmt.Println("\n=== Account Settings ===")
	if err := yt.Account.Settings.Notifications.SetSubscriptions(true); err != nil {
		fmt.Printf("Set subscriptions failed: %v\n", err)
	} else {
		fmt.Println("Subscription notifications enabled ✓")
	}

	if err := yt.Account.Settings.Privacy.SetSubscriptionsPrivate(false); err != nil {
		fmt.Printf("Set privacy failed: %v\n", err)
	} else {
		fmt.Println("Subscriptions set to public ✓")
	}
}
