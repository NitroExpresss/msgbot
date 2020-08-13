//Создание нового заказа и работа с регистрацией клиентов
package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

func (h *Handler) InitBuffer(ctx context.Context) error {

	activeOrders, err := h.DB.GetActiveOrders(ctx)
	if err != nil {
		return err
	}
	if i := len(activeOrders); i == 0 {
		return nil
	}
	for _, v := range activeOrders {
		h.Buffers.WIPOrders[v.OrderUUID] = variables.TranslateOrderStateForClient(v.State, false)
	}
	return nil
}

func (h *Handler) UpdateOrderState(ctx context.Context, newState structures.OfferStates) {
	log := logs.Eloger.WithFields(logrus.Fields{
		"event":     "handling new callback",
		"orderUUID": newState.OrderUUID,
		"state":     newState.State,
	})
	if _, ok := h.Buffers.WIPOrders[newState.OrderUUID]; !ok {
		//заказа нет в буфере - ливаем
		return
	}

	order, err := h.DB.SetOrderState(ctx, newState)
	if err != nil {
		log.WithField("reason", "cant update state in DB").Error(err)
	}
	log.Debug("State Updated")

	translatedState := variables.TranslateOrderStateForClient(newState.State, false)

	if translatedState == h.Buffers.WIPOrders[newState.OrderUUID] {
		//Статус такой же как и был - ливаем
		return
	}
	h.Buffers.WIPOrders[newState.OrderUUID] = translatedState

	//
	chatID, err := strconv.ParseInt(order.ChatMsgId, 10, 64)
	if err != nil {
		log.WithField("reason", "cant convert chatID to int").Error(err)
	}

	switch newState.State {
	case variables.OrderStates["OnPlace"]:
		wb, ok := order.OrderJSON.Tariff.WaitingBoarding[0]
		if !ok {
			log.WithField("reason", "WaitingBoarding[0] empty").Warnln("order.OrderJSON.Tariff.WaitingBoarding[0] not found")
			wb = 0
		}
		resp := fmt.Sprintf(string(proto.Consts.BotSend.Answers.ForStates.OnPlace), int(wb))
		_, err = h.Telegram.SendMessage(chatID, resp)
		if err != nil {
			log.WithField("reason", "cant send msg to telegram").Error(err)
		}
	case variables.OrderStates["OnTheWay"]:
		resp := fmt.Sprintf(string(proto.Consts.BotSend.Answers.ForStates.OnTheWay), order.OrderJSON.Routes[1].Value)
		_, err = h.Telegram.SendMessage(chatID, resp)
		if err != nil {
			log.WithField("reason", "cant send msg to telegram").Error(err)
		}
	case variables.OrderStates["OrderPayment"]:
		resp := fmt.Sprintf(string(proto.Consts.BotSend.Answers.ForStates.OrderPayment), order.OrderJSON.Tariff.TotalPrice)
		_, err = h.Telegram.SendMessage(chatID, resp)
		if err != nil {
			log.WithField("reason", "cant send msg to telegram").Error(err)
		}
	case variables.OrderStates["Finished"]:
		//err = h.Telegram.SendMessage(chatID, string(proto.Consts.BotSend.Answers.ForStates.Finished), getRaitingButtons())
		//if err != nil {
		//	log.WithField("reason", "cant send msg to telegram").Error(err)
		//}
	default:
	}
}

func getRaitingButtons() proto.ButtonsSet {
	var btnsCount int = 5
	var btns []proto.MsgKeyboardRows

	for i := btnsCount; i > 0; i-- {
		var btnText string
		for j := 0; j < i; j++ {
			btnText += string(proto.Consts.BotSend.Answers.Symbol.Star) + " "
		}

		btns = append(btns, proto.MsgKeyboardRows{
			MsgButtons: []proto.MsgButton{
				{
					Text: btnText,
					Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.SetRating), fmt.Sprint(i)),
				},
			},
		})
	}

	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons:         btns,
	}
}
