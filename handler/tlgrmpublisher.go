package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

const separator string = ";"

type TelegramInterface interface {
	// Отправляем сообщение в чат, опция - клавиатура
	SendMessage(chatID int64, msg string, keyboard ...proto.ButtonsSet) (proto.SendedMessage, error)
	// Отправляем сообщение пришедшее от оператора
	SendOperatorMessage(chatID int64, msg string) error
	// Обновляем сообщение
	UpdateMessage(chatID int64, msgId int, msg string) (proto.SendedMessage, error)
	// Ставим новую клаву
	UpdateKeyboard(chatID int64, msgID int, newKeyborad proto.ButtonsSet) error
	// Удаляем сообщение целиком
	DeleteMsg(chatID int64, msgID int) error
}

//SendToBotClient отправляем клиенту сообщение
func (h *Handler) SendToBotClient(ctx context.Context, sendMsg structures.ChatMessages) error {

	locOrder, err := h.DB.GetLocalOrderByUUID(ctx, sendMsg.OrderUUID)
	if err != nil {
		return errors.Wrap(err, "Cant get chatID from DB")
	}
	// // если оператор написал, сменить статус на CreatingOrderWithOperator
	// _, err = h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: sendMsg.OrderUUID, State: string(proto.Consts.Order.CreationStates.ProcessingWithOperator)})
	// if err != nil {
	// 	return errpath.Err(err)
	// }

	switch locOrder.Source {
	case "telegram":
		chatID, err := strconv.ParseInt(locOrder.ChatMsgId, 10, 64)
		if err != nil {
			er := fmt.Sprintf("Cant convert chat id (%s) to int64", locOrder.ChatMsgId)
			return errors.New(er)
		}
		err = h.Telegram.SendOperatorMessage(chatID, sendMsg.Message)
		if err != nil {
			errors.Wrap(err, "Cant send message to Telegram API")
		}
	default:
		return errors.New("Unknown message source")
	}

	logs.Eloger.WithFields(logrus.Fields{
		"event":    "handling new message to telegram",
		"chatUUID": sendMsg.OrderUUID,
		"message":  sendMsg.Message,
	}).Debug("Sended!")

	return nil
}

// ButtonDataAddValues -
func ButtonDataAddValues(data string, vals ...string) string {
	for _, item := range vals {
		if data != "" {
			data += separator + item
		} else {
			data = item
		}
	}
	return data
}

func cutButtonData(data string) string {
	n := proto.ButtonDataSize - 4 // -4 ибо 1 знак идет как 3 символа
	if len(data) > n {
		return data[:n]
	}
	return data
}

// ButtonDataGetValues -
func ButtonDataGetValues(data string) (string, string) {
	btnData := strings.Split(data, separator)

	var action string
	if len(btnData) >= 1 {
		action = btnData[0]
	}

	var btnValue string
	if len(btnData) >= 2 {
		btnValue = btnData[1]
	}

	return action, btnValue
}

// ButtonDataGetValues -
func parseButtonDataValues(data string) []string {
	spdata := strings.Split(data, separator)
	return spdata
}
