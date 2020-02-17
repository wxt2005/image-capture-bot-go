package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image-capture-bot-go/model"
)

const telegramEndpointSendVideo = "/sendVideo"
const telegramEndpointSendPhoto = "/sendPhoto"
const telegramEndpointSendMessage = "/sendMessage"
const telegramEndpointGetUpdates = "/getUpdates"
const telegramEndpointEditMessageReplyMarkup = "/editMessageReplyMarkup"
const telegramEndpointGetFile = "/getFile"
const telegramLikeBtnText = "❤️ Like"

type TelegramService struct {
	Service        Type
	client         *http.Client
	chatID         string
	token          string
	endpointPrefix string
}

func NewTelegramService() *TelegramService {
	return &TelegramService{
		Service:        Telegram,
		client:         &http.Client{},
		chatID:         viper.GetString("telegram.channel_name"),
		token:          viper.GetString("telegram.bot_token"),
		endpointPrefix: "https://api.telegram.org/bot" + viper.GetString("telegram.bot_token"),
	}
}

// type Service interface {
// 	SendDuplicateMessage(url string, chatID int, messageID int) error
// 	ConsumeMedias(medias []*model.Media)
// 	UpdateLikeButton(chatID int, messageID int, count int) error
// 	ExtractMediaFromMsg(msg *model.IncomingMessage) ([]*model.Media, []string, error)
// }

func (s TelegramService) ExtractURLWithEntities(text string, entities []model.Entity) []string {
	var urls []string

	if len(text) == 0 {
		return nil
	}

	for _, entity := range entities {
		if entity.Type == "url" || entity.Type == "text_link" {
			start := entity.Offset
			end := entity.Offset + entity.Length
			urls = append(urls, string([]rune(text)[start:end]))
		}
	}

	return urls
}

func (s TelegramService) ExtractUrl(message model.IncomingMessage) []string {
	return s.ExtractURLWithEntities(message.Message.Text, message.Message.Entities)
}

type TelegramMessageRequestBody struct {
	ChatID                int    `json:"chat_id"`
	Text                  string `json:"text"`
	ReplayToMessageID     int    `json:"reply_to_message_id"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
	DisableNotificaton    bool   `json:"disable_notification"`
	ParseMode             string `json:"parse_mode"`
	ReplyMarkup           string `json:"reply_markup"`
}

type TelegramReplyMarkup struct {
	InlineKeyboard [][]TelegramKeyboard `json:"inline_keyboard"`
}

type TelegramKeyboard struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type TelegramUpdateRequestBody struct {
	ChatID      int    `json:"chat_id"`
	MessageID   int    `json:"message_id"`
	ReplyMarkup string `json:"reply_markup"`
}

func (s TelegramService) UpdateLikeButton(chatID int, messageID int, count int) error {
	keyboard := TelegramKeyboard{
		fmt.Sprintf("%s (%d)", telegramLikeBtnText, count),
		"like",
	}

	replyMarkup := TelegramReplyMarkup{[][]TelegramKeyboard{[]TelegramKeyboard{keyboard}}}
	replyMarkupJSON, err := json.Marshal(replyMarkup)
	if err != nil {
		return err
	}

	requestBody := TelegramUpdateRequestBody{chatID, messageID, string(replyMarkupJSON)}
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.endpointPrefix+telegramEndpointEditMessageReplyMarkup, bytes.NewBuffer(requestJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s TelegramService) SendDuplicateMessage(url string, chatID int, messageID int) error {
	keyboard := TelegramKeyboard{
		"强制发送",
		"force",
	}

	replyMarkup := TelegramReplyMarkup{[][]TelegramKeyboard{[]TelegramKeyboard{keyboard}}}
	replyMarkupJSON, _ := json.Marshal(replyMarkup)

	requestBody := TelegramMessageRequestBody{
		ChatID:                chatID,
		Text:                  "图片地址重复: <a href=\"" + url + "\">" + url + "</a>",
		ReplayToMessageID:     messageID,
		DisableWebPagePreview: true,
		DisableNotificaton:    true,
		ParseMode:             "HTML",
		ReplyMarkup:           string(replyMarkupJSON),
	}

	requestJSON, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", s.endpointPrefix+telegramEndpointSendMessage, bytes.NewBuffer(requestJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	log.Info(bodyString)

	return nil
}

type photoRequestBody struct {
	ChatID      string `json:"chat_id"`
	Photo       string `json:"photo"`
	Caption     string `json:"caption"`
	ReplyMarkup string `json:"reply_markup"`
}

type videoRequestBody struct {
	ChatID      string `json:"chat_id"`
	Video       string `json:"video"`
	Caption     string `json:"caption"`
	ReplyMarkup string `json:"reply_markup"`
}

func (s TelegramService) ConsumeMedia(mediaList []*model.Media) {
	for _, media := range mediaList {
		var err error
		if media.File != nil {
			err = s.sendByStream(media)
		} else {
			err = s.sendByUrl(media)
		}
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Send telegram media failed")
		}
	}
}

func (s TelegramService) sendByUrl(media *model.Media) error {
	var endpoint string
	var requestBody interface{}
	keyboard := TelegramKeyboard{telegramLikeBtnText, "like"}
	replyMarkup := TelegramReplyMarkup{[][]TelegramKeyboard{[]TelegramKeyboard{keyboard}}}
	replyMarkupJSON, err := json.Marshal(replyMarkup)

	if err != nil {
		return err
	}

	url := media.URL
	if len(media.TGFileID) != 0 {
		url = media.TGFileID
	}

	switch media.Type {
	case "photo":
		endpoint = telegramEndpointSendPhoto
		requestBody = photoRequestBody{
			s.chatID,
			url,
			media.Source,
			string(replyMarkupJSON),
		}
	case "video":
		endpoint = telegramEndpointSendVideo
		requestBody = videoRequestBody{
			s.chatID,
			url,
			media.Source,
			string(replyMarkupJSON),
		}
	default:
		return nil
	}

	dataJSON, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.endpointPrefix+endpoint, bytes.NewBuffer(dataJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

func (s TelegramService) sendByStream(media *model.Media) error {
	var endpoint string
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	keyboard := TelegramKeyboard{telegramLikeBtnText, "like"}
	replyMarkup := TelegramReplyMarkup{[][]TelegramKeyboard{[]TelegramKeyboard{keyboard}}}
	replyMarkupJSON, err := json.Marshal(replyMarkup)
	if err != nil {
		return err
	}

	w.WriteField("chat_id", s.chatID)
	w.WriteField("caption", media.Source)
	w.WriteField("reply_markup", string(replyMarkupJSON))
	fw, err := w.CreateFormFile(media.Type, media.FileName)
	if err != nil {
		return err
	}
	fw.Write(*media.File)
	w.Close()

	switch media.Type {
	case "photo":
		endpoint = telegramEndpointSendPhoto
	case "video":
		endpoint = telegramEndpointSendVideo
	default:
		return nil
	}

	req, err := http.NewRequest("POST", s.endpointPrefix+endpoint, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

func getLargestPhoto(msg *model.IncomingMessage) *model.Photo {
	maxH := 0
	maxW := 0
	var result *model.Photo

	for _, photo := range msg.Message.Photo {
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

type fileRes struct {
	Ok     bool
	Result struct {
		FileID   string `json:"file_id"`
		FileSize int    `json:"file_size"`
		FilePath string `json:"file_path"`
	}
}

func (s TelegramService) ExtractMediaFromMsg(msg *model.IncomingMessage) ([]*model.Media, []string, error) {
	var result []*model.Media
	var remains []string
	manager := GetServiceManager()

	if captionURLs := s.ExtractURLWithEntities(msg.Message.Caption, msg.Message.CaptionEntities); captionURLs != nil {
		for _, url := range captionURLs {
			_, danbooruValid := manager.All.Danbooru.CheckValid(url)
			_, twitterValid := manager.All.Twitter.CheckValid(url)
			_, pixivValid := manager.All.Pixiv.CheckValid(url)

			if danbooruValid || twitterValid || pixivValid {
				remains = append(remains, url)
			}
		}

		if len(remains) == len(captionURLs) {
			return result, remains, nil
		}
	}

	photo := getLargestPhoto(msg)
	fileID := photo.FileID

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s?file_id=%s", s.endpointPrefix, telegramEndpointGetFile, fileID), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var m fileRes
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, nil, err
	}

	filePath := m.Result.FilePath
	if len(filePath) == 0 {
		return nil, nil, errors.New("no result file path")
	}

	urlParts := strings.Split(filePath, "/")
	fileName := urlParts[len(urlParts)-1]
	fileURL := fmt.Sprintf("https://s.telegram.org/file/bot%s/%s", s.token, filePath)

	media := model.Media{
		FileName: fileName,
		URL:      fileURL,
		Type:     "photo", // support photo for now
		Service:  string(s.Service),
		TGFileID: fileID,
	}

	result = append(result, &media)

	return result, remains, nil
}
