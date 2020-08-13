package handler

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"strconv"
)

const (
	skipStateConstant = "skip it"
)

//BrokerMsgHandler специальный хендлер для обработки сообщений из брокера
func (h *Handler) BrokerMsgHandler(ctx context.Context, order *models.LocalOrders, newState proto.Constant) error {
	var err error
	var msg models.ChatMsgFull
	log := logs.Eloger.WithFields(logrus.Fields{
		"event": "handling broker message",
	})
	msg.Source = models.SourceBroker
	msg.Type = proto.MsgTypes.BrokerMessage.S()

	//это нужно что бы как то сохранить состояния
	msg.Order.OrderPrefs.NewState = newState.S()

	msg.FSM = h.InitOrderStateFSM()
	msg.OrderUUID = order.OrderUUID
	msg.ClientMsgID = order.ClientMsgID
	msg.State = order.State
	//msg.ChatMsgID = order.ClientMsgID
	msg.ChatMsgID, _ = strconv.ParseInt(order.ClientMsgID, 10, 64)

	//var answer string
	//var buttons proto.ButtonsSet

	//Отправляем в хендлер для получения ответа

	answerType, err := h.getTextAnswer(&msg)
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason":        "failed to get answer",
			"client msg id": msg.ClientMsgID,
			"msgText":       msg.Text,
			"msgState":      msg.State,
		}).Error(err)

		return errors.Wrap(err, "Failed to get Answer")
	}

	if answerType.textAnswer == skipStateConstant {
		return nil
	}

	//отправляем ответ
	_, err = h.Telegram.SendMessage(msg.ChatMsgID, answerType.textAnswer, answerType.buttonsAnswer)
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason":  "failed to send msg",
			"chat id": msg.ChatMsgID,
			"msgText": answerType.textAnswer,
		}).Error(err)
		return errors.Wrap(err, "Failed to send message")
	} else {
		//
		msgToCRM := structures.MessageFromBot{
			Source:       string(structures.TelegramBotMember),
			ClientMsgID:  msg.ClientMsgID,
			UserLogin:    msg.UserLogin,
			ChatMsgID:    msg.ChatMsgID,
			MsgID:        msg.MsgID,
			Text:         answerType.textAnswer,
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
	}

	log.WithFields(logrus.Fields{
		"chat id":   msg.ChatMsgID,
		"answer":    answerType.textAnswer,
		"state":     msg.State,
		"orderUUID": msg.OrderUUID,
	}).Info("transition ok")
	return nil
}

//GetStateObjectFromText из текст возвращается объект типа proto.Constant
//это как бы сопоставление текстовых имен статусов в системе и объектов proto.Constant
func (h *Handler) GetStateObjectFromText(state string) proto.Constant {
	switch state {
	//
	case variables.OrderStates["OrderCreated"]:
		return proto.States.Taxi.Order.OrderCreated
	case variables.OrderStates["SmartDistribution"]:
		return proto.States.Taxi.Order.SmartDistribution
	case variables.OrderStates["Offered"]:
		return proto.States.Taxi.Order.OfferOffered
	case variables.OrderStates["DriverAccepted"]:
		return proto.States.Taxi.Order.DriverAccepted
	case variables.OrderStates["OrderCancelledState"]:
		return proto.States.Taxi.Order.OfferCancelled
	case variables.OrderStates["FindingDriver"]:
		return proto.States.Taxi.Order.FindingDriver
	case variables.OrderStates["Start"]:
		return proto.States.Taxi.Order.OrderStart
	case variables.OrderStates["OnTheWay"]:
		return proto.States.Taxi.Order.OnTheWay
	case variables.OrderStates["OnPlace"]:
		return proto.States.Taxi.Order.OnPlace
	case variables.OrderStates["Waiting"]:
		return proto.States.Taxi.Order.Waiting
	case variables.OrderStates["OrderPayment"]:
		return proto.States.Taxi.Order.OrderPayment
	case variables.OrderStates["Finished"]:
		return proto.States.Taxi.Order.Finished
	}
	return proto.States.Taxi.Order.Unknown
}
