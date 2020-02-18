package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type DanbooruService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *http.Client
	endpint   string
}

func NewDanbooruService() *DanbooruService {
	return &DanbooruService{
		Service:   Danbooru,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/danbooru\.donmai\.us\/posts\/(\d+)`),
		client:    &http.Client{},
		endpint:   "https://danbooru.donmai.us/posts/",
	}
}
func (s DanbooruService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	strID := match[1]
	intID, _ := strconv.Atoi(strID)

	return &IncomingURL{
		Service:  Danbooru,
		Original: urlString,
		URL:      match[0],
		StrID:    strID,
		IntID:    intID,
	}, true
}

func (s DanbooruService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s DanbooruService) ExtractMediaFromURL(incomingURL *IncomingURL) (result []*Media, err error) {
	manager := GetServiceManager()

	id := incomingURL.IntID
	if id == 0 {
		return
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%d.json", s.endpint, id), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(viper.GetString("danbooru.username"), viper.GetString("danbooru.key"))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := struct {
		ID      int
		Source  string
		FileURL string `json:"file_url"`
		PixivID int    `json:"pixiv_id"`
	}{}
	if err := json.Unmarshal(body, &m); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get danbooru info failed")
	}
	// log.Debug(string(body))
	// use source image
	if len(m.Source) != 0 {
		// Replace original pixiv image url with page url for indentity
		log.Debug(m.PixivID)
		if m.PixivID != 0 {
			m.Source = fmt.Sprintf("https://www.pixiv.net/artworks/%d", m.PixivID)
			log.WithField("New Source", m.Source).Debug("Pixiv image url to page url")
		}

		sourceServices := []ProviderService{manager.All.Twitter, manager.All.Pixiv}
		for _, provider := range sourceServices {
			if incomingURL, ok := provider.CheckValid(m.Source); ok == true {
				if media, err := provider.ExtractMediaFromURL(incomingURL); err == nil && len(media) > 0 {
					result = append(result, media...)
					return result, nil
				}
			}
		}
	}

	urlParts := strings.Split(m.FileURL, "/")
	fileName := urlParts[len(urlParts)-1]
	media := Media{
		FileName: fileName,
		URL:      m.FileURL,
		Type:     "photo",
		Source:   incomingURL.URL,
		Service:  string(s.Service),
	}
	result = append(result, &media)

	return result, nil
}
