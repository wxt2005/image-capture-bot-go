package pixiv

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/search2d/go-pixiv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image_capture_bot_go/model"
)

const ugoiraVideoEndpoint = "http://ugoira.dataprocessingclub.org/convert"

type apiImpl struct {
	Client *pixiv.Client
}

func getIDFromTweetURL(url string) int {
	re := regexp.MustCompile(`(?i)(?:www|touch)\.pixiv\.net.+illust_id=(\d+)`)
	match := re.FindStringSubmatch(url)
	if match == nil {
		return 0
	}
	if id, error := strconv.Atoi(match[1]); error == nil {
		return id
	}

	return 0
}

func (api apiImpl) ExtractMedias(urls []string) ([]*model.Media, []string, error) {
	var result []*model.Media
	var remains []string
	cli := api.Client

	for _, url := range urls {
		var urlResult []*model.Media

		id := getIDFromTweetURL(url)
		if id == 0 {
			remains = append(remains, url)
			continue
		}
		detail, err := cli.GetIllustDetail(context.TODO(), pixiv.NewGetIllustDetailParams().SetIllustID(id))
		if err != nil {
			return nil, nil, err
		}
		illust := detail.Illust
		switch illust.Type {
		case "illust", "manga":
			urlResult = append(urlResult, extractPhoto(illust)...)
		case "ugoira":
			urlResult = append(urlResult, extractUgoira(url))
		}

		for _, media := range urlResult {
			media.Source = url
			media.Service = "pixiv"
		}

		result = append(result, urlResult...)
	}

	return result, remains, nil
}

func extractPhoto(illust pixiv.GetIllustDetailIllust) []*model.Media {
	var result []*model.Media
	var urls []string
	httpClient := &http.Client{}

	if illust.PageCount == 1 {
		urls = append(urls, illust.ImageURLs["large"])
	} else {
		for _, item := range illust.MetaPages {
			urls = append(urls, item.ImageURLs["original"])
		}
	}

	for _, imageURL := range urls {
		req, err := http.NewRequest("GET", imageURL, nil)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get pixiv image failed")

			continue
		}
		req.Header.Add("Referer", `https://www.pixiv.net/`)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get pixiv image failed")
			continue
		}
		urlParts := strings.Split(imageURL, "/")
		fileName := urlParts[len(urlParts)-1]
		file, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get pixiv image failed")
			continue
		}
		defer resp.Body.Close()

		media := model.Media{
			FileName: fileName,
			URL:      imageURL,
			File:     &file,
			Type:     "photo",
		}
		result = append(result, &media)
	}

	return result
}

func extractUgoira(pageURL string) *model.Media {
	httpClient := &http.Client{}
	type reqBody struct {
		URL string `json:"url"`
	}
	form := url.Values{}
	// use gif for now, telegram do not support webm yet
	form.Add("format", "gif")
	form.Add("url", pageURL)
	req, err := http.NewRequest("POST", ugoiraVideoEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get pixiv ugoira failed")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get pixiv ugoira failed")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get pixiv ugoira failed")
	}
	var m model.UgoiraConverResponse
	json.Unmarshal(body, &m)
	videoURL := m.URL
	urlParts := strings.Split(videoURL, "/")
	fileName := urlParts[len(urlParts)-1]

	media := model.Media{
		FileName: fileName,
		URL:      videoURL,
		Type:     "video",
	}
	return &media
}

func New() model.ImageService {
	tokenProvider := &pixiv.OauthTokenProvider{Credential: pixiv.Credential{
		Username:     viper.GetString("pixiv.username"),
		Password:     viper.GetString("pixiv.password"),
		ClientID:     viper.GetString("pixiv.client_id"),
		ClientSecret: viper.GetString("pixiv.client_secret"),
	}}
	cli := &pixiv.Client{TokenProvider: tokenProvider}

	return apiImpl{
		Client: cli,
	}
}
