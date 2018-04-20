package twitter

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/spf13/viper"
	"github.com/wxt2005/image_capture_bot_go/model"
)

const mp4ContentType = "video/mp4"

type apiImpl struct {
	Client *anaconda.TwitterApi
}

func CheckValid(url string) ([]string, bool) {
	re := regexp.MustCompile(`(?i)(?:www\.)?twitter\.com\/(.+?)\/status\/(\d+)`)
	match := re.FindStringSubmatch(url)
	if match == nil {
		return nil, false
	}

	return match, true
}

func getIDFromTweetURL(url string) int {
	match, ok := CheckValid(url)
	if ok == false {
		return 0
	}
	if url, error := strconv.Atoi(match[2]); error == nil {
		return url
	}

	return 0
}

func (api apiImpl) ExtractMedias(urls []string) ([]*model.Media, []string, error) {
	cli := api.Client
	var result []*model.Media
	var remains []string

	for _, urlItem := range urls {
		id := getIDFromTweetURL(urlItem)

		if id == 0 {
			remains = append(remains, urlItem)
			continue
		}

		tweet, err := cli.GetTweet(int64(id), url.Values{"tweet_mode": []string{"extended"}})
		if err != nil {
			return nil, nil, err
		}
		mediaEntities := tweet.Entities.Media
		if len(tweet.ExtendedEntities.Media) >= len(mediaEntities) {
			mediaEntities = tweet.ExtendedEntities.Media
		}

		for _, mediaEntity := range mediaEntities {
			var resultMedia *model.Media

			switch mediaEntity.Type {
			case "photo":
				resultMedia = extractPhoto(&mediaEntity)
			case "animated_gif":
				resultMedia = extractAnimatedGIF(&mediaEntity)
			default:
				continue
			}

			resultMedia.Service = "twitter"
			resultMedia.Source = urlItem
			resultMedia.FileName = fmt.Sprintf("@%s_%s", tweet.User.ScreenName, resultMedia.FileName)

			result = append(result, resultMedia)
		}
	}

	return result, remains, nil
}

func extractPhoto(media *anaconda.EntityMedia) *model.Media {
	urlParts := strings.Split(media.Media_url_https, "/")
	// wxt2005_1.jpg
	fileName := urlParts[len(urlParts)-1]

	return &model.Media{
		FileName: fileName,
		URL:      media.Media_url_https,
		Type:     "photo",
	}
}

func extractAnimatedGIF(media *anaconda.EntityMedia) *model.Media {
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

	return &model.Media{
		FileName: fileName,
		URL:      url,
		Type:     "video",
	}
}

func New() model.ImageService {
	var consumerKey = viper.GetString("twitter.consumer_key")
	var consumerSecret = viper.GetString("twitter.consumer_secret")
	var accessTokenKey = viper.GetString("twitter.access_token_key")
	var accessTokenSecret = viper.GetString("twitter.access_token_secret")
	cli := anaconda.NewTwitterApiWithCredentials(accessTokenKey, accessTokenSecret, consumerKey, consumerSecret)

	return apiImpl{
		Client: cli,
	}
}
