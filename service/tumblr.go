package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

type TumblrService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *http.Client
}

func NewTumblrService() *TumblrService {
	return &TumblrService{
		Service:   Tumblr,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/\w+\.tumblr\.com\/post\/.+`),
		client:    &http.Client{},
	}
}

func (s TumblrService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	return &IncomingURL{
		Service:  Tumblr,
		Original: urlString,
		URL:      match[0],
	}, true
}

func (s TumblrService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s TumblrService) ExtractMediaFromURL(incomingURL *IncomingURL) (result []*Media, err error) {
	req, err := http.NewRequest("GET", incomingURL.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?i)src="(https?:\/\/\d+\.media\.tumblr\.com\/\w+\/)(tumblr_\w+_)(\d+)\.(jpe?g|gif|png|)`)
	match := re.FindSubmatch(body)

	if match == nil {
		return
	}

	fileName := fmt.Sprintf("%s.%s", match[2][:len(match[2])-1], match[4])
	imageURL := fmt.Sprintf("%s%s1280.%s", match[1], match[2], match[4])
	var imageType string
	if string(match[4]) == "gif" {
		imageType = "video"
	} else {
		imageType = "photo"
	}
	media := Media{
		FileName: fileName,
		URL:      imageURL,
		Type:     imageType,
		Source:   incomingURL.URL,
		Service:  string(s.Service),
	}
	result = append(result, &media)

	return result, nil
}
