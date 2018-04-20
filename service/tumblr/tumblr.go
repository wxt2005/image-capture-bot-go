package tumblr

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/wxt2005/image_capture_bot_go/model"
)

type apiImpl struct {
	client *http.Client
}

func CheckValid(url string) ([]string, bool) {
	re := regexp.MustCompile(`(?i)tumblr\.com\/`)
	match := re.FindStringSubmatch(url)
	if match == nil {
		return nil, false
	}

	return match, true
}

func (api apiImpl) ExtractMedias(urls []string) ([]*model.Media, []string, error) {
	var result []*model.Media
	var remains []string

	for _, url := range urls {
		if _, ok := CheckValid(url); ok == false {
			remains = append(remains, url)
			continue
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := api.client.Do(req)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Get tumblr page failed")
		}
		re := regexp.MustCompile(`(?i)src="(https?:\/\/\d+\.media\.tumblr\.com\/\w+\/)(tumblr_\w+_)(\d+)\.(jpe?g|gif|png|)`)
		match := re.FindSubmatch(body)

		if match == nil {
			continue
		}

		fileName := fmt.Sprintf("%s.%s", match[2][:len(match[2])-1], match[4])
		imageURL := fmt.Sprintf("%s%s1280.%s", match[1], match[2], match[4])
		var imageType string
		if string(match[4]) == "gif" {
			imageType = "video"
		} else {
			imageType = "photo"
		}
		media := model.Media{
			FileName: fileName,
			URL:      imageURL,
			Type:     imageType,
			Source:   url,
			Service:  "tumblr",
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
