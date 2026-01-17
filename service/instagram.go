package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type InstagramService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *http.Client
}

const instagramUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func NewInstagramService() *InstagramService {
	return &InstagramService{
		Service:   Instagram,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?instagram\.com\/(?:p|reel)\/([A-Za-z0-9_-]+)`),
		client:    &http.Client{},
	}
}

func (s InstagramService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	postID := match[1]
	// Preserve the original path type (p or reel)
	pathType := "p"
	if strings.Contains(urlString, "/reel/") {
		pathType = "reel"
	}
	normalizedURL := fmt.Sprintf("https://www.instagram.com/%s/%s/", pathType, postID)

	return &IncomingURL{
		Service:  s.Service,
		Original: urlString,
		URL:      normalizedURL,
		StrID:    postID,
	}, true
}

func (s InstagramService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

// OEmbedResponse represents the response from Instagram's oEmbed API
type OEmbedResponse struct {
	Version         string `json:"version"`
	Title           string `json:"title"`
	AuthorName      string `json:"author_name"`
	AuthorURL       string `json:"author_url"`
	AuthorID        int64  `json:"author_id"`
	MediaID         string `json:"media_id"`
	ProviderName    string `json:"provider_name"`
	ProviderURL     string `json:"provider_url"`
	Type            string `json:"type"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	HTML            string `json:"html"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_width"`
	ThumbnailHeight int    `json:"thumbnail_height"`
}

func (s InstagramService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	// Try method 1: Instagram's own oEmbed API (legacy, but sometimes works)
	result, err := s.tryInstagramOEmbed(incomingURL)
	if err == nil && len(result) > 0 {
		return result, nil
	}
	log.WithError(err).Debug("Instagram oEmbed failed, trying direct media URL")

	// Try method 2: Direct media URL
	result, err = s.tryDirectMediaURL(incomingURL)
	if err == nil && len(result) > 0 {
		return result, nil
	}
	log.WithError(err).Debug("Direct media URL failed, trying JSON endpoint")

	// Try method 3: JSON endpoint with __a parameter
	result, err = s.tryJSONEndpoint(incomingURL)
	if err == nil && len(result) > 0 {
		return result, nil
	}
	log.WithError(err).Debug("JSON endpoint failed, trying HTML scraping")

	// Try method 4: HTML scraping as last resort
	result, err = s.fallbackExtraction(incomingURL)
	if err == nil && len(result) > 0 {
		return result, nil
	}

	return result, fmt.Errorf("all extraction methods failed for Instagram URL: %s", incomingURL.URL)
}

// tryInstagramOEmbed tries Instagram's own oEmbed endpoint
func (s InstagramService) tryInstagramOEmbed(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	oembedURL := fmt.Sprintf("https://api.instagram.com/oembed/?url=%s", url.QueryEscape(incomingURL.URL))

	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", instagramUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("oEmbed API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	var oembedResp OEmbedResponse
	err = json.Unmarshal(body, &oembedResp)
	if err != nil {
		return result, err
	}

	if oembedResp.ThumbnailURL != "" {
		fileName := fmt.Sprintf("instagram_%s.jpg", incomingURL.StrID)
		
		mediaType := "photo"
		if oembedResp.Type == "video" {
			mediaType = "video"
		}

		media := &Media{
			FileName:    fileName,
			URL:         oembedResp.ThumbnailURL,
			Type:        mediaType,
			Source:      incomingURL.URL,
			Service:     string(s.Service),
			Author:      oembedResp.AuthorName,
			AuthorURL:   oembedResp.AuthorURL,
			Title:       oembedResp.Title,
			Description: "",
		}
		result = append(result, media)
	}

	if len(result) == 0 {
		return result, fmt.Errorf("no media found in oEmbed response")
	}

	return result, nil
}

// tryDirectMediaURL tries to get the image directly via Instagram's media endpoint
func (s InstagramService) tryDirectMediaURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	// Determine the path type from the URL (p for posts, reel for reels)
	pathType := "p"
	if strings.Contains(incomingURL.URL, "/reel/") {
		pathType = "reel"
	}
	mediaURL := fmt.Sprintf("https://www.instagram.com/%s/%s/media/?size=l", pathType, incomingURL.StrID)

	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", instagramUserAgent)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	// If we get a redirect or 200 with an image content type, it worked
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "image") {
			fileName := fmt.Sprintf("instagram_%s.jpg", incomingURL.StrID)
			
			media := &Media{
				FileName:    fileName,
				URL:         mediaURL,
				Type:        "photo",
				Source:      incomingURL.URL,
				Service:     string(s.Service),
				Author:      "",
				AuthorURL:   "",
				Title:       "",
				Description: "",
			}
			result = append(result, media)
			return result, nil
		}
	}

	return result, fmt.Errorf("direct media URL returned status %d", resp.StatusCode)
}

// tryJSONEndpoint tries to get data from Instagram's JSON endpoint
func (s InstagramService) tryJSONEndpoint(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	// Determine the path type from the URL (p for posts, reel for reels)
	pathType := "p"
	if strings.Contains(incomingURL.URL, "/reel/") {
		pathType = "reel"
	}
	jsonURL := fmt.Sprintf("https://www.instagram.com/%s/%s/?__a=1&__d=dis", pathType, incomingURL.StrID)

	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", instagramUserAgent)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("JSON endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	// Try to parse JSON response and extract media URLs
	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		return result, err
	}

	// Navigate the JSON structure to find media
	// Instagram's JSON structure: items[0].carousel_media or items[0].image_versions2
	if items, ok := jsonData["items"].([]interface{}); ok && len(items) > 0 {
		if item, ok := items[0].(map[string]interface{}); ok {
			imageURL := ""
			
			// Try to get image from image_versions2
			if imageVersions, ok := item["image_versions2"].(map[string]interface{}); ok {
				if candidates, ok := imageVersions["candidates"].([]interface{}); ok && len(candidates) > 0 {
					if candidate, ok := candidates[0].(map[string]interface{}); ok {
						if url, ok := candidate["url"].(string); ok {
							imageURL = url
						}
					}
				}
			}

			if imageURL != "" {
				fileName := fmt.Sprintf("instagram_%s.jpg", incomingURL.StrID)
				
				// Extract author info if available
				author := ""
				authorURL := ""
				if user, ok := item["user"].(map[string]interface{}); ok {
					if username, ok := user["username"].(string); ok {
						author = username
						authorURL = fmt.Sprintf("https://www.instagram.com/%s/", username)
					}
				}

				// Extract caption
				description := ""
				if caption, ok := item["caption"].(map[string]interface{}); ok {
					if text, ok := caption["text"].(string); ok {
						description = text
					}
				}

				media := &Media{
					FileName:    fileName,
					URL:         imageURL,
					Type:        "photo",
					Source:      incomingURL.URL,
					Service:     string(s.Service),
					Author:      author,
					AuthorURL:   authorURL,
					Title:       "",
					Description: description,
				}
				result = append(result, media)
			}
		}
	}

	if len(result) == 0 {
		return result, fmt.Errorf("no media found in JSON response")
	}

	return result, nil
}

func (s InstagramService) fallbackExtraction(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	// Fallback: Try to scrape the page HTML
	req, err := http.NewRequest("GET", incomingURL.URL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return result, fmt.Errorf("Instagram returned 403 Forbidden - rate limiting or authentication required")
	}

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	// Extract og:image meta tag
	ogImageRegex := regexp.MustCompile(`<meta\s+property="og:image"\s+content="([^"]+)"`)
	ogImageMatch := ogImageRegex.FindSubmatch(body)

	// Extract og:title
	ogTitleRegex := regexp.MustCompile(`<meta\s+property="og:title"\s+content="([^"]+)"`)
	ogTitleMatch := ogTitleRegex.FindSubmatch(body)

	// Extract og:description
	ogDescRegex := regexp.MustCompile(`<meta\s+property="og:description"\s+content="([^"]+)"`)
	ogDescMatch := ogDescRegex.FindSubmatch(body)

	// Extract og:type
	ogTypeRegex := regexp.MustCompile(`<meta\s+property="og:type"\s+content="([^"]+)"`)
	ogTypeMatch := ogTypeRegex.FindSubmatch(body)

	if ogImageMatch != nil && len(ogImageMatch) > 1 {
		imageURL := string(ogImageMatch[1])
		fileName := fmt.Sprintf("instagram_%s.jpg", incomingURL.StrID)

		title := ""
		if ogTitleMatch != nil && len(ogTitleMatch) > 1 {
			title = string(ogTitleMatch[1])
		}

		description := ""
		if ogDescMatch != nil && len(ogDescMatch) > 1 {
			description = string(ogDescMatch[1])
		}

		mediaType := "photo"
		if ogTypeMatch != nil && len(ogTypeMatch) > 1 {
			ogType := string(ogTypeMatch[1])
			if strings.Contains(ogType, "video") {
				mediaType = "video"
			}
		}

		// Extract author from title (Instagram format is usually "Username on Instagram: ...")
		author := ""
		authorURL := ""
		if title != "" {
			parts := strings.Split(title, " on Instagram:")
			if len(parts) > 0 {
				author = strings.TrimSpace(parts[0])
				authorURL = fmt.Sprintf("https://www.instagram.com/%s/", author)
			}
		}

		media := &Media{
			FileName:    fileName,
			URL:         imageURL,
			Type:        mediaType,
			Source:      incomingURL.URL,
			Service:     string(s.Service),
			Author:      author,
			AuthorURL:   authorURL,
			Title:       title,
			Description: description,
		}
		result = append(result, media)
	}

	if len(result) == 0 {
		return result, fmt.Errorf("could not extract media from Instagram URL - HTML parsing found no og:image tags")
	}

	return result, nil
}
