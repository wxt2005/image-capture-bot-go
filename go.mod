module github.com/wxt2005/image-capture-bot-go

go 1.13

replace github.com/search2d/go-pixiv => ./vendor/go-pixiv

require (
	github.com/ChimeraCoder/anaconda v2.0.0+incompatible
	github.com/ChimeraCoder/tokenbucket v0.0.0-20131201223612-c5a927568de7 // indirect
	github.com/azr/backoff v0.0.0-20160115115103-53511d3c7330 // indirect
	github.com/dropbox/dropbox-sdk-go-unofficial v5.6.0+incompatible
	github.com/dustin/go-jsonpointer v0.0.0-20160814072949-ba0abeacc3dc // indirect
	github.com/dustin/gojson v0.0.0-20160307161227-2e71ec9dd5ad // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/h2non/bimg v1.1.9
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/search2d/go-pixiv v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.16.0
	go.etcd.io/bbolt v1.3.7
	golang.org/x/oauth2 v0.10.0 // indirect
)
