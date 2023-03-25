package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type MisskeyService struct {
	Service   Type
	urlRegexp *regexp.Regexp
	client    *http.Client
	endpint   string
}

type NoteFile struct {
	ID   string
	Name string
	Type string
	URL  string
}

type NoteUser struct {
	ID       string
	Name     string
	Username string
}

type Note struct {
	ID    string
	Text  string
	User  NoteUser
	Files []NoteFile
}

func NewMisskeyService() *MisskeyService {
	return &MisskeyService{
		Service:   Misskey,
		urlRegexp: regexp.MustCompile(`(?i)https?:\/\/misskey\.io\/notes\/(\w+)`),
		client:    &http.Client{},
		endpint:   "https://misskey.io/api/notes/show",
	}
}

func (s MisskeyService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)
	if match == nil {
		return nil, false
	}

	return &IncomingURL{
		Service:  s.Service,
		Original: urlString,
		URL:      match[0],
		StrID:    match[1],
		IntID:    0,
	}, true
}

func (s MisskeyService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

func (s MisskeyService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media
	id := incomingURL.StrID

	if id == "" {
		return result, nil
	}

	var jsonStr = []byte(fmt.Sprintf(`{"noteId":"%s"}`, id))
	req, err := http.NewRequest("POST", s.endpint, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	req.Header.Set("Host", "misskey.io")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get Misskey info failed")
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get Misskey info failed")
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get Misskey info failed")
		return nil, err
	}

	note := Note{}
	if err := json.Unmarshal(body, &note); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Get Misskey info failed")
	}

	if len(note.Files) == 0 {
		return result, nil
	}

	for _, file := range note.Files {
		var resultMedia *Media
		fileType := strings.Split(file.Type, "/")[0]

		if file.Type == "image/gif" {
			resultMedia = s.extractAnimation(&file)
		} else {
			switch fileType {
			case "image":
				resultMedia = s.extractPhoto(&file)
			case "video":
				resultMedia = s.extractVideo(&file)
			default:
				continue
			}
		}

		resultMedia.Service = string(s.Service)
		resultMedia.Source = incomingURL.URL
		s.completeMediaMeta(resultMedia, &note)
		result = append(result, resultMedia)
	}

	return result, nil
}

func (s MisskeyService) completeMediaMeta(media *Media, note *Note) {
	media.Author = note.User.Name
	media.AuthorURL = fmt.Sprintf("https://misskey.io/@%s", note.User.Username)
	media.Description = note.Text
}

func (s MisskeyService) extractPhoto(file *NoteFile) *Media {
	return &Media{
		FileName: file.Name,
		URL:      file.URL,
		Type:     "photo",
	}
}

func (s MisskeyService) extractVideo(file *NoteFile) *Media {
	return &Media{
		FileName: file.Name,
		URL:      file.URL,
		Type:     "video",
	}
}

func (s MisskeyService) extractAnimation(file *NoteFile) *Media {
	return &Media{
		FileName: file.Name,
		URL:      file.URL,
		Type:     "animation",
	}
}
