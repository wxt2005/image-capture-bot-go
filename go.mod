module github.com/wxt2005/image-capture-bot-go

go 1.13

replace github.com/search2d/go-pixiv => ./vendor/go-pixiv

require (
	github.com/dropbox/dropbox-sdk-go-unofficial v5.6.0+incompatible
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/google/uuid v1.3.0
	github.com/h2non/bimg v1.1.9
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/search2d/go-pixiv v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.16.0
	github.com/u2takey/ffmpeg-go v0.5.0 // indirect
	go.etcd.io/bbolt v1.3.7
	golang.org/x/oauth2 v0.10.0 // indirect
)
