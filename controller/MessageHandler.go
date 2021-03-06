package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/etcd-io/bbolt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image-capture-bot-go/db"
	"github.com/wxt2005/image-capture-bot-go/service"
)

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	serviceManager := service.GetServiceManager()
	telegramService := serviceManager.All.Telegram
	header := w.Header()
	header["Content-Type"] = []string{"application/json; charset=utf-8"}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var output Response

	var update tgbotapi.Update
	skipCheckDuplicate := false
	err = json.Unmarshal(body, &update)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	// handle callback
	if update.CallbackQuery != nil {
		switch update.CallbackQuery.Data {
		case "like":
			chatID := update.CallbackQuery.Message.Chat.ID
			messageID := update.CallbackQuery.Message.MessageID
			userID := update.CallbackQuery.From.ID
			count, ok := saveLike(chatID, messageID, userID)
			if ok {
				go telegramService.UpdateLikeButton(chatID, messageID, count)
			}
			w.WriteHeader(200)
			return
		case "force":
			// extract Message, go through
			update.Message = update.CallbackQuery.Message
			skipCheckDuplicate = true
		}
	}

	var mediaList []*service.Media
	var duplicates []*service.IncomingURL
	urlStringList := telegramService.ExtractURL(update.Message)

	if update.Message.Photo != nil {
		media, remains, _ := telegramService.ExtractMediaFromMsg(update.Message)
		if len(media) > 0 {
			mediaList = append(mediaList, media...)
		}

		if len(remains) > 0 {
			urlStringList = remains
		}
	}

	incomingURLList := serviceManager.BuildIncomingURL(&urlStringList)

	if skipCheckDuplicate != true {
		incomingURLList, duplicates = extractDuplicate(incomingURLList)
	}

	if len(duplicates) > 0 {
		go sendDuplicateMessages(duplicates, update.Message.Chat.ID, update.Message.MessageID)
	}

	mediaList = append(mediaList, serviceManager.ExtraMediaFromURL(incomingURLList)...)

	if len(mediaList) > 0 {
		serviceManager.ConsumeMedia(mediaList)
	}

	output.Media = &mediaList
	output.Message = MsgSuccess
	jsonByte, _ := json.Marshal(output)
	fmt.Fprintf(w, string(jsonByte))
}

func extractDuplicate(incomingURLList []*service.IncomingURL) (remains []*service.IncomingURL, duplicates []*service.IncomingURL) {
	db.DB.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.url_bucket")))

		for _, incomingURL := range incomingURLList {
			urlString := strings.ToLower(incomingURL.URL)
			exist := b.Get([]byte(urlString))

			if exist == nil {
				remains = append(remains, incomingURL)
				b.Put([]byte(urlString), []byte("1"))
			} else {
				duplicates = append(duplicates, incomingURL)
			}
		}

		return nil
	})

	return
}

func sendDuplicateMessages(incomingURLList []*service.IncomingURL, chatID int64, messageID int) {
	telegramService := service.GetServiceManager().All.Telegram

	for _, incomingURL := range incomingURLList {
		log.WithField("URL", incomingURL.URL).Debug("Duplicate url")
		if err := telegramService.SendDuplicateMessage(incomingURL.URL, chatID, messageID); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Send duplicate message failed")
		}
	}
}

func saveLike(chatID int64, messageID int, userID int) (count int, ok bool) {
	db.DB.Batch(func(tx *bbolt.Tx) error {
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
