package innertube

import (
	"context"
	"fmt"
)

// GetComments retrieves comments for the given video ID.
//
//	comments, err := yt.GetComments("dQw4w9WgXcQ")
//	for _, c := range comments.Comments {
//	    fmt.Printf("%s: %s\n", c.Author.Name, c.Text)
//	}
func (c *Client) GetComments(videoID string) (*CommentsResult, error) {
	return c.GetCommentsCtx(context.Background(), videoID)
}

// GetCommentsCtx is GetComments with context.
func (c *Client) GetCommentsCtx(ctx context.Context, videoID string) (*CommentsResult, error) {
	// Comments are loaded via /next with a specific continuation
	payload := map[string]interface{}{
		"videoId": videoID,
	}

	raw, err := c.request(ctx, epNext, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	// Extract the comments continuation token from the initial response
	token := extractCommentsContinuation(raw)
	if token == "" {
		// No comments available
		return &CommentsResult{
			client:  c,
			videoID: videoID,
		}, nil
	}

	// Fetch comments using continuation
	return c.getCommentsContinuation(videoID, token)
}

// getCommentsContinuation fetches a page of comments using a continuation token.
func (c *Client) getCommentsContinuation(videoID, token string) (*CommentsResult, error) {
	payload := map[string]interface{}{
		"continuation": token,
	}

	raw, err := c.request(context.Background(), epNext, ClientWeb, payload)
	if err != nil {
		return nil, err
	}

	return parseCommentsResult(raw, videoID, c), nil
}

// getCommentReplies fetches replies to a specific comment.
func (c *Client) getCommentReplies(videoID, commentID, replyToken string) (*CommentsResult, error) {
	if replyToken == "" {
		return nil, fmt.Errorf("innertube: no reply continuation token for comment %s", commentID)
	}
	payload := map[string]interface{}{
		"continuation": replyToken,
	}
	raw, err := c.request(context.Background(), epNext, ClientWeb, payload)
	if err != nil {
		return nil, err
	}
	return parseCommentsResult(raw, videoID, c), nil
}

// likeComment sends a like action on a comment.
func (c *Client) likeComment(videoID, commentID, action string) error {
	if err := c.requireAuth(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"actions": []interface{}{
			map[string]interface{}{
				"action":                   "ACTION_LIKE",
				"comment":                  map[string]interface{}{"commentId": commentID},
				"commentVideoTimestampMsec": "0",
			},
		},
	}
	_, err := c.request(context.Background(), epPerformCommentAction, ClientWeb, payload)
	return err
}

// dislikeComment sends a dislike action on a comment.
func (c *Client) dislikeComment(videoID, commentID, action string) error {
	if err := c.requireAuth(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"actions": []interface{}{
			map[string]interface{}{
				"action":  "ACTION_DISLIKE",
				"comment": map[string]interface{}{"commentId": commentID},
			},
		},
	}
	_, err := c.request(context.Background(), epPerformCommentAction, ClientWeb, payload)
	return err
}

// replyToComment creates a reply to an existing comment.
func (c *Client) replyToComment(videoID, commentID, text string) error {
	if err := c.requireAuth(); err != nil {
		return err
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
				"parentCommentId": commentID,
			},
		},
	}
	_, err := c.request(context.Background(), epCreateCommentReply, ClientWeb, payload)
	return err
}

// ─── Parsers ──────────────────────────────────────────────────────────────────

func extractCommentsContinuation(raw map[string]interface{}) string {
	// Comments continuation lives under the results in the /next response
	contents := digArr(raw,
		"contents", "twoColumnWatchNextResults", "results", "results", "contents")
	for _, content := range contents {
		m := toMap(content)
		if m == nil {
			continue
		}
		token := digStr(m, "itemSectionRenderer", "continuations", "0",
			"nextContinuationData", "continuation")
		if token != "" {
			return token
		}
		// Also check sectionListRenderer
		token = digStr(m, "itemSectionRenderer", "header",
			"commentsHeaderRenderer", "countText", "runs")
		if token != "" {
			return token
		}
	}
	return ""
}

func parseCommentsResult(raw map[string]interface{}, videoID string, c *Client) *CommentsResult {
	result := &CommentsResult{
		client:  c,
		videoID: videoID,
	}

	// Comments come through onResponseReceivedEndpoints
	for _, cmd := range digArr(raw, "onResponseReceivedEndpoints") {
		cmdMap := toMap(cmd)
		if cmdMap == nil {
			continue
		}

		// Header (comment count)
		if header := digMap(cmdMap, "reloadContinuationItemsCommand"); header != nil {
			for _, item := range digArr(header, "continuationItems") {
				if cr := digMap(item, "commentsHeaderRenderer"); cr != nil {
					result.CommentCount = digStr(cr, "countText", "runs", "0", "text")
					if result.CommentCount == "" {
						result.CommentCount = digStr(cr, "countText", "simpleText")
					}
				}
			}
		}

		// Comment items
		if appendAction := digMap(cmdMap, "appendContinuationItemsAction"); appendAction != nil {
			for _, item := range digArr(appendAction, "continuationItems") {
				im := toMap(item)
				if im == nil {
					continue
				}

				// Continuation token
				if token := digStr(im, "continuationItemRenderer",
					"continuationEndpoint", "continuationCommand", "token"); token != "" {
					result.Continuation = token
					continue
				}

				// Comment thread renderer
				ctr := digMap(im, "commentThreadRenderer")
				if ctr == nil {
					continue
				}
				comment := parseComment(ctr, videoID, c)
				if comment != nil {
					result.Comments = append(result.Comments, *comment)
				}
			}
		}
	}

	return result
}

func parseComment(ctr map[string]interface{}, videoID string, c *Client) *Comment {
	cr := digMap(ctr, "comment", "commentRenderer")
	if cr == nil {
		return nil
	}

	id := digStr(cr, "commentId")
	text := concatRuns(digArr(cr, "contentText", "runs"))

	authorName := digStr(cr, "authorText", "simpleText")
	authorChannelID := digStr(cr, "authorEndpoint", "browseEndpoint", "browseId")
	authorThumbs := parseThumbnails(digArr(cr, "authorThumbnail", "thumbnails"))

	likeCount := int(digFloat(cr, "voteCount", "simpleText"))
	published := digStr(cr, "publishedTimeText", "runs", "0", "text")
	if published == "" {
		published = digStr(cr, "publishedTimeText", "simpleText")
	}

	isLiked := digBool(cr, "isLiked")
	isPinned := digBool(cr, "pinnedCommentBadge", "pinnedCommentBadgeRenderer", "pinned")
	isChannelOwner := digBool(cr, "authorIsChannelOwner")

	replyCount := 0
	if rc, ok := ctr["replies"].(map[string]interface{}); ok {
		if rcr := digMap(rc, "commentRepliesRenderer"); rcr != nil {
			replyCount = int(digFloat(rcr, "viewReplies", "buttonRenderer", "text", "simpleText"))
		}
	}

	// Reply continuation
	replyAction := ""
	if rc := digMap(ctr, "replies", "commentRepliesRenderer"); rc != nil {
		for _, cont := range digArr(rc, "contents") {
			if token := digStr(cont, "continuationItemRenderer",
				"continuationEndpoint", "continuationCommand", "token"); token != "" {
				replyAction = token
				break
			}
		}
	}

	return &Comment{
		ID:   id,
		Text: text,
		Author: CommentAuthor{
			Name:      authorName,
			ChannelID: authorChannelID,
			Thumbnail: authorThumbs,
		},
		Metadata: CommentMeta{
			Published:      published,
			IsLiked:        isLiked,
			IsPinned:       isPinned,
			IsChannelOwner: isChannelOwner,
			LikeCount:      likeCount,
			ReplyCount:     replyCount,
			ReplyAction:    replyAction,
		},
		client:  c,
		videoID: videoID,
	}
}
