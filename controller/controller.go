package controller

import "github.com/wxt2005/image-capture-bot-go/service"

type Response struct {
	Media   *[]*service.Media `json:"media"`
	Message ResponseMsg       `json:"message"`
}

type ResponseMsg string

const (
	MsgSuccess   ResponseMsg = "success"
	MsgDuplicate ResponseMsg = "duplicate"
)
