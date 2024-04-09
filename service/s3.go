package service

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

type S3Service struct {
	clinet *s3.Client
}

func NewS3Service() *S3Service {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(viper.GetString("s3.region")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(viper.GetString("s3.access_key_id"), viper.GetString("s3.secret_access_key"), "")),
	)

	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	return &S3Service{
		clinet: s3.NewFromConfig(cfg),
	}
}

func (s S3Service) ConsumeMedia(mediaList []*Media) {
	for _, media := range mediaList {
		path := fmt.Sprintf("%s/%s/%s", viper.GetString("s3.save_path"), strings.ToLower(media.Service), media.FileName)
		// stream upload
		if media.File != nil {
			_, err := s.clinet.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: aws.String(viper.GetString("s3.bucket")),
				Key:    aws.String(path),
				Body:   bytes.NewReader(*media.File),
			})
			if err != nil {
				log.Fatalf("unable to upload file, %v", err)
			}
		} else {
			// request media.URL first then upload
			req, err := http.NewRequest("GET", media.URL, nil)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Get pixiv image failed")
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Get pixiv image failed")
				continue
			}
			defer resp.Body.Close()
			file, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Get pixiv image failed")
				continue
			}

			_, err = s.clinet.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: aws.String(viper.GetString("s3.bucket")),
				Key:    aws.String(path),
				Body:   bytes.NewReader(file),
			})
			if err != nil {
				log.Fatalf("unable to upload file, %v", err)
			}
		}
	}
}
