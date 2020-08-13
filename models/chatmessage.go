package models

import (
	"github.com/looplab/fsm"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/dialogflow"
)

const (
	//Sources of messages
	SourceTelegram = "telegram"
	SourceBroker   = "broker"
	SourceWhatsApp = "whatsapp"
	//Telegram message types
)

//ChatMsg global chat message for using inside service
type ChatMsgFull struct {
	structures.MessageFromBot
	State    string                 `json:"type"`
	Type     string                 `json:"type"`    //тип входящего сообщение
	Order    LocalOrders            `json:"order"`   //локальная копия заказа
	Payload  interface{}            `json:"payload"` //полное содержимое сообщения
	DFAnswer dialogflow.NLPResponse //ответ от DialogFlow
	FSM      *fsm.FSM               //стейт машина
}
