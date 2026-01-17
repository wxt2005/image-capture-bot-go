package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	normalizedURL := fmt.Sprintf("https://www.instagram.com/p/%s/", postID)

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

	// Try using Instagram's oEmbed API
	oembedURL := fmt.Sprintf("https://graph.facebook.com/v12.0/instagram_oembed?url=%s&access_token=&fields=thumbnail_url,author_name,author_url,media_id,title",
		url.QueryEscape(incomingURL.URL))

	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		log.WithError(err).Error("Failed to create oEmbed request")
		return s.fallbackExtraction(incomingURL)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		log.WithError(err).Error("Failed to get oEmbed response")
		return s.fallbackExtraction(incomingURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
		}).Warn("oEmbed API returned non-200 status, trying fallback")
		return s.fallbackExtraction(incomingURL)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Failed to read oEmbed response body")
		return s.fallbackExtraction(incomingURL)
	}

	var oembedResp OEmbedResponse
	err = json.Unmarshal(body, &oembedResp)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal oEmbed response")
		return s.fallbackExtraction(incomingURL)
	}

	// Extract media from oEmbed response
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

	if len(result) > 0 {
		return result, nil
	}

	return s.fallbackExtraction(incomingURL)
}

func (s InstagramService) fallbackExtraction(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	// Fallback: Try to scrape the page HTML
	req, err := http.NewRequest("GET", incomingURL.URL, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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
		return result, fmt.Errorf("could not extract media from Instagram URL")
	}

	return result, nil
}
