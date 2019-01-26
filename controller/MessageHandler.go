package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/boltdb/bolt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image_capture_bot_go/db"
	"github.com/wxt2005/image_capture_bot_go/model"
	"github.com/wxt2005/image_capture_bot_go/service/danbooru"
	"github.com/wxt2005/image_capture_bot_go/service/dropbox"
	"github.com/wxt2005/image_capture_bot_go/service/pixiv"
	"github.com/wxt2005/image_capture_bot_go/service/telegram"
	"github.com/wxt2005/image_capture_bot_go/service/tumblr"
	"github.com/wxt2005/image_capture_bot_go/service/twitter"
)

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	telegramService := telegram.New()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
	}
	var m model.IncomingMessage
	var cm model.CallbackMessage
	skipCheckDuplicate := false

	// handle callback
	if err := json.Unmarshal(body, &cm); err == nil {
		switch cm.CallbackQuery.Data {
		case "like":
			chatID := cm.CallbackQuery.Message.Chat.ID
			messageID := cm.CallbackQuery.Message.MessageID
			userID := cm.CallbackQuery.From.ID
			count, ok := saveLike(chatID, messageID, userID)
			if ok {
				go telegramService.UpdateLikeButton(chatID, cm.CallbackQuery.Message.MessageID, count)
			}
			return
		case "force":
			// extract Message, go through
			m = model.IncomingMessage{Message: cm.CallbackQuery.Message}
			skipCheckDuplicate = true
		}
	}

	if err := json.Unmarshal(body, &m); err != nil {
		w.WriteHeader(500)
	}

	var finalMedias []*model.Media
	urls := telegram.ExtractUrl(m)
	var duplicates []string

	if m.Message.Photo != nil {
		medias, remains, _ := telegramService.ExtractMediaFromMsg(&m)
		if len(medias) > 0 {
			finalMedias = append(finalMedias, medias...)
		}

		if len(remains) > 0 {
			urls = append(urls, remains...)
		}
	}

	urls = clearUrls(&urls)

	if skipCheckDuplicate != true {
		urls, duplicates = extractDuplicate(&urls)
	}

	if len(duplicates) > 0 {
		go sendDuplicateMessages(&duplicates, m.Message.Chat.ID, m.Message.MessageID)
	}

	imageServices := []model.ImageService{
		danbooru.New(),
		pixiv.New(),
		twitter.New(),
		tumblr.New(),
	}
	consumerServices := []model.ConsumerService{
		dropbox.New(),
		model.ConsumerService(telegramService),
	}

	for _, imageService := range imageServices {
		medias, remains, err := imageService.ExtractMedias(urls)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Extract media failed")
			continue
		}
		if medias != nil {
			finalMedias = append(finalMedias, medias...)
		}
		if remains != nil {
			urls = remains
		}
	}

	if len(finalMedias) <= 0 {
		return
	}
	for _, consumerService := range consumerServices {
		go consumerService.ConsumeMedias(finalMedias)
	}

	header := w.Header()
	header["Content-Type"] = []string{"application/json; charset=utf-8"}
	jsonString, _ := json.Marshal(finalMedias)
	fmt.Fprintf(w, string(jsonString))
}

func clearUrls(urls *[]string) (results []string) {
	for _, url := range *urls {
		// remove twitter s=xxx
		twitterReg := regexp.MustCompile(`(?i)(https?:\/\/(?:www\.)?twitter\.com\/.+?\/status\/\d+)`)
		twitterMatch := twitterReg.FindStringSubmatch(url)
		if twitterMatch != nil {
			results = append(results, twitterMatch[1])
			log.WithFields(log.Fields{
				"before": url,
				"after":  twitterMatch[1],
			}).Info("url cleared")
			break
		}

		results = append(results, url)
	}

	return
}

func extractDuplicate(urls *[]string) (remains []string, duplicates []string) {
	db.DB.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.url_bucket")))

		for _, url := range *urls {
			exist := b.Get([]byte(url))

			if exist == nil {
				remains = append(remains, url)
				b.Put([]byte(url), []byte("1"))
			} else {
				duplicates = append(duplicates, url)
			}
		}

		return nil
	})

	return
}

func sendDuplicateMessages(urls *[]string, chatID int, messageID int) {
	service := telegram.New()
	for _, url := range *urls {
		if err := service.SendDuplicateMessage(url, chatID, messageID); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Send duplicate message failed")
		}
	}
}

func saveLike(chatID int, messageID int, userID int) (count int, ok bool) {
	db.DB.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.like_bucket")))
		key := fmt.Sprintf("chat_%d_msg_%d", chatID, messageID)
		var value []int
		liked := false

		exist := b.Get([]byte(key))

		if exist != nil {
			json.Unmarshal(exist, &value)
			for _, id := range value {
				if id == userID {
					liked = true
					break
				}
			}
			if liked == false {
				value = append(value, userID)
			}
		} else {
			value = []int{userID}
		}

		if liked == false {
			json, _ := json.Marshal(value)
			b.Put([]byte(key), json)
			ok = true
		}

		count = len(value)

		return nil
	})

	return
}
