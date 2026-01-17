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


func (s InstagramService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	log.WithFields(log.Fields{
		"url": incomingURL.URL,
	}).Debug("Extracting Instagram media")

	// First, get metadata from HTML page
	metadata, err := s.extractMetadata(incomingURL)
	if err != nil {
		log.WithError(err).Debug("Failed to extract metadata, continuing with image extraction")
	}

	// Get the direct media URL
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
				Type:        metadata.MediaType,
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
			}).Info("Successfully extracted Instagram media")

			return result, nil
		}
	}

	return result, fmt.Errorf("direct media URL returned status %d", resp.StatusCode)
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

	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

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

	// Extract og:title
	ogTitleRegex := regexp.MustCompile(`<meta\s+property="og:title"\s+content="([^"]+)"`)
	if ogTitleMatch := ogTitleRegex.FindSubmatch(body); ogTitleMatch != nil && len(ogTitleMatch) > 1 {
		metadata.Title = string(ogTitleMatch[1])
	}

	// Extract og:description
	ogDescRegex := regexp.MustCompile(`<meta\s+property="og:description"\s+content="([^"]+)"`)
	if ogDescMatch := ogDescRegex.FindSubmatch(body); ogDescMatch != nil && len(ogDescMatch) > 1 {
		metadata.Description = string(ogDescMatch[1])
	}

	// Extract og:type
	ogTypeRegex := regexp.MustCompile(`<meta\s+property="og:type"\s+content="([^"]+)"`)
	if ogTypeMatch := ogTypeRegex.FindSubmatch(body); ogTypeMatch != nil && len(ogTypeMatch) > 1 {
		ogType := string(ogTypeMatch[1])
		if strings.Contains(ogType, "video") {
			metadata.MediaType = "video"
		}
	}

	// Extract author from title (Instagram format is usually "Username on Instagram: ...")
	if metadata.Title != "" {
		parts := strings.Split(metadata.Title, " on Instagram:")
		if len(parts) > 0 {
			metadata.Author = strings.TrimSpace(parts[0])
			metadata.AuthorURL = fmt.Sprintf("https://www.instagram.com/%s/", metadata.Author)
		}
	}

	return metadata, nil
}

