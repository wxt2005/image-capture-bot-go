package service

import (
	"sync"
)

type Type string

const (
	Twitter   Type = "Twitter"
	Tumblr    Type = "Tumblr"
	Pixiv     Type = "Pixiv"
	Danbooru  Type = "Danbooru"
	// Dropbox  Type = "Dropbox"
	Telegram  Type = "Telegram"
	Misskey   Type = "Misskey"
	Bluesky   Type = "Bluesky"
	S3        Type = "S3"
	Instagram Type = "Instagram"
)

type Media struct {
	FileName    string
	URL         string
	File        *[]byte `json:"-"`
	Type        string  // photo, video, animation
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
	Host     string
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
	Danbooru  *DanbooruService
	Pixiv     *PixivService
	Tumblr    *TumblrService
	Twitter   *TwitterService
	Misskey   *MisskeyService
	Bluesky   *BlueskyService
	Instagram *InstagramService
	// Dropbox  *DropboxService
	Telegram  *TelegramService
	S3        *S3Service
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
		bluesky := NewBlueskyService()
		instagram := NewInstagramService()
		// dropbox := NewDropboxService()
		telegram := NewTelegramService()
		s3 := NewS3Service()

		allServices := &AllServices{danbooru, pixiv, tumblr, twitter, misskey, bluesky, instagram, telegram, s3}
		providers := []ProviderService{danbooru, pixiv, tumblr, twitter, misskey, bluesky, instagram}
		consumers := []ConsumerService{telegram, s3}

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
			if incomingURL, ok := provider.CheckValid(urlString); ok {
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
