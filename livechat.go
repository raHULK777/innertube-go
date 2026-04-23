package innertube

import (
	"context"
	"time"
)

const (
	liveChatPollInterval = 5 * time.Second
)

// poll starts a goroutine that continuously fetches live chat messages.
func (lc *LiveChat) poll() {
	defer close(lc.chatUpdateCh)
	defer close(lc.metaUpdateCh)

	for {
		select {
		case <-lc.stopCh:
			return
		default:
		}

		messages, metadata, nextContinuation, err := lc.fetchLiveChatPage(lc.continuation)
		if err != nil {
			// Back off on error and retry
			select {
			case <-lc.stopCh:
				return
			case <-time.After(liveChatPollInterval):
				continue
			}
		}

		if nextContinuation != "" {
			lc.continuation = nextContinuation
		}

		for _, msg := range messages {
			select {
			case lc.chatUpdateCh <- msg:
			case <-lc.stopCh:
				return
			}
		}

		if metadata != nil {
			select {
			case lc.metaUpdateCh <- *metadata:
			default:
			}
		}

		select {
		case <-lc.stopCh:
			return
		case <-time.After(liveChatPollInterval):
		}
	}
}

// fetchLiveChatPage fetches one page of live chat messages.
func (lc *LiveChat) fetchLiveChatPage(continuation string) ([]ChatMessage, *LiveMetadata, string, error) {
	payload := map[string]interface{}{
		"continuation": continuation,
	}

	raw, err := lc.client.request(context.Background(), epLiveChatGetMessages, ClientWeb, payload)
	if err != nil {
		return nil, nil, "", err
	}

	return parseLiveChatPage(raw)
}

// sendLiveChatMessage sends a message to the live chat.
func (c *Client) sendLiveChatMessage(videoID, text, continuation string) (*ChatMessage, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"params":  continuation,
		"richMessage": map[string]interface{}{
			"textSegments": []interface{}{
				map[string]interface{}{
					"text": text,
				},
			},
		},
	}

	raw, err := c.request(context.Background(), epLiveChatSendMessage, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	return parseSentMessage(raw, text), nil
}

// deleteLiveChatMessage deletes a live chat message by its delete token.
func (c *Client) deleteLiveChatMessage(videoID, deleteToken string) error {
	if err := c.requireAuth(); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"params": deleteToken,
	}

	_, err := c.request(context.Background(), epLiveChatDeleteMessage, ClientWeb, payload)
	return err
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func parseLiveChatPage(raw map[string]interface{}) ([]ChatMessage, *LiveMetadata, string, error) {
	var messages []ChatMessage
	var metadata *LiveMetadata

	liveChatContinuation := digMap(raw,
		"continuationContents", "liveChatContinuation")
	if liveChatContinuation == nil {
		return messages, metadata, "", nil
	}

	// Continuation token for next poll
	nextToken := ""
	for _, cont := range digArr(liveChatContinuation, "continuations") {
		if token := digStr(cont, "timedContinuationData", "continuation"); token != "" {
			nextToken = token
			break
		}
		if token := digStr(cont, "invalidationContinuationData", "continuation"); token != "" {
			nextToken = token
			break
		}
		if token := digStr(cont, "reloadContinuationData", "continuation"); token != "" {
			nextToken = token
			break
		}
	}

	// Chat messages
	for _, action := range digArr(liveChatContinuation, "actions") {
		am := toMap(action)
		if am == nil {
			continue
		}

		// Regular chat message
		if addAction := digMap(am, "addChatItemAction"); addAction != nil {
			item := digMap(addAction, "item")
			if item == nil {
				continue
			}

			msg := parseChatItem(item)
			if msg != nil {
				messages = append(messages, *msg)
			}
		}

		// Update metadata action
		if updateAction := digMap(am, "updateViewershipAction"); updateAction != nil {
			metadata = parseViewershipData(updateAction)
		}
	}

	return messages, metadata, nextToken, nil
}

func parseChatItem(item map[string]interface{}) *ChatMessage {
	// Regular message
	if lctr := digMap(item, "liveChatTextMessageRenderer"); lctr != nil {
		return parseChatTextMessage(lctr, false, "")
	}

	// Superchat / paid message
	if lcpr := digMap(item, "liveChatPaidMessageRenderer"); lcpr != nil {
		msg := parseChatTextMessage(digMap(lcpr, "message", "runs", "0"), true, "")
		if msg == nil {
			msg = &ChatMessage{}
		}
		msg.Amount = digStr(lcpr, "purchaseAmountText", "simpleText")
		msg.Color = getColorName(int(digFloat(lcpr, "bodyBackgroundColor")))
		return msg
	}

	// Member message
	if lcmr := digMap(item, "liveChatMembershipItemRenderer"); lcmr != nil {
		msg := parseChatTextMessage(lcmr, false, "")
		if msg != nil {
			msg.IsMember = true
			if msg.Text == "" {
				msg.Text = concatRuns(digArr(lcmr, "headerSubtext", "runs"))
			}
		}
		return msg
	}

	return nil
}

func parseChatTextMessage(r map[string]interface{}, isMember bool, amount string) *ChatMessage {
	if r == nil {
		return nil
	}

	id := digStr(r, "id")
	text := concatRuns(digArr(r, "message", "runs"))
	authorName := digStr(r, "authorName", "simpleText")
	authorChannelID := digStr(r, "authorExternalChannelId")
	authorThumbs := parseThumbnails(digArr(r, "authorPhoto", "thumbnails"))

	badgeText := ""
	for _, badge := range digArr(r, "authorBadges") {
		if lb := digMap(badge, "liveChatAuthorBadgeRenderer"); lb != nil {
			badgeText = digStr(lb, "tooltip")
		}
	}

	deleteToken := digStr(r, "contextMenuEndpoint", "liveChatItemContextMenuEndpoint", "params")

	var ts time.Time
	if tsMs := digStr(r, "timestampUsec"); tsMs != "" {
		var usec int64
		for _, ch := range tsMs {
			usec = usec*10 + int64(ch-'0')
		}
		ts = time.UnixMicro(usec)
	}

	return &ChatMessage{
		ID:   id,
		Text: text,
		Author: ChatAuthor{
			Name:      authorName,
			ChannelID: authorChannelID,
			Thumbnail: authorThumbs,
			BadgeText: badgeText,
		},
		Timestamp:   ts,
		IsMember:    isMember,
		Amount:      amount,
		deleteToken: deleteToken,
	}
}

func parseViewershipData(action map[string]interface{}) *LiveMetadata {
	return &LiveMetadata{
		ViewerCount: digStr(action, "viewCount", "videoViewCountRenderer",
			"viewCount", "simpleText"),
	}
}

func parseSentMessage(raw map[string]interface{}, text string) *ChatMessage {
	// The sent message response echoes back the message details
	item := digMap(raw, "liveChatItemActionEndpoint", "item")
	if item == nil {
		return &ChatMessage{Text: text, Timestamp: time.Now()}
	}
	msg := parseChatItem(item)
	if msg == nil {
		return &ChatMessage{Text: text, Timestamp: time.Now()}
	}
	return msg
}

// getColorName maps a YouTube superchat ARGB color to a friendly name.
func getColorName(argb int) string {
	colors := map[int]string{
		0xFFE62117: "red",
		0xFFF57C00: "orange",
		0xFFFFCA28: "yellow",
		0xFF1DE9B6: "teal",
		0xFF00BFA5: "green",
		0xFF1565C0: "blue",
		0xFF7B1FA2: "purple",
	}
	if name, ok := colors[argb]; ok {
		return name
	}
	return "unknown"
}
