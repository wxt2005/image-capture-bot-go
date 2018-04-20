package model

type Entity struct {
	Offset int
	Length int
	Type   string
}

type Photo struct {
	FileID   string `json:"file_id"`
	FileSize int    `json:"file_size"`
	Width    int
	Height   int
}

type Message struct {
	MessageID       int `json:"message_id"`
	Text            string
	Caption         string
	CaptionEntities []Entity `json:"caption_entities"`
	Entities        []Entity
	Chat            struct {
		ID int
	} `json:"chat"`
	From struct {
		ID int
	}
	Photo []Photo
}

type IncomingMessage struct {
	Message struct {
		Message
	}
}

type CallbackMessage struct {
	CallbackQuery struct {
		Message struct {
			Message
		}
		From struct {
			ID int
		}
		Data string
	} `json:"callback_query"`
}

type Updates struct {
	Result []IncomingMessage `json:"result"`
}
