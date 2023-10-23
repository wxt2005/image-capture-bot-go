package service

import (
	"bytes"
	"fmt"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type DropboxService struct {
	client *files.Client
}

func NewDropboxService() *DropboxService {
	config := dropbox.Config{
		Token: viper.GetString("dropbox.access_token"),
	}
	db := files.New(config)

	return &DropboxService{
		client: &db,
	}
}

func (s DropboxService) ConsumeMedia(mediaList []*Media) {
	db := *s.client

	for _, media := range mediaList {
		path := fmt.Sprintf("%s/%s/%s", viper.GetString("dropbox.save_path"), media.Service, media.FileName)
		// stream upload
		if media.File != nil {
			uploadArg := *files.NewUploadArg(path)
			uploadArg.Autorename = true
			uploadArg.Mute = true
			reader := bytes.NewReader(*media.File)
			_, err := db.Upload(&uploadArg, reader)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Dropbox upload file failed")
			}
			// defer media.Reasder.Close()
		} else {
			arg := files.SaveUrlArg{
				Path: path,
				Url:  media.URL,
			}
			// omit error for now, seems like a bug
			db.SaveUrl(&arg)
		}
	}
}
