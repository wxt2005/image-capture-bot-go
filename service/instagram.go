package service

import (
	"fmt"
	"io"
	"net/http"
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

// Pre-compiled regex patterns for Open Graph meta tag extraction
// These handle multi-line HTML, different quote styles, and attribute ordering
var (
	ogTitleRegex = regexp.MustCompile(`(?s)<meta[^>]*property\s*=\s*["']og:title["'][^>]*content\s*=\s*["']([^"']+)["'][^>]*>|<meta[^>]*content\s*=\s*["']([^"']+)["'][^>]*property\s*=\s*["']og:title["'][^>]*>`)
	ogDescRegex  = regexp.MustCompile(`(?s)<meta[^>]*property\s*=\s*["']og:description["'][^>]*content\s*=\s*["']([^"']+)["'][^>]*>|<meta[^>]*content\s*=\s*["']([^"']+)["'][^>]*property\s*=\s*["']og:description["'][^>]*>`)
	ogTypeRegex  = regexp.MustCompile(`(?s)<meta[^>]*property\s*=\s*["']og:type["'][^>]*content\s*=\s*["']([^"']+)["'][^>]*>|<meta[^>]*content\s*=\s*["']([^"']+)["'][^>]*property\s*=\s*["']og:type["'][^>]*>`)
)

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
	pathType := s.getPathType(urlString)
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


func (s InstagramService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	log.WithFields(log.Fields{
		"url": incomingURL.URL,
	}).Debug("Extracting Instagram media")

	// First, get metadata from HTML page
	metadata, err := s.extractMetadata(incomingURL)
	if err != nil {
		log.WithError(err).Debug("Failed to extract metadata, using defaults")
		// Initialize with defaults if metadata extraction fails
		metadata = &instagramMetadata{
			MediaType: "photo",
		}
	}

	// Get the direct media URL
	pathType := s.getPathType(incomingURL.URL)
	mediaURL := fmt.Sprintf("https://www.instagram.com/%s/%s/media/?size=l", pathType, incomingURL.StrID)

	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", instagramUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	// If we get a redirect or 200, check content type
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
		contentType := resp.Header.Get("Content-Type")
		
		// Determine file extension based on content type
		var fileName string
		var mediaType string
		if strings.Contains(contentType, "image") {
			fileName = fmt.Sprintf("instagram_%s.jpg", incomingURL.StrID)
			mediaType = "photo"
		} else if strings.Contains(contentType, "video") {
			fileName = fmt.Sprintf("instagram_%s.mp4", incomingURL.StrID)
			mediaType = "video"
		} else {
			return result, fmt.Errorf("unsupported content type: %s", contentType)
		}

		// Use metadata type if available, otherwise use detected type
		if metadata.MediaType != "" {
			mediaType = metadata.MediaType
		}

		media := &Media{
			FileName:    fileName,
			URL:         mediaURL,
			Type:        mediaType,
			Source:      incomingURL.URL,
			Service:     string(s.Service),
			Author:      metadata.Author,
			AuthorURL:   metadata.AuthorURL,
			Title:       metadata.Title,
			Description: metadata.Description,
		}
		result = append(result, media)

		log.WithFields(log.Fields{
			"url":    incomingURL.URL,
			"author": metadata.Author,
			"type":   mediaType,
		}).Info("Successfully extracted Instagram media")

		return result, nil
	}

	return result, fmt.Errorf("direct media URL returned status %d", resp.StatusCode)
}

// getPathType determines if the URL is for a post or reel
func (s InstagramService) getPathType(urlString string) string {
	if strings.Contains(urlString, "/reel/") {
		return "reel"
	}
	return "p"
}

// instagramMetadata holds metadata extracted from HTML
type instagramMetadata struct {
	Author      string
	AuthorURL   string
	Title       string
	Description string
	MediaType   string
}

// extractMetadata fetches and parses HTML to extract metadata
func (s InstagramService) extractMetadata(incomingURL *IncomingURL) (*instagramMetadata, error) {
	metadata := &instagramMetadata{
		MediaType: "photo", // default
	}

	req, err := http.NewRequest("GET", incomingURL.URL, nil)
	if err != nil {
		return metadata, err
	}

	// Set headers to match what curl sends by default
	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := s.client.Do(req)
	if err != nil {
		return metadata, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return metadata, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return metadata, err
	}

	// Extract Open Graph metadata using pre-compiled patterns
	metadata.Title = extractOGValue(ogTitleRegex, body)
	metadata.Description = extractOGValue(ogDescRegex, body)
	
	ogType := extractOGValue(ogTypeRegex, body)
	if strings.Contains(ogType, "video") {
		metadata.MediaType = "video"
	}

	// Extract author from OG title (Instagram format: "Username on Instagram: ...")
	if metadata.Title != "" {
		parts := strings.Split(metadata.Title, " on Instagram:")
		if len(parts) > 1 {
			metadata.Author = strings.TrimSpace(parts[0])
			metadata.AuthorURL = fmt.Sprintf("https://www.instagram.com/%s/", metadata.Author)
		}
	}

	// If OG tags didn't work, try extracting from <title> tag as fallback
	if metadata.Title == "" {
		titleRegex := regexp.MustCompile(`<title>([^<]+)</title>`)
		if titleMatch := titleRegex.FindSubmatch(body); titleMatch != nil && len(titleMatch) > 1 {
			metadata.Title = string(titleMatch[1])
			
			// Extract author from title (Instagram format: "Username on Instagram: ..." or "Username (@handle) • Instagram")
			if strings.Contains(metadata.Title, " on Instagram:") {
				parts := strings.Split(metadata.Title, " on Instagram:")
				if len(parts) > 1 {
					metadata.Author = strings.TrimSpace(parts[0])
					if metadata.Description == "" {
						metadata.Description = strings.TrimSpace(parts[1])
					}
				}
			} else if strings.Contains(metadata.Title, " • Instagram") {
				// Handle format like "Username (@handle) • Instagram photos and videos"
				parts := strings.Split(metadata.Title, " • Instagram")
				if len(parts) > 0 {
					titlePart := strings.TrimSpace(parts[0])
					// Extract username before (@handle)
					if strings.Contains(titlePart, " (@") {
						userParts := strings.Split(titlePart, " (@")
						if len(userParts) > 1 {
							metadata.Author = strings.TrimSpace(userParts[0])
							// Extract handle
							handlePart := strings.TrimSuffix(userParts[1], ")")
							metadata.AuthorURL = fmt.Sprintf("https://www.instagram.com/%s/", handlePart)
						}
					} else {
						metadata.Author = titlePart
					}
				}
			}
		}
	}

	// If we still don't have AuthorURL, construct it from the author
	if metadata.AuthorURL == "" && metadata.Author != "" {
		metadata.AuthorURL = fmt.Sprintf("https://www.instagram.com/%s/", metadata.Author)
	}

	log.WithFields(log.Fields{
		"url":         incomingURL.URL,
		"title":       metadata.Title,
		"author":      metadata.Author,
		"description": metadata.Description,
	}).Debug("Extracted Instagram metadata")

	return metadata, nil
}

// extractOGValue is a helper function to extract the value from Open Graph meta tag regex matches
// The regex patterns have two capture groups to handle both attribute orderings
func extractOGValue(pattern *regexp.Regexp, body []byte) string {
	if match := pattern.FindSubmatch(body); match != nil {
		// First capture group: property before content
		if len(match) > 1 && len(match[1]) > 0 {
			return string(match[1])
		}
		// Second capture group: content before property
		if len(match) > 2 && len(match[2]) > 0 {
			return string(match[2])
		}
	}
	return ""
}

