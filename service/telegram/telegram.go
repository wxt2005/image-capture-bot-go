package telegram

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
	"github.com/wxt2005/image_capture_bot_go/model"
)

const endpointSendVideo = "/sendVideo"
const endpointSendPhoto = "/sendPhoto"
const endpointSendMessage = "/sendMessage"
const endpointGetUpdates = "/getUpdates"
const endpointEditMessageReplyMarkup = "/editMessageReplyMarkup"
const endpointGetFile = "/getFile"
const likeBtnText = "❤️ Like"

type apiImpl struct {
	client         *http.Client
	chatID         string
	token          string
	endpointPrefix string
}

type Service interface {
	SendDuplicateMessage(url string, chatID int, messageID int) error
	ConsumeMedias(medias []*model.Media)
	// GetUpdates() []model.IncomingMessage
	UpdateLikeButton(chatID int, messageID int, count int) error
	ExtractMediaFromMsg(msg *model.IncomingMessage) ([]*model.Media, []string, error)
}

// func (api apiImpl) GetUpdates() []model.IncomingMessage {
// 	updates := model.Updates{}
//
// 	req, err := http.NewRequest("POST", api.endpointPrefix+endpointGetUpdates, nil)
// 	req.Header.Set("Content-Type", "application/json")
//
// 	resp, err := api.client.Do(req)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()
//
// 	fmt.Println("response Status:", resp.Status)
// 	body, _ := ioutil.ReadAll(resp.Body)
// 	fmt.Println("response Body:", string(body))
// 	json.Unmarshal(body, &updates)
// 	return updates.Result
// }

func ExtractUrl(message model.IncomingMessage) []string {
	text := message.Message.Text
	var urls []string

	for _, entry := range message.Message.Entities {
		if entry.Type == "url" || entry.Type == "text_link" {
			start := entry.Offset
			end := entry.Offset + entry.Length
			urls = append(urls, string([]rune(text)[start:end]))
		}
	}

	return urls
}

type MessageRequestBody struct {
	ChatID                int    `json:"chat_id"`
	Text                  string `json:"text"`
	ReplayToMessageID     int    `json:"reply_to_message_id"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
	DisableNotificaton    bool   `json:"disable_notification"`
	ParseMode             string `json:"parse_mode"`
	ReplyMarkup           string `json:"reply_markup"`
}

type ReplyMarkup struct {
	InlineKeyboard [][]Keyboard `json:"inline_keyboard"`
}

type Keyboard struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type UpdateRequestBody struct {
	ChatID      int    `json:"chat_id"`
	MessageID   int    `json:"message_id"`
	ReplyMarkup string `json:"reply_markup"`
}

func (api apiImpl) UpdateLikeButton(chatID int, messageID int, count int) error {
	keyboard := Keyboard{
		fmt.Sprintf("%s (%d)", likeBtnText, count),
		"like",
	}

	replyMarkup := ReplyMarkup{[][]Keyboard{[]Keyboard{keyboard}}}
	replyMarkupJSON, err := json.Marshal(replyMarkup)
	if err != nil {
		return err
	}

	requestBody := UpdateRequestBody{chatID, messageID, string(replyMarkupJSON)}
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", api.endpointPrefix+endpointEditMessageReplyMarkup, bytes.NewBuffer(requestJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (api apiImpl) SendDuplicateMessage(url string, chatID int, messageID int) error {
	keyboard := Keyboard{
		"强制发送",
		"force",
	}

	replyMarkup := ReplyMarkup{[][]Keyboard{[]Keyboard{keyboard}}}
	replyMarkupJSON, _ := json.Marshal(replyMarkup)

	requestBody := MessageRequestBody{
		ChatID:                chatID,
		Text:                  "图片地址重复: <a href=\"" + url + "\">" + url + "</a>",
		ReplayToMessageID:     messageID,
		DisableWebPagePreview: true,
		DisableNotificaton:    true,
		ParseMode:             "HTML",
		ReplyMarkup:           string(replyMarkupJSON),
	}

	requestJSON, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", api.endpointPrefix+endpointSendMessage, bytes.NewBuffer(requestJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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

func (api apiImpl) ConsumeMedias(medias []*model.Media) {
	for _, media := range medias {
		var err error
		if media.File != nil {
			err = api.sendByStream(media)
		} else {
			err = api.sendByUrl(media)
		}
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Send telegram media failed")
		}
	}
}

func (api apiImpl) sendByUrl(media *model.Media) error {
	var endpoint string
	var requestBody interface{}
	keyboard := Keyboard{likeBtnText, "like"}
	replyMarkup := ReplyMarkup{[][]Keyboard{[]Keyboard{keyboard}}}
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
		endpoint = endpointSendPhoto
		requestBody = photoRequestBody{
			api.chatID,
			url,
			media.Source,
			string(replyMarkupJSON),
		}
	case "video":
		endpoint = endpointSendVideo
		requestBody = videoRequestBody{
			api.chatID,
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

	req, err := http.NewRequest("POST", api.endpointPrefix+endpoint, bytes.NewBuffer(dataJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

func (api apiImpl) sendByStream(media *model.Media) error {
	var endpoint string
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	keyboard := Keyboard{likeBtnText, "like"}
	replyMarkup := ReplyMarkup{[][]Keyboard{[]Keyboard{keyboard}}}
	replyMarkupJSON, err := json.Marshal(replyMarkup)
	if err != nil {
		return err
	}

	w.WriteField("chat_id", api.chatID)
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
		endpoint = endpointSendPhoto
	case "video":
		endpoint = endpointSendVideo
	default:
		return nil
	}

	req, err := http.NewRequest("POST", api.endpointPrefix+endpoint, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := api.client.Do(req)
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

func (api apiImpl) ExtractMediaFromMsg(msg *model.IncomingMessage) ([]*model.Media, []string, error) {
	var result []*model.Media
	var remains []string

	photo := getLargestPhoto(msg)
	fileID := photo.FileID

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s?file_id=%s", api.endpointPrefix, endpointGetFile, fileID), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := api.client.Do(req)
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
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", api.token, filePath)

	media := model.Media{
		FileName: fileName,
		URL:      fileURL,
		Type:     "photo", // support photo for now
		Service:  "telegram",
		TGFileID: fileID,
	}

	result = append(result, &media)

	return result, remains, nil
}

func New() Service {
	return apiImpl{
		client:         &http.Client{},
		chatID:         viper.GetString("telegram.channel_name"),
		token:          viper.GetString("telegram.bot_token"),
		endpointPrefix: "https://api.telegram.org/bot" + viper.GetString("telegram.bot_token"),
	}
}
