package service

import (
	"sync"
)

type Type string

const (
	Twitter  Type = "Twitter"
	Tumblr        = "Tumblr"
	Pixiv         = "Pixiv"
	Danbooru      = "Danbooru"
	Dropbox       = "Dropbox"
	Telegram      = "Telegram"
	Misskey       = "Misskey"
)

type Media struct {
	FileName    string
	URL         string
	File        *[]byte `json:"-"`
	Type        string  // photo, video
	Source      string
	Service     string
	TGFileID    string `json:"-"`
	Author      string
	AuthorURL   string
	Title       string
	Description string
}

type IncomingURL struct {
	Service  Type
	Original string
	URL      string
	StrID    string
	IntID    int
}

type ProviderService interface {
	IsService(Type Type) bool
	CheckValid(urlString string) (*IncomingURL, bool)
	ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error)
}

type ConsumerService interface {
	ConsumeMedia(mediaList []*Media)
}

type AllServices struct {
	Danbooru *DanbooruService
	Pixiv    *PixivService
	Tumblr   *TumblrService
	Twitter  *TwitterService
	Misskey  *MisskeyService
	Dropbox  *DropboxService
	Telegram *TelegramService
}

type ServiceManager struct {
	Providers []ProviderService
	Consumers []ConsumerService
	All       *AllServices
}

var serviceManagerInstance *ServiceManager
var once sync.Once

func GetServiceManager() *ServiceManager {
	once.Do(func() {
		danbooru := NewDanbooruService()
		pixiv := NewPixivService()
		tumblr := NewTumblrService()
		twitter := NewTwitterService()
		misskey := NewMisskeyService()
		dropbox := NewDropboxService()
		telegram := NewTelegramService()

		allServices := &AllServices{danbooru, pixiv, tumblr, twitter, misskey, dropbox, telegram}
		providers := []ProviderService{danbooru, pixiv, tumblr, twitter, misskey}
		consumers := []ConsumerService{telegram, dropbox}

		serviceManagerInstance = &ServiceManager{
			Providers: providers,
			Consumers: consumers,
			All:       allServices,
		}
	})
	return serviceManagerInstance
}

func (s ServiceManager) BuildIncomingURL(urlList *[]string) (result []*IncomingURL) {
	for _, urlString := range *urlList {
		for _, provider := range s.Providers {
			if incomingURL, ok := provider.CheckValid(urlString); ok == true {
				result = append(result, incomingURL)
				break
			}
		}
	}

	return
}

func (s ServiceManager) ExtraMediaFromURL(incomingURLList []*IncomingURL) (result []*Media) {
	for _, incomingURL := range incomingURLList {
		for _, provider := range s.Providers {
			if !provider.IsService(incomingURL.Service) {
				continue
			}

			if media, err := provider.ExtractMediaFromURL(incomingURL); err == nil {
				result = append(result, media...)
			}
		}
	}

	return
}

func (s ServiceManager) ConsumeMedia(media []*Media) {
	for _, consumer := range s.Consumers {
		go consumer.ConsumeMedia(media)
	}
}
