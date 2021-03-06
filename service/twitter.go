package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/spf13/viper"
)

const mp4ContentType = "video/mp4"

type TwitterService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *anaconda.TwitterApi
}

func NewTwitterService() *TwitterService {
	consumerKey := viper.GetString("twitter.consumer_key")
	consumerSecret := viper.GetString("twitter.consumer_secret")
	accessTokenKey := viper.GetString("twitter.access_token_key")
	accessTokenSecret := viper.GetString("twitter.access_token_secret")
	client := anaconda.NewTwitterApiWithCredentials(accessTokenKey, accessTokenSecret, consumerKey, consumerSecret)

	return &TwitterService{
		Service:   Twitter,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?twitter\.com\/(.+?)\/status\/(\d+)`),
		client:    client,
	}
}

func (s TwitterService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	strID := match[2]
	intID, _ := strconv.Atoi(strID)

	return &IncomingURL{
		Service:  s.Service,
		Original: urlString,
		URL:      match[0],
		StrID:    strID,
		IntID:    intID,
	}, true
}

func (s TwitterService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s TwitterService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media
	client := s.client
	id := incomingURL.IntID

	if id == 0 {
		return result, nil
	}

	tweet, err := client.GetTweet(int64(id), url.Values{"tweet_mode": []string{"extended"}})
	if err != nil {
		return nil, err
	}

	mediaEntities := tweet.Entities.Media
	if len(tweet.ExtendedEntities.Media) >= len(mediaEntities) {
		mediaEntities = tweet.ExtendedEntities.Media
	}

	for _, mediaEntity := range mediaEntities {
		var resultMedia *Media

		switch mediaEntity.Type {
		case "photo":
			resultMedia = s.extractPhoto(&mediaEntity)
		case "animated_gif":
			resultMedia = s.extractAnimatedGIF(&mediaEntity)
		default:
			continue
		}

		resultMedia.Service = string(s.Service)
		resultMedia.Source = incomingURL.URL
		resultMedia.FileName = fmt.Sprintf("@%s_%s", tweet.User.ScreenName, resultMedia.FileName)

		result = append(result, resultMedia)
	}

	return result, nil
}

func (s TwitterService) extractPhoto(media *anaconda.EntityMedia) *Media {
	urlParts := strings.Split(media.Media_url_https, "/")
	// wxt2005_1.jpg
	fileName := urlParts[len(urlParts)-1]

	return &Media{
		FileName: fileName,
		URL:      media.Media_url_https,
		Type:     "photo",
	}
}

func (s TwitterService) extractAnimatedGIF(media *anaconda.EntityMedia) *Media {
	variants := media.VideoInfo.Variants
	var variant *anaconda.Variant
	for _, item := range variants {
		if item.ContentType == mp4ContentType {
			variant = &item
		}
	}

	if variant == nil {
		return nil
	}

	url := variant.Url
	urlParts := strings.Split(url, "/")
	// wxt2005_1.mp4
	fileName := urlParts[len(urlParts)-1]

	return &Media{
		FileName: fileName,
		URL:      url,
		Type:     "video",
	}
}
