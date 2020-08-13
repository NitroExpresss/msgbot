// Telegram Bot Subscriber v.2.0
// Это обновленная версия обработчика входящих сообщений из телеграмма.
// На каждое новое сообщение мы заполняем структуру в которой есть вся информация о
// заказе, его контекст (статус) и сам заказ. То есть в рамках структуры
// есть вся необходимая информация для обработки

//данные хендлер принимает сообщение, отправляет его в DialogFlow если надо
//собирает всю информацию и получет ответ и отправляет этот ответ

package handler

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"strconv"
	"time"
)

const ()

//handleIncomeMsg обновленный метод обработки входящих запросов
//идея в том что мы навешываем на структуру всю возможно необходимую информацию
func (h *Handler) HandleIncomeTelegramMsg(ctx context.Context, update *tgbotapi.Update) {
	var err error

	var msgBodyToChat string

	var msg models.ChatMsgFull
	log := logs.Eloger.WithFields(logrus.Fields{
		"event": "handling income message callback 2.0",
	})

	msg.Source = models.SourceTelegram
	msg.Payload = update

	//Обрабатыаем тип сообщения
	if update.Message != nil { // ignore any non-Message Updates
		fillUpMsgData(&msg, update)
		msgBodyToChat = update.Message.Text
		//получили контакт
		if update.Message.Contact != nil {
			msg.Type = proto.MsgTypes.TelegramContact.S()
			msgBodyToChat = "contact " + update.Message.Contact.PhoneNumber
		}
		if update.Message.Location != nil {
			msg.Type = proto.MsgTypes.Coordinates.S()
			msgBodyToChat = "coordinates sended"
		}
		//коллбек
	} else if update.CallbackQuery != nil {
		fillUpCallbackData(&msg, update)
		msgBodyToChat = "callback " + update.CallbackQuery.Message.Text
	} else {
		log.WithFields(logrus.Fields{
			"reason": "unknown type. not message and not callback",
		}).Error("failed to response telegram message type")
	}

	//Получаем заказ связанные с этим сообщением
	msg.Order, err = h.GetMsgOrder(ctx, structures.MessageFromBot{
		ClientMsgID: msg.ClientMsgID,
		Source:      msg.Source,
		ChatMsgID:   msg.ChatMsgID,
	})
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason":        "failed to get order uuid",
			"client msg id": msg.ClientMsgID,
		}).Error(err)
		return
	}

	//Отправляем данные в чат для истории
	msgToCRM := structures.MessageFromBot{
		Source:       string(structures.ClientMember),
		ClientMsgID:  msg.ClientMsgID,
		UserLogin:    msg.UserLogin,
		ChatMsgID:    msg.ChatMsgID,
		MsgID:        msg.MsgID,
		Text:         msgBodyToChat,
		OrderUUID:    msg.OrderUUID,
		ClientUUID:   msg.ClientUUID,
		CreatedAt:    msg.CreatedAt,
		CreatedAtMsg: msg.CreatedAtMsg,
	}

	pubErr := h.Pub.NewMsg(&msgToCRM)
	if pubErr != nil {
		log.WithFields(logrus.Fields{
			"reason":  "failed to publish event",
			"chat id": msg.ChatMsgID,
		}).Error(err)
	}
	//

	msg.State = msg.Order.State
	msg.FSM = h.InitOrderStateFSM()
	msg.OrderUUID = msg.Order.OrderUUID

	//var answer string
	//var buttons proto.ButtonsSet

	//создаем очередь обработки сообщений, что бы все обрабатывалось последовательно
	//msgJob := h.Jobs.GetJobQueue(JobQueueNameMsgs, JobQueueLimitMsgs)
	//err = msgJob.Execute(msg.MsgID, func() error {

	//если сообщение текстовое, которое нужно понять - отправляем его в DialogFlow
	if msg.Type == proto.MsgTypes.TelegramMessage.S() {
		intentContext := swapContext(msg.State)
		msg.DFAnswer, err = h.DF.DetectIntentText(msg.Text, strconv.FormatInt(msg.ChatMsgID, 10), intentContext)
		if err != nil {
			log.WithFields(logrus.Fields{
				"reason": "failed to DFAnswer",
			}).Error(err)
		}
	}

	//Отправляем в хендлер для получения ответа
	answerTextButton, err := h.getTextAnswer(&msg)
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason":        "failed to get answer",
			"client msg id": msg.ClientMsgID,
			"msgType":       msg.Type,
			"msgText":       msg.Text,
			"msgIntent":     msg.DFAnswer.Intent,
			"msgState":      msg.State,
		}).Error(err)
	}

	if answerTextButton.textAnswer == skipStateConstant {
		return
	}

	//отпраляем ответ если 0 значит новое сообщение иначе - обновляем
	if answerTextButton.msgId == 0 {
		sndMsg, err := h.Telegram.SendMessage(msg.ChatMsgID, answerTextButton.textAnswer, answerTextButton.buttonsAnswer)
		if err != nil {
			log.WithFields(logrus.Fields{
				"reason":  "failed to send new msg",
				"chat id": msg.ChatMsgID,
				"msgText": answerTextButton.textAnswer,
			}).Error(err)
		} else {
			msgToCRM := structures.MessageFromBot{
				Source:       string(structures.TelegramBotMember),
				ClientMsgID:  msg.ClientMsgID,
				UserLogin:    msg.UserLogin,
				ChatMsgID:    msg.ChatMsgID,
				MsgID:        msg.MsgID,
				Text:         answerTextButton.textAnswer,
				OrderUUID:    msg.OrderUUID,
				ClientUUID:   msg.ClientUUID,
				CreatedAt:    msg.CreatedAt,
				CreatedAtMsg: msg.CreatedAtMsg,
			}
			pubErr := h.Pub.NewMsg(&msgToCRM)
			if pubErr != nil {
				log.WithFields(logrus.Fields{
					"reason":  "failed to publish event",
					"chat id": msg.ChatMsgID,
				}).Error(err)
			}
		}

		h.saveMsgId(ctx, sndMsg, &msg)
	} else {
		sndMsg, err := h.Telegram.UpdateMessage(msg.ChatMsgID, answerTextButton.msgId, answerTextButton.textAnswer)
		if err != nil {
			log.WithFields(logrus.Fields{
				"reason":  "failed to updated msg",
				"chat id": msg.ChatMsgID,
				"msgText": answerTextButton.textAnswer,
				"msgID":   answerTextButton.msgId,
			}).Error(err)
		}

		msgToCRM := structures.MessageFromBot{
			Source:       string(structures.TelegramBotMember),
			ClientMsgID:  msg.ClientMsgID,
			UserLogin:    msg.UserLogin,
			ChatMsgID:    msg.ChatMsgID,
			MsgID:        msg.MsgID,
			Text:         "UPD:" + answerTextButton.textAnswer,
			OrderUUID:    msg.OrderUUID,
			ClientUUID:   msg.ClientUUID,
			CreatedAt:    msg.CreatedAt,
			CreatedAtMsg: msg.CreatedAtMsg,
		}
		pubErr := h.Pub.NewMsg(&msgToCRM)
		if pubErr != nil {
			log.WithFields(logrus.Fields{
				"reason":  "failed to publish event",
				"chat id": msg.ChatMsgID,
			}).Error(err)
		}

		h.saveMsgId(ctx, sndMsg, &msg)
		if len(answerTextButton.buttonsAnswer.Buttons) > 0 {
			err = h.Telegram.UpdateKeyboard(msg.ChatMsgID, answerTextButton.msgId, answerTextButton.buttonsAnswer)
			if err != nil {
				log.WithFields(logrus.Fields{
					"reason":  "failed to update keyboard",
					"chat id": msg.ChatMsgID,
					"msgText": answerTextButton.textAnswer,
					"msgID":   answerTextButton.msgId,
				}).Error(err)
			}
		}
	}

	log.WithFields(logrus.Fields{
		"chat id":   msg.ChatMsgID,
		"answer":    answerTextButton.textAnswer,
		"msgID":     answerTextButton.msgId,
		"state":     msg.State,
		"orderUUID": msg.OrderUUID,
	}).Info("transition ok")
}

//сохраняем ID отправленного  сообщения если оно будет нужно в последующем
func (h *Handler) saveMsgId(ctx context.Context, sendedMsg proto.SendedMessage, msg *models.ChatMsgFull) {
	if len(msg.Order.OrderPrefs.MsgsIDs) == 0 {
		msg.Order.OrderPrefs.MsgsIDs = make(map[string]int)
	}
	switch msg.State {
	case proto.States.Taxi.Order.FixDeparture.S():
		msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.FixDeparture.S()] = sendedMsg.Id
	case proto.States.Taxi.Order.FixArrival.S():
		msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.FixArrival.S()] = sendedMsg.Id
	case proto.States.Taxi.Order.Departure.S():
		msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.Departure.S()] = sendedMsg.Id
	case proto.States.Taxi.Order.Arrival.S():
		msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.Arrival.S()] = sendedMsg.Id
	default:
		return
	}
	err := h.DB.SaveLocalOrder(ctx, &msg.Order)
	if err != nil {
		logs.Eloger.WithFields(logrus.Fields{
			"event":  "saving sended msg ids",
			"reason": "failed to save in DB",
		}).Error(err)
	}
	return
}

//swapContext подменяет контекст в некоторых случаях, это связано с особенностями работы DialogFlow
//входящий контекст должен быть подмножеством контекстов интента, а не просто одни из
func swapContext(state string) string {
	switch state {
	case proto.States.Taxi.Order.FixDeparture.S():
		return proto.States.Taxi.Order.CreateDraft.S()
	case proto.States.Taxi.Order.FixArrival.S():
		return proto.States.Taxi.Order.Departure.S()
	default:
		return state
	}
}

func fillUpMsgData(msg *models.ChatMsgFull, update *tgbotapi.Update) {
	msg.UserLogin = update.Message.From.UserName
	msg.Text = update.Message.Text
	msg.CreatedAt = time.Now()
	msg.ChatMsgID = update.Message.Chat.ID
	msg.ClientMsgID = strconv.Itoa(update.Message.From.ID)
	msg.CreatedAtMsg = time.Unix(int64(update.Message.Date), 0)
	msg.Type = proto.MsgTypes.TelegramMessage.S()
	msg.MsgID = update.Message.MessageID
}

func fillUpCallbackData(msg *models.ChatMsgFull, update *tgbotapi.Update) {
	msg.UserLogin = update.CallbackQuery.From.UserName
	msg.Type = proto.MsgTypes.TelegramCallback.S()
	msg.Text = update.CallbackQuery.Data
	msg.CreatedAt = time.Now()
	msg.ChatMsgID = update.CallbackQuery.Message.Chat.ID
	msg.ClientMsgID = strconv.Itoa(update.CallbackQuery.From.ID)
	msg.CreatedAtMsg = time.Unix(int64(update.CallbackQuery.Message.Date), 0)
	msg.MsgID = update.CallbackQuery.Message.MessageID
}
