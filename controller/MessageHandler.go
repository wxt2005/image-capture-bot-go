package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image-capture-bot-go/db"
	"github.com/wxt2005/image-capture-bot-go/service"
	"go.etcd.io/bbolt"
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

	userID := update.Message.From.ID

	// Handle "/start" command
	if update.Message.Command() == "start" {
		go telegramService.SendWelcomeMessage(update.Message.Chat.ID, update.Message.MessageID)
		return
	}

	// Handle "/auth xxxx" command
	if update.Message.Command() == "auth" {
		key := update.Message.CommandArguments()
		isSuccess := false
		if key == viper.GetString("telegram.auth_key") {
			isSuccess = saveUserAuth(userID)
		}

		if isSuccess {
			go telegramService.SendAuthMessage(update.Message.Chat.ID, update.Message.MessageID, true)
		} else {
			go telegramService.SendAuthMessage(update.Message.Chat.ID, update.Message.MessageID, false)
		}
		return
	}

	// Handle "/revoke" command
	if update.Message.Command() == "revoke" {
		isSuccess := revokeUserAuth(userID)
		if isSuccess {
			go telegramService.SendRevokeMessage(update.Message.Chat.ID, update.Message.MessageID, true)
		} else {
			go telegramService.SendRevokeMessage(update.Message.Chat.ID, update.Message.MessageID, false)
		}
		return
	}

	// Check auth
	if !isUserAuthed(userID) {
		go telegramService.SendNoPremissionMessage(update.Message.Chat.ID, update.Message.MessageID)
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

	incomingURLList := serviceManager.BuildIncomingURL(&urlStringList)

	if !skipCheckDuplicate {
		incomingURLList, duplicates = extractDuplicate(incomingURLList)
	}

	if len(duplicates) > 0 {
		go sendDuplicateMessages(duplicates, update.Message.Chat.ID, update.Message.MessageID)
	}

	mediaList = append(mediaList, serviceManager.ExtraMediaFromURL(incomingURLList)...)

	if len(mediaList) == 0 && update.Message.Photo != nil {
		media, remains, _ := telegramService.ExtractMediaFromMsg(update.Message)
		if len(media) > 0 {
			mediaList = append(mediaList, media...)
		}

		if len(remains) > 0 {
			urlStringList = remains
		}
	}

	if len(mediaList) > 0 {
		serviceManager.ConsumeMedia(mediaList)
	}

	output.Media = &mediaList
	output.Message = MsgSuccess
	jsonByte, _ := json.Marshal(output)
	fmt.Fprintf(w, string(jsonByte))
}

func isUserAuthed(userID int64) (authed bool) {
	authed = false

	db.DB.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.auth_bucket")))
		if (b.Get([]byte(fmt.Sprintf("%d", userID)))) != nil {
			authed = true
		}
		return nil
	})
	return
}

func saveUserAuth(userID int64) bool {
	err := db.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.auth_bucket")))
		key := fmt.Sprintf("%d", userID)
		err := b.Put([]byte(key), []byte("1"))
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Save user auth failed")
		return false
	}
	return true
}

func revokeUserAuth(userID int64) bool {
	err := db.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.auth_bucket")))
		key := fmt.Sprintf("%d", userID)
		err := b.Delete([]byte(key))
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Revoke user auth failed")
		return false
	}
	return true
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

func saveLike(chatID int64, messageID int, userID int64) (count int, ok bool) {
	db.DB.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(viper.GetString("db.like_bucket")))
		key := fmt.Sprintf("chat_%d_msg_%d", chatID, messageID)
		var value []int64
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
			if !liked {
				value = append(value, userID)
			}
		} else {
			value = []int64{userID}
		}

		if !liked {
			json, _ := json.Marshal(value)
			b.Put([]byte(key), json)
			ok = true
		}

		count = len(value)

		return nil
	})

	return
}
