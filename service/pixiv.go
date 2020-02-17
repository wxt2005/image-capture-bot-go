package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/search2d/go-pixiv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image-capture-bot-go/model"
)

const ugoiraVideoEndpoint = "http://ugoira.dataprocessingclub.org/convert"

type PixivService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *pixiv.Client
}

func NewPixivService() *PixivService {
	time := time.Now().Format(time.RFC3339)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s28c1fdd170a5204386cb1313c7077b34f83e4aaf4aa829ce78c231e05b0bae2c", time))))
	headers := map[string]string{
		"User-Agent":      "PixivAndroidApp/5.0.136 (Android 6.0; Google Pixel C - 6.0.0 - API 23 - 2560x1800)",
		"Accept-Language": "en_US",
		"App-OS":          "android",
		"App-OS-Version":  "4.4.2",
		"App-Version":     "5.0.136",
		"X-Client-Time":   time,
		"X-Client-Hash":   hash,
	}
	tokenProvider := &pixiv.OauthTokenProvider{Credential: pixiv.Credential{
		Username:     viper.GetString("pixiv.username"),
		Password:     viper.GetString("pixiv.password"),
		ClientID:     viper.GetString("pixiv.client_id"),
		ClientSecret: viper.GetString("pixiv.client_secret"),
	}, Headers: headers}

	client := &pixiv.Client{TokenProvider: tokenProvider, Headers: headers}

	return &PixivService{
		Service:   Pixiv,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/(?:www|touch)\.pixiv\.net.+(?:illust_id=|artworks\/)(\d+)`),
		client:    client,
	}
}

func (s PixivService) CheckValid(urlString string) (*IncomingURL, bool) {
	log.Debug(urlString)
	match := s.urlRegexp.FindStringSubmatch(urlString)
	log.Debug(match)
	if match == nil {
		return nil, false
	}

	strID := match[1]
	intID, _ := strconv.Atoi(strID)

	return &IncomingURL{
		Service:  Pixiv,
		Original: urlString,
		URL:      match[0],
		StrID:    strID,
		IntID:    intID,
	}, true
}

func (s PixivService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s PixivService) GetIDFromURL(url string) int {
	match := s.urlRegexp.FindStringSubmatch(url)
	if match == nil {
		return 0
	}

	if id, error := strconv.Atoi(match[1]); error == nil {
		return id
	}

	return 0
}

func (s PixivService) extractPhoto(illust pixiv.GetIllustDetailIllust) []*model.Media {
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

func (s PixivService) extractUgoira(pageURL string) *model.Media {
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

func (s PixivService) ExtractMediaFromURL(incomingURL *IncomingURL) (result []*model.Media, err error) {
	client := s.client
	id := incomingURL.IntID
	if id == 0 {
		return
	}
	detail, err := client.GetIllustDetail(context.TODO(), pixiv.NewGetIllustDetailParams().SetIllustID(id))
	if err != nil {
		return nil, err
	}
	illust := detail.Illust
	switch illust.Type {
	case "illust", "manga":
		result = append(result, s.extractPhoto(illust)...)
	case "ugoira":
		result = append(result, s.extractUgoira(incomingURL.URL))
	}

	for _, media := range result {
		(*media).Source = incomingURL.URL
		(*media).Service = string(s.Service)
	}

	return result, nil
}
