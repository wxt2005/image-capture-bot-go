package model

type Message struct {
	MessageID int `json:"message_id"`
	Text      string
	Entities  []struct {
		Offset int
		Length int
		Type   string
	}
	Chat struct {
		ID int
	} `json:"chat"`
	From struct {
		ID int
	}
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
