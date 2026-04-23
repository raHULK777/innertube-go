package innertube

import (
	"context"
	"fmt"
)

// InteractClient provides interaction endpoints (like, subscribe, comment, etc.).
// All methods require authentication.
type InteractClient struct {
	client *Client
}

// Subscribe subscribes to a channel.
//
//	err := yt.Interact.Subscribe("UCxxxxxx")
func (ic *InteractClient) Subscribe(channelID string) (*ActionResult, error) {
	return ic.SubscribeCtx(context.Background(), channelID)
}

// SubscribeCtx is Subscribe with context.
func (ic *InteractClient) SubscribeCtx(ctx context.Context, channelID string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"channelIds": []string{channelID},
		"params":     "EgIIAhgA",
	}
	_, err := ic.client.request(ctx, epSubscribe, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// Unsubscribe unsubscribes from a channel.
//
//	err := yt.Interact.Unsubscribe("UCxxxxxx")
func (ic *InteractClient) Unsubscribe(channelID string) (*ActionResult, error) {
	return ic.UnsubscribeCtx(context.Background(), channelID)
}

// UnsubscribeCtx is Unsubscribe with context.
func (ic *InteractClient) UnsubscribeCtx(ctx context.Context, channelID string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"channelIds": []string{channelID},
		"params":     "CgIIAhgA",
	}
	_, err := ic.client.request(ctx, epUnsubscribe, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// Like likes a video.
//
//	err := yt.Interact.Like("dQw4w9WgXcQ")
func (ic *InteractClient) Like(videoID string) (*ActionResult, error) {
	return ic.LikeCtx(context.Background(), videoID)
}

// LikeCtx is Like with context.
func (ic *InteractClient) LikeCtx(ctx context.Context, videoID string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"target": map[string]interface{}{
			"videoId": videoID,
		},
	}
	_, err := ic.client.request(ctx, epLike, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// Dislike dislikes a video.
//
//	err := yt.Interact.Dislike("dQw4w9WgXcQ")
func (ic *InteractClient) Dislike(videoID string) (*ActionResult, error) {
	return ic.DislikeCtx(context.Background(), videoID)
}

// DislikeCtx is Dislike with context.
func (ic *InteractClient) DislikeCtx(ctx context.Context, videoID string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"target": map[string]interface{}{
			"videoId": videoID,
		},
	}
	_, err := ic.client.request(ctx, epDislike, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// RemoveLike removes a like or dislike from a video.
//
//	err := yt.Interact.RemoveLike("dQw4w9WgXcQ")
func (ic *InteractClient) RemoveLike(videoID string) (*ActionResult, error) {
	return ic.RemoveLikeCtx(context.Background(), videoID)
}

// RemoveLikeCtx is RemoveLike with context.
func (ic *InteractClient) RemoveLikeCtx(ctx context.Context, videoID string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"target": map[string]interface{}{
			"videoId": videoID,
		},
	}
	_, err := ic.client.request(ctx, epRemoveLike, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// Comment posts a comment on a video.
//
//	err := yt.Interact.Comment("dQw4w9WgXcQ", "Great video!")
func (ic *InteractClient) Comment(videoID, text string) (*ActionResult, error) {
	return ic.CommentCtx(context.Background(), videoID, text)
}

// CommentCtx is Comment with context.
func (ic *InteractClient) CommentCtx(ctx context.Context, videoID, text string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"videoId": videoID,
		"comment": map[string]interface{}{
			"commentText": map[string]interface{}{
				"simpleText": text,
			},
		},
		"commentsRelatedEntities": []interface{}{
			map[string]interface{}{
				"videoId": videoID,
			},
		},
	}
	_, err := ic.client.request(ctx, epCreateComment, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// ChangeNotificationPreferences changes notification preferences for a channel.
// Valid values: NotifAll, NotifNone, NotifPersonalized.
//
//	err := yt.Interact.ChangeNotificationPreferences("UCxxxxxx", innertube.NotifAll)
func (ic *InteractClient) ChangeNotificationPreferences(channelID, preference string) (*ActionResult, error) {
	return ic.ChangeNotificationPreferencesCtx(context.Background(), channelID, preference)
}

// ChangeNotificationPreferencesCtx is ChangeNotificationPreferences with context.
func (ic *InteractClient) ChangeNotificationPreferencesCtx(ctx context.Context, channelID, preference string) (*ActionResult, error) {
	if err := ic.client.requireAuth(); err != nil {
		return nil, err
	}

	prefParam, ok := notifPreferenceParams[preference]
	if !ok {
		return nil, fmt.Errorf("innertube: invalid notification preference %q; use NotifAll, NotifNone, or NotifPersonalized", preference)
	}

	payload := map[string]interface{}{
		"channelId": channelID,
		"params":    prefParam,
	}
	_, err := ic.client.request(ctx, epNotifyChangePreference, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Success: true, StatusCode: 200}, nil
}

// notifPreferenceParams maps preference names to InnerTube params.
var notifPreferenceParams = map[string]string{
	NotifAll:          "EgIIAxAA",
	NotifNone:         "EgIIARAA",
	NotifPersonalized: "EgIIAhAA",
}
