package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

const (
	ActionOrderStart = "start_order"
	//Типы клавиатур
	ContactRequest = "request_contact_keyboard"
)

func (h *Handler) HandleNewTelegramCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	var (
		err error
		//response string
	)
	log := logs.Eloger.WithFields(logrus.Fields{
		"event":  "handling new callback",
		"userID": callback.From.ID,
	})
	log.Debug("Handling Callback")

	chatMsg := structures.MessageFromBot{
		Source:       "telegram",
		UserLogin:    callback.From.UserName,
		Text:         fmt.Sprintf("callback:%s", callback.Data),
		CreatedAt:    time.Now(),
		ChatMsgID:    callback.Message.Chat.ID,
		ClientMsgID:  strconv.Itoa(callback.From.ID),
		CreatedAtMsg: time.Unix(int64(callback.Message.Date), 0),
	}

	currentOrder, err := h.DB.GetCurrentOrder(ctx, strconv.Itoa(callback.From.ID), "telegram")
	if currentOrder.OrderUUID == "" {
		// проверка последнего заказа на статус order_created
		lastOrder, err := h.DB.GetLastOrder(ctx, strconv.Itoa(callback.From.ID), "telegram")
		if err != nil {
			log.Error(errpath.Err(err))
		}
		if lastOrder.State == variables.OrderStates["OrderCreated"] || lastOrder.State == variables.OrderStates["Start"] {
			currentOrder = lastOrder
		} else {
			log.WithField("reason", "can't get order").Error(errpath.Errorf("curent order is empty"))
			_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, ErrorOccurs)
			if err != nil {
				log.Error(errpath.Err(err))
			}
			return
		}
	}

	chatMsg.OrderUUID = currentOrder.OrderUUID
	err = h.Pub.NewMsg(&chatMsg)
	if err != nil {
		log.Error(err)
	}

	log.WithField("orderUUID", chatMsg.OrderUUID).Debug("Local Order for Callback")

	action, btnValue := ButtonDataGetValues(callback.Data)

	switch action {

	case string(proto.Consts.BotSend.ButtonsActions.PaymentChoice):
		//присваиваем код услуги
		serviceuuid := btnValue
		currentOrder.OrderJSON.ServiceUUID = serviceuuid
		err = h.DB.SaveLocalOrder(ctx, &currentOrder)
		if err != nil {
			log.WithField("reason", "can't update order").Error(errpath.Err(err))
		}

		stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.PaymentChoice)})
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		currentOrder.State = stOrder.State

		_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.PaymentChoice), getChoicePaymentTypeButtons())
		if err != nil {
			log.Errorln(errpath.Err(err))
		}

	case string(proto.Consts.BotSend.ButtonsActions.OrderStart):

		//теперь проверяем пользователя
		user, err := h.DB.GetUser(ctx, chatMsg.ClientMsgID, chatMsg.Source)
		if err != nil {
			log.WithField("reason", "can't get user from DB").Error(errpath.Err(err))
			h.Telegram.SendMessage(callback.Message.Chat.ID, ErrorOccurs)
			return
		}
		// Номера нет, кидаем запрос на получение номера
		if user.Phone == "" {
			// currentOrder.State = variables.OrderStates["ChatStartOrder"] ????
			h.DB.SaveLocalOrder(ctx, &currentOrder)
			_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, EnterYourPhone, getContactButton())
			if err != nil {
				log.WithField("reason", "can't request contact keyboard").Error(errpath.Err(err))
			}
			return
		}

		err = h.FillTariff(&currentOrder.OrderJSON)
		if err != nil {
			log.WithField("reason", "can't calculate order tariff").Error(errpath.Err(err))
			h.Telegram.SendMessage(callback.Message.Chat.ID, ErrorOccurs)
			return
		}

		currentOrder.OrderJSON.Client.MainPhone = "+" + user.Phone
		currentOrder.OrderJSON.CallbackPhone = "+" + user.Phone

		//fmt.Printf("Service: %+v\n", currentOrder.OrderJSON.Service)
		err = h.DB.SaveLocalOrder(ctx, &currentOrder)
		if err != nil {
			log.WithField("reason", "can't update order").Error(errpath.Err(err))
		}
		log.WithField("reason", "orderd datat updated").Debug("SUCCES")

		log.WithField("OrderUUID", chatMsg.OrderUUID).Info("STARTING ORDER!")

		err = h.Pub.StartOrder(&currentOrder.OrderJSON)
		if err != nil {
			log.WithField("reason", "can't create order").Error(errpath.Err(err))
			h.Telegram.SendMessage(callback.Message.Chat.ID, ErrorOccurs)
			return
		}
		//добавляем заказ в буфер
		h.Buffers.WIPOrders[chatMsg.OrderUUID] = currentOrder.State
		_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.OrderCreated), getCancelOrderButton())
		if err != nil {
			log.Errorln(errpath.Err(err))
		}

	case string(proto.Consts.BotSend.ButtonsActions.AnotherDepartureAdress), string(proto.Consts.BotSend.ButtonsActions.AnotherArrivalAdress):

		// получаем адресв из автокомплита
		var find string
		// откат статуса назад
		switch currentOrder.State {
		case string(proto.Consts.Order.CreationStates.SetArrivalAddress):
			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetDepartureAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			if len(currentOrder.OrderJSON.Routes) < 1 {
				log.Errorln(errpath.Errorf("ошибка при попытке смены начального адреса. начальный адрес пустой"))
			}
			find = currentOrder.OrderJSON.Routes[0].UnrestrictedValue

		case string(proto.Consts.Order.CreationStates.ServiceChoice):
			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetArrivalAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			if len(currentOrder.OrderJSON.Routes) < 2 {
				log.Errorln(errpath.Errorf("ошибка при попытке смены конечного адреса. конечный адрес пустой"))
			}
			find = currentOrder.OrderJSON.Routes[1].UnrestrictedValue

		default:
			log.Warnln(errpath.Errorf("RewriteAddress case not found"))
		}

		// if currentOrder.State == string(proto.Consts.Order.CreationStates.SetArrivalAddress) {
		// 	if len(currentOrder.OrderJSON.Routes) < 1 {
		// 		log.Errorln(errpath.Errorf("ошибка при попытке смены начального адреса. начальный адрес пустой"))
		// 	}
		// 	find = currentOrder.OrderJSON.Routes[0].UnrestrictedValue
		// }
		// if currentOrder.State == string(proto.Consts.Order.CreationStates.ServiceChoice) {
		// 	if len(currentOrder.OrderJSON.Routes) < 2 {
		// 		log.Errorln(errpath.Errorf("ошибка при попытке смены конечного адреса. конечный адрес пустой"))
		// 	}
		// 	find = currentOrder.OrderJSON.Routes[1].UnrestrictedValue
		// }

		if find == "" {
			log.Warnln(errpath.Errorf("adr for find is empty"))
		}

		routes, err := h.GetCRMAdresses(find)
		if err != nil {
			log.Errorln(errpath.Err(err))
			return
		}
		var countChoiceAdrBtns int = 5
		if len(routes) >= countChoiceAdrBtns {
			routes = routes[:countChoiceAdrBtns]
		}

		// сообщение с кнопками выбора предложенного адреса
		msg := fmt.Sprintln(proto.Consts.BotSend.Answers.AdressChoices + "\n")
		for i, route := range routes {
			msg += fmt.Sprint(i+1, ". ", route.Value, "\n")
		}

		var setAction string
		if action == string(proto.Consts.BotSend.ButtonsActions.AnotherDepartureAdress) {
			setAction = string(proto.Consts.BotSend.ButtonsActions.SetDepartureAdress)
		}
		if action == string(proto.Consts.BotSend.ButtonsActions.AnotherArrivalAdress) {
			setAction = string(proto.Consts.BotSend.ButtonsActions.SetArrivalAdress)
		}
		if setAction == "" {
			log.Warnln(errpath.Errorf("пустой action при смене адреса"))
		}

		_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, msg, getChoiceAdressButtons(routes, setAction))
		if err != nil {
			log.Errorln(errpath.Err(err))
			return
		}

	case string(proto.Consts.BotSend.ButtonsActions.SetDepartureAdress), string(proto.Consts.BotSend.ButtonsActions.SetArrivalAdress):
		routes, err := h.GetCRMAdresses(btnValue)
		if err != nil {
			log.Error(errpath.Err(err))
			break
		}
		if len(routes) == 0 {
			log.Error(errpath.Errorf("routes list from CRM is empty"))
			break
		}

		var routeNumber proto.Constant
		if action == string(proto.Consts.BotSend.ButtonsActions.SetDepartureAdress) {
			routeNumber = proto.Consts.BotSend.ButtonsActions.SetDepartureAdress

			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetArrivalAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, fmt.Sprintf(string(proto.Consts.BotSend.Answers.DepartureAddress), routes[0].UnrestrictedValue), getAnotherDepartureAddressButtons(routes[0].UnrestrictedValue))
			if err != nil {
				log.Error(errpath.Err(err))
				break
			}

		}
		if action == string(proto.Consts.BotSend.ButtonsActions.SetArrivalAdress) {
			routeNumber = proto.Consts.BotSend.ButtonsActions.SetArrivalAdress
			var tariffs []models.ShortTariff
			var buildmsg string

			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.ServiceChoice)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			if len(currentOrder.OrderJSON.Routes) >= 2 {
				tariffs, err = h.GetTariffs(currentOrder.OrderJSON)
				if err != nil {
					err = errpath.Err(err, "Error getting tariff")
					log.Errorln(err)
					// errResponse = err.Error()
					break
				}
				buildmsg = fmt.Sprintf(string(proto.Consts.BotSend.Answers.ArrivalAddress), currentOrder.OrderJSON.Routes[0].UnrestrictedValue, currentOrder.OrderJSON.Routes[1].UnrestrictedValue)

				if buildmsg == "" {
					log.Warnln(errpath.Err(err, "отправленное сообщение пустое"))
				}
			}

			_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, buildmsg, getTariffButtons(tariffs, currentOrder.OrderJSON.Routes[1].UnrestrictedValue))
			if err != nil {
				log.Error(errpath.Err(err))
				break
			}
		}
		if routeNumber != "" {
			_, err = h.DB.UpdateOrderRoute(ctx, currentOrder.OrderUUID, routeNumber, routes[0])
			if err != nil {
				log.Error(errpath.Err(err))
				break
			}

		}

	case string(proto.Consts.BotSend.ButtonsActions.RewriteAddress):
		// откат статуса назад
		switch currentOrder.State {
		case string(proto.Consts.Order.CreationStates.SetArrivalAddress):
			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetDepartureAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.RewriteAddress))

		case string(proto.Consts.Order.CreationStates.ServiceChoice):
			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetArrivalAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

			h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.RewriteAddress))
		default:
			log.Warnln(errpath.Errorf("RewriteAddress case not found"))
		}

	// оценка поездки клиентом
	case string(proto.Consts.BotSend.ButtonsActions.SetRating):
		r, err := strconv.Atoi(btnValue)
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		currentOrder.OrderJSON.ClientRating.Value = r

		// TODO: отправить в кролик ордер с обновленным рейтигом, но не сегодняяя!!!

	case string(proto.Consts.BotSend.ButtonsActions.CreatingWithOperator):
		// перевести ордер в статус создания оператором
		stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.ProcessingWithOperator)})
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		currentOrder.State = stOrder.State

		// отправить в кролик ордер с обновленным статусом т.к. обновленный ордер из црм придет со старым статусом (проверка стоит на CreatingWithOperator)
		// вроде не актуально ^ тк статус меняется в момент принятия сообщения от оператора

		// уведомить оператора дабы обработать заказ
		action := structures.ActionOnOrder{OrderUUID: currentOrder.OrderUUID, Action: structures.ActionOnOrderSetOrderImportant}
		err = h.Pub.ActionOnOrder(&action)
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		//
		_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.WaitingForOperator))
		if err != nil {
			log.Error(errpath.Err(err))
			break
		}

	case string(proto.Consts.BotSend.ButtonsActions.CancelOrder):
		// установка статуса - заказ отменен // возможно статус обновиться через кролика // в срмке статус обновляется напрямую без кролика
		stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: variables.OrderStates["OrderCancelledState"]})
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		currentOrder.State = stOrder.State
		// отправить action на отмену заказа в кролик
		err = h.Pub.ActionOnOrder(&structures.ActionOnOrder{OrderUUID: currentOrder.OrderUUID, Action: structures.ActionOnOrderCancelOrder})
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		// сообщение клиенту об отмене заказа
		_, err = h.Telegram.SendMessage(callback.Message.Chat.ID, string(proto.Consts.BotSend.Answers.ForStates.OrderCanceled))
		if err != nil {
			log.Error(errpath.Err(err))
			break
		}

	default:
	}

}

// --- buttonsSets

func getContactButton() proto.ButtonsSet {
	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Reply,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Taxi.SendContactButton),
						Type: proto.Consts.ButtonsTypes.Contact,
					},
				},
			},
		},
	}
}

func getChoiceAdressButtons(routes []structures.Route, setActions string) proto.ButtonsSet {
	var adressBtns []proto.MsgButton
	for i, route := range routes {
		adressBtns = append(adressBtns, proto.MsgButton{
			Text: fmt.Sprintln(i + 1),
			Data: cutButtonData(ButtonDataAddValues("", setActions, route.UnrestrictedValue)),
		})
	}

	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: adressBtns,
			},
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: "ввести адресс заного",
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.RewriteAddress)),
					},
				},
			},
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: "позвать на помощь человека",
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.CreatingWithOperator)),
					},
				},
			},
		},
	}
}

func getChoicePaymentTypeButtons() proto.ButtonsSet {
	// список кредитных карт
	// var adressBtns []proto.MsgButton
	// for i, route := range routes {
	// 	adressBtns = append(adressBtns, proto.MsgButton{
	// 		Text: fmt.Sprintln(i + 1),
	// 		Data: cutButtonData(ButtonDataAddValues("", setActions, route.UnrestrictedValue)),
	// 	})
	// }

	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: "Вызвать такси 🏁 ",
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.OrderStart)),
					},
				},
			},
			// {
			// 	MsgButtons: adressBtns,
			// },
			//{
			//	MsgButtons: []proto.MsgButton{
			//		{
			//			Text: "другой картой [#]",
			//			Data: "[#]", // ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.CreatingWithOperator)),
			//		},
			//	},
			//},
		},
	}
}

func getCancelOrderButton() proto.ButtonsSet {
	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: "Отменить заказ",
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.CancelOrder)),
					},
				},
			},
		},
	}
}
