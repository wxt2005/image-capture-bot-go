package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/h2non/bimg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const telegramPhotoSize = 10 * 1024 * 1024 // Photo size is 10MB
const telegramResizeRatio = 0.8
const retryLimit = 5

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
		return urls
	}

	if entities == nil {
		return urls
	}

	for _, entity := range *entities {
		if entity.Type == "url" {
			start := entity.Offset
			end := entity.Offset + entity.Length
			// Telegram is using utf16 encoding to calculate the offset and length
			utf16Runes := utf16.Encode([]rune(text))
			extractedUrl := string(utf16.Decode(utf16Runes[start:end]))
			urls = append(urls, extractedUrl)
		}
		if entity.Type == "text_link" {
			urls = append(urls, entity.URL)
		}
	}

	return urls
}

func (s TelegramService) ExtractURL(msg *tgbotapi.Message) []string {
	var allKeys = make(map[string]bool)
	var urls = []string{}

	if msg.Text != "" && msg.Entities != nil {
		for _, url := range s.ExtractURLWithEntities(msg.Text, &msg.Entities) {
			if allKeys[url] {
				continue
			}
			allKeys[url] = true
			urls = append(urls, url)
		}
	}

	if msg.Caption != "" && msg.CaptionEntities != nil {
		for _, url := range s.ExtractURLWithEntities(msg.Caption, &msg.CaptionEntities) {
			if allKeys[url] {
				continue
			}
			allKeys[url] = true
			urls = append(urls, url)
		}
	}

	return urls
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

func (s TelegramService) SendAuthMessage(chatID int64, messageID int, isSuccess bool) error {
	var config tgbotapi.MessageConfig
	if isSuccess {
		config = tgbotapi.NewMessage(chatID, "授权成功")
	} else {
		config = tgbotapi.NewMessage(chatID, "授权失败")
	}

	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Send auth message failed")
	}

	return err
}

func (s TelegramService) SendWelcomeMessage(chatID int64, messageID int) error {
	config := tgbotapi.NewMessage(chatID, "meow")

	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Send auth message failed")
	}

	return err
}

func (s TelegramService) SendNoPremissionMessage(chatID int64, messageID int) error {
	config := tgbotapi.NewMessage(chatID, "您没有执行此操作的权限，请联系管理员")

	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Send auth message failed")
	}

	return err
}

func (s TelegramService) SendRevokeMessage(chatID int64, messageID int, isSuccess bool) error {
	var config tgbotapi.MessageConfig
	if isSuccess {
		config = tgbotapi.NewMessage(chatID, "解除授权成功")
	} else {
		config = tgbotapi.NewMessage(chatID, "解除授权失败")
	}

	_, err := s.bot.Send(config)

	if err != nil {
		jsonByte, _ := json.Marshal(config)
		log.WithFields(log.Fields{
			"config": string(jsonByte),
			"error":  err,
		}).Error("Send revoke message failed")
	}

	return err
}

func (s TelegramService) ConsumeMedia(mediaList []*Media) {
	for _, media := range mediaList {
		var err error
		if media.File != nil {
			err = s.sendByStream(media, false, 0)
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
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileURL(url),
			},
		}
	case "video":
		config = tgbotapi.VideoConfig{
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileURL(url),
			},
		}
	case "animation":
		config = tgbotapi.AnimationConfig{
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileURL(url),
			},
		}
	default:
		return nil
	}

	_, err := s.bot.Send(config)

	if err != nil {
		log.WithFields(log.Fields{
			"url":   media.URL,
			"error": err,
		}).Error("Send image by url failed")
	}

	return err
}

func (s TelegramService) sendByStream(media *Media, forceRisze bool, retryCount int) error {
	likeButton := tgbotapi.NewInlineKeyboardButtonData(s.likeBtnText, "like")
	keyboardRow := tgbotapi.NewInlineKeyboardRow(likeButton)
	keyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	var config tgbotapi.Chattable

	switch media.Type {
	case "photo":
		imageFile := *media.File
		for len(imageFile) >= telegramPhotoSize {
			log.Info("Photo too large, start compress")
			imageFile = *zoomeLargeImage(&imageFile, telegramResizeRatio)
			log.WithField("Resized Size", len(imageFile)).Info("Resized")
		}
		if forceRisze && retryCount < retryLimit {
			imageFile = *zoomeLargeImage(&imageFile, telegramResizeRatio)
			log.WithField("Resized Size", len(imageFile)).Info("Force Resized")
		}
		config = tgbotapi.PhotoConfig{
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileReader{
					Name:   media.FileName,
					Reader: bytes.NewReader(imageFile),
				},
			},
		}
	case "video":
		config = tgbotapi.VideoConfig{
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileReader{
					Name:   media.FileName,
					Reader: bytes.NewReader(*media.File),
				},
			},
		}
	case "animation":
		config = tgbotapi.AnimationConfig{
			Caption:   generateCaption(media),
			ParseMode: "MarkdownV2",
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: s.channelName,
					ReplyMarkup:     keyboardMarkup,
				},
				File: tgbotapi.FileReader{
					Name:   media.FileName,
					Reader: bytes.NewReader(*media.File),
				},
			},
		}
	default:
		return nil
	}

	_, err := s.bot.Send(config)

	if err != nil {
		log.WithFields(log.Fields{
			"url":   media.URL,
			"error": err,
		}).Error("Send image by stream failed")

		// if message includes PHOTO_INVALID_DIMENSIONS, try to send again
		if strings.Contains(err.Error(), "PHOTO_INVALID_DIMENSIONS") {
			log.Info("Try to send again")
			s.sendByStream(media, true, retryCount+1)
		}
	}

	return err
}

func generateCaption(media *Media) string {
	result := ""
	if media.Title != "" {
		result += ("*" + escape(media.Title) + "*\n")
	}
	if media.Description != "" {
		result += ("" + escape(media.Description) + "\n")
	}
	if media.Author != "" {
		result += "作者: "
		authorText := escape(media.Author)
		if media.AuthorURL != "" {
			result += ("[" + authorText + "](" + media.AuthorURL + ")\n")
		} else {
			result += (authorText + "\n")
		}
	} else {
		result += "\n"
	}
	if media.Source != "" {
		result += ("来源: [" + media.Service + "](" + media.Source + ")\n")
	}
	return result
}

func escape(origin string) string {
	var arrowRe = regexp.MustCompile(`<.+?>`)
	var escapeRe = regexp.MustCompile("(\\.|_|\\*|\\[|\\]|\\(|\\)|\\~|>|#|\\+|-|=|\\||\\{|\\}|!|`)")
	s := arrowRe.ReplaceAllString(origin, " ")
	s = escapeRe.ReplaceAllString(s, `\$1`)
	return s
}

func zoomeLargeImage(original *[]byte, ratio float64) *[]byte {
	log.WithField("ratio", ratio).Debug("Zoom")
	image := bimg.NewImage(*original)
	imageDimension, err := image.Size()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return &[]byte{}
	}

	log.WithFields(log.Fields{
		"Width":  imageDimension.Width,
		"Height": imageDimension.Height,
	}).Debug("Original image dimension")
	newWidth := int(float64(imageDimension.Width) * ratio)
	newHeight := int(float64(imageDimension.Height) * ratio)
	log.WithFields(log.Fields{
		"Width":  newWidth,
		"Height": newHeight,
	}).Debug("New image dimension")

	newImage, err := image.Resize(newWidth, newHeight)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return &[]byte{}
	}

	return &newImage
}

func getLargestPhoto(msg *tgbotapi.Message) *tgbotapi.PhotoSize {
	maxH := 0
	maxW := 0
	var result *tgbotapi.PhotoSize

	for _, photo := range msg.Photo {
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
