package service

import (
	"encoding/json"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const telegramEndpointGetFile = "/getFile"

type TelegramService struct {
	Service        Type
	channelName    string
	token          string
	endpointPrefix string
	bot            *tgbotapi.BotAPI
	likeBtnText    string
	likeBtnAction  string
	forceBtnText   string
	forceBtnAction string
}

func NewTelegramService() *TelegramService {
	bot, err := tgbotapi.NewBotAPI(viper.GetString("telegram.bot_token"))
	if err != nil {
		log.WithError(err).Error("Initialize bot api failed")
	}

	return &TelegramService{
		Service:        Telegram,
		channelName:    viper.GetString("telegram.channel_name"),
		token:          viper.GetString("telegram.bot_token"),
		endpointPrefix: "https://api.telegram.org/bot" + viper.GetString("telegram.bot_token"),
		bot:            bot,
		likeBtnText:    "❤️ Like",
		likeBtnAction:  "like",
		forceBtnText:   "强制发送",
		forceBtnAction: "force",
	}
}

func (s TelegramService) ExtractURLWithEntities(text string, entities *[]tgbotapi.MessageEntity) []string {
	var urls []string

	if len(text) == 0 {
		return nil
	}

	if entities == nil {
		return nil
	}

	for _, entity := range *entities {
		if entity.Type == "url" || entity.Type == "text_link" {
			start := entity.Offset
			end := entity.Offset + entity.Length
			urls = append(urls, string([]rune(text)[start:end]))
		}
	}

	return urls
}

func (s TelegramService) ExtractURL(msg *tgbotapi.Message) []string {
	return s.ExtractURLWithEntities(msg.Text, msg.Entities)
}

func (s TelegramService) UpdateLikeButton(chatID int64, messageID int, count int) error {
	keyboardButton := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s (%d)", s.likeBtnText, count), s.likeBtnAction)
	keyboardRow := tgbotapi.NewInlineKeyboardRow(keyboardButton)
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	config := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboardMarkup)
	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Update like button failed")
	}

	return err
}

func (s TelegramService) SendDuplicateMessage(url string, chatID int64, messageID int) error {
	keyboardButton := tgbotapi.NewInlineKeyboardButtonData(s.forceBtnText, s.forceBtnAction)
	keyboardRow := tgbotapi.NewInlineKeyboardRow(keyboardButton)
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	config := tgbotapi.NewMessage(chatID, fmt.Sprintf("图片地址重复: <a href=\"%s\">%s</a>", url, url))
	config.DisableWebPagePreview = true
	config.DisableNotification = true
	config.ParseMode = tgbotapi.ModeHTML
	config.ReplyToMessageID = messageID
	config.ReplyMarkup = keyboardMarkup

	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Send duplicate message failed")
	}

	return err
}

func (s TelegramService) ConsumeMedia(mediaList []*Media) {
	for _, media := range mediaList {
		var err error
		if media.File != nil {
			err = s.sendByStream(media)
		} else {
			err = s.sendByURL(media)
		}
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Send telegram media failed")
		}
	}
}

func (s TelegramService) sendByURL(media *Media) error {
	keyboardButton := tgbotapi.NewInlineKeyboardButtonData(s.likeBtnText, s.likeBtnAction)
	keyboardRow := tgbotapi.NewInlineKeyboardRow(keyboardButton)
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	var config tgbotapi.Chattable

	url := media.URL
	if len(media.TGFileID) != 0 {
		url = media.TGFileID
	}

	switch media.Type {
	case "photo":
		config = tgbotapi.PhotoConfig{
			Caption: media.Source,
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				FileID:      url,
				UseExisting: true,
			},
		}
	case "video":
		config = tgbotapi.VideoConfig{
			Caption: media.Source,
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				FileID:      url,
				UseExisting: true,
			},
		}
	default:
		return nil
	}

	_, err := s.bot.Send(config)

	if err != nil {
		log.WithFields(log.Fields{
			"config": config,
			"error":  err,
		}).Error("Send image by url failed")
	}

	return err
}

func (s TelegramService) sendByStream(media *Media) error {
	keyboardButton := tgbotapi.NewInlineKeyboardButtonData(s.likeBtnText, "like")
	keyboardRow := tgbotapi.NewInlineKeyboardRow(keyboardButton)
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	var config tgbotapi.Chattable

	switch media.Type {
	case "photo":
		config = tgbotapi.PhotoConfig{
			Caption: media.Source,
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileBytes{
					Name:  media.FileName,
					Bytes: *media.File,
				},
				UseExisting: false,
			},
		}
	case "video":
		config = tgbotapi.VideoConfig{
			Caption: media.Source,
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileBytes{
					Name:  media.FileName,
					Bytes: *media.File,
				},
				UseExisting: false,
			},
		}
	default:
		return nil
	}

	_, err := s.bot.Send(config)

	if err != nil {
		log.WithFields(log.Fields{
			"config": config,
			"error":  err,
		}).Error("Send image by stream failed")
	}

	return err
}

func getLargestPhoto(msg *tgbotapi.Message) *tgbotapi.PhotoSize {
	maxH := 0
	maxW := 0
	var result *tgbotapi.PhotoSize

	for _, photo := range *msg.Photo {
		if photo.Width > maxW {
			maxW = photo.Width
			result = &photo
		} else if photo.Height > maxH {
			maxH = photo.Height
			result = &photo
		}
	}

	return result
}

func (s TelegramService) ExtractMediaFromMsg(msg *tgbotapi.Message) (result []*Media, remains []string, err error) {
	photo := getLargestPhoto(msg)
	fileID := photo.FileID
	filePath, err := s.bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, nil, err
	}
	urlParts := strings.Split(filePath, "/")
	fileName := urlParts[len(urlParts)-1]

	media := Media{
		FileName: fileName,
		URL:      filePath,
		Type:     "photo", // support photo for now
		Service:  string(s.Service),
		TGFileID: fileID,
	}

	result = append(result, &media)
	return result, remains, nil
}
