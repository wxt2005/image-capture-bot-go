package danbooru

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
	"github.com/wxt2005/image-capture-bot-go/model"
	"github.com/wxt2005/image-capture-bot-go/service/twitter"
)

const endpointPrefix = "https://danbooru.donmai.us/posts/"

type apiImpl struct {
	client *http.Client
}

func CheckValid(url string) ([]string, bool) {
	re := regexp.MustCompile(`(?i)danbooru\.donmai\.us\/posts\/(\d+)`)
	match := re.FindStringSubmatch(url)
	if match == nil {
		return nil, false
	}

	return match, true
}

func getIDFromURL(url string) int {
	match, ok := CheckValid(url)
	if ok == false {
		return 0
	}
	if id, error := strconv.Atoi(match[1]); error == nil {
		return id
	}

	return 0
}

type respBody struct {
	ID      int
	Source  string
	FileURL string `json:"file_url"`
}

func (api apiImpl) ExtractMedias(urls []string) ([]*model.Media, []string, error) {
	var result []*model.Media
	var remains []string

	for _, url := range urls {
		id := getIDFromURL(url)
		if id == 0 {
			remains = append(remains, url)
			continue
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s%d.json", endpointPrefix, id), nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(viper.GetString("danbooru.username"), viper.GetString("danbooru.key"))

		resp, err := api.client.Do(req)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get danbooru info failed")
		}
		var m respBody
		if err := json.Unmarshal(body, &m); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get danbooru info failed")
		}
		// use source image
		if len(m.Source) != 0 {
			_, twitterValid := twitter.CheckValid(m.Source)
			_, pixivValid := twitter.CheckValid(m.Source)

			if twitterValid || pixivValid {
				remains = append(remains, m.Source)
				continue
			}
		}

		urlParts := strings.Split(m.FileURL, "/")
		fileName := urlParts[len(urlParts)-1]
		media := model.Media{
			FileName: fileName,
			URL:      m.FileURL,
			Type:     "photo",
			Source:   url,
			Service:  "danbooru",
		}
		result = append(result, &media)
	}

	return result, remains, nil
}

func New() model.ImageService {
	return apiImpl{
		client: &http.Client{},
	}
}
