package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	log "github.com/sirupsen/logrus"
)

type BlueskyService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *xrpc.Client
}

func NewBlueskyService() *BlueskyService {
	client := &xrpc.Client{
		Host: "https://public.api.bsky.app",
	}

	return &BlueskyService{
		Service:   Bluesky,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/bsky\.app\/profile\/([^\/]+)\/post\/([^\/\?]+)`),
		client:    client,
	}
}

func (s BlueskyService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	handle := match[1]
	rkey := match[2]

	return &IncomingURL{
		Service:  s.Service,
		Original: urlString,
		URL:      match[0],
		Host:     handle,
		StrID:    rkey,
		IntID:    0,
	}, true
}

func (s BlueskyService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s BlueskyService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media
	ctx := context.Background()

	handle := incomingURL.Host
	rkey := incomingURL.StrID

	if handle == "" || rkey == "" {
		return result, fmt.Errorf("invalid Bluesky URL: missing handle or post ID")
	}

	// Resolve handle to DID if needed
	did := handle
	if !strings.HasPrefix(handle, "did:") {
		resolvedDid, err := s.resolveHandle(ctx, handle)
		if err != nil {
			log.WithError(err).Error("Failed to resolve Bluesky handle")
			return result, err
		}
		did = resolvedDid
	}

	// Construct AT URI
	uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey)

	// Get post thread
	output, err := bsky.FeedGetPostThread(ctx, s.client, 0, 0, uri)
	if err != nil {
		log.WithError(err).Error("Failed to get Bluesky post thread")
		return result, err
	}

	// Extract the main post from the thread
	if output.Thread.FeedDefs_ThreadViewPost == nil {
		return result, fmt.Errorf("unexpected thread type")
	}

	post := output.Thread.FeedDefs_ThreadViewPost.Post

	// Extract author information
	author := post.Author.DisplayName
	if author == nil {
		author = &post.Author.Handle
	}
	authorURL := fmt.Sprintf("https://bsky.app/profile/%s", post.Author.Handle)

	// Extract description/text
	var description string
	if postRecord, ok := post.Record.Val.(*bsky.FeedPost); ok {
		description = postRecord.Text
	}

	// Extract media from embed
	if post.Embed != nil {
		if post.Embed.EmbedImages_View != nil {
			for _, image := range post.Embed.EmbedImages_View.Images {
				media := s.extractImage(image, incomingURL.URL, *author, authorURL, description)
				result = append(result, media)
			}
		} else if post.Embed.EmbedVideo_View != nil {
			media := s.extractVideo(post.Embed.EmbedVideo_View, incomingURL.URL, *author, authorURL, description)
			if media != nil {
				result = append(result, media)
			}
		} else if post.Embed.EmbedRecordWithMedia_View != nil {
			embed := post.Embed.EmbedRecordWithMedia_View
			// Handle posts with quoted posts that have media
			if mediaEmbed := embed.Media; mediaEmbed != nil {
				if mediaEmbed.EmbedImages_View != nil {
					for _, image := range mediaEmbed.EmbedImages_View.Images {
						media := s.extractImage(image, incomingURL.URL, *author, authorURL, description)
						result = append(result, media)
					}
				} else if mediaEmbed.EmbedVideo_View != nil {
					media := s.extractVideo(mediaEmbed.EmbedVideo_View, incomingURL.URL, *author, authorURL, description)
					if media != nil {
						result = append(result, media)
					}
				}
			}
		}
	}

	return result, nil
}

func (s BlueskyService) resolveHandle(ctx context.Context, handle string) (string, error) {
	output, err := bsky.ActorGetProfile(ctx, s.client, handle)
	if err != nil {
		return "", err
	}
	return output.Did, nil
}

func (s BlueskyService) extractImage(image *bsky.EmbedImages_ViewImage, source string, author string, authorURL string, description string) *Media {
	// Extract filename from URL
	urlParts := strings.Split(image.Fullsize, "/")
	fileName := urlParts[len(urlParts)-1]

	// Remove query parameters if present
	if idx := strings.Index(fileName, "?"); idx != -1 {
		fileName = fileName[:idx]
	}

	return &Media{
		FileName:    fileName,
		URL:         image.Fullsize,
		Type:        "photo",
		Source:      source,
		Service:     string(s.Service),
		Author:      author,
		AuthorURL:   authorURL,
		Description: description,
	}
}

func (s BlueskyService) extractVideo(video *bsky.EmbedVideo_View, source string, author string, authorURL string, description string) *Media {
	if video.Playlist == "" {
		return nil
	}

	// Extract filename from playlist URL
	urlParts := strings.Split(video.Playlist, "/")
	fileName := urlParts[len(urlParts)-1]

	// Remove query parameters and replace with .mp4
	if idx := strings.Index(fileName, "?"); idx != -1 {
		fileName = fileName[:idx]
	}
	if !strings.HasSuffix(fileName, ".mp4") {
		fileName = strings.TrimSuffix(fileName, ".m3u8") + ".mp4"
	}

	return &Media{
		FileName:    fileName,
		URL:         video.Playlist,
		Type:        "video",
		Source:      source,
		Service:     string(s.Service),
		Author:      author,
		AuthorURL:   authorURL,
		Description: description,
	}
}
