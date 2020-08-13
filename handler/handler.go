package handler

import (
	"context"
	"gitlab.com/faemproject/backend/faem/pkg/jobqueue"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/config"
	"gitlab.com/faemproject/backend/faem/services/msgbot/dialogflow"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
)

type NLP interface {
	//GetAnswer генерирует ответ на входяшее сообщение
	GetAnswer(ctx context.Context, msg *models.ChatMsgFull) (string, error)
}

const (
	JobQueueNameMsgs  = "income.msg/telegram"
	JobQueueLimitMsgs = 999
)

type Repository interface {
	MsgBotRepository
}

type Publisher interface {
	BrokerPublisher
}

type TelegramClient interface {
	TelegramInterface
}

type Handler struct {
	DB       Repository
	Pub      Publisher
	Telegram TelegramClient
	DF       *dialogflow.DFProcessor
	Config   config.Settings
	Buffers  Buffers
	Jobs     *jobqueue.JobQueues
	//OrderStateFSM *fsm.FSM
}

type Buffers struct {
	//мапа с заказами и статусами
	WIPOrders map[string]string

	//мапа с заказами и статусами
	WIPOrdersFull map[string]models.LocalOrders

	//мапа с заказами из CRM
	CRMOrders map[string]string

	//мапа с заказами из CRM
	DriverFounded map[string]string
}

type BrokerPublisher interface {
	NewMsg(msg *structures.MessageFromBot) error
	NewDraftOrder(order *models.OrderCRM) error
	StartOrder(order *models.OrderCRM) error
	ActionOnOrder(action *structures.ActionOnOrder) error
}
