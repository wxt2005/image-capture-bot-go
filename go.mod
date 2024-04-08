module github.com/wxt2005/image-capture-bot-go

go 1.13

replace github.com/search2d/go-pixiv => ./vendor/go-pixiv

require (
	github.com/aws/aws-sdk-go-v2 v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.11 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.53.1 // indirect
	github.com/dropbox/dropbox-sdk-go-unofficial/v6 v6.0.5
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/google/uuid v1.3.0
	github.com/h2non/bimg v1.1.9
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/search2d/go-pixiv v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.16.0
	github.com/u2takey/ffmpeg-go v0.5.0
	go.etcd.io/bbolt v1.3.7
	golang.org/x/oauth2 v0.10.0 // indirect
)
