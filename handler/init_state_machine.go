package handler

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/looplab/fsm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/tool"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"net/http"
)

const (
	noUserPhoneErrorCode = "noUserPhoneCode"
)

func (h *Handler) InitOrderStateFSM() *fsm.FSM {
	return fsm.NewFSM(
		proto.States.Welcome.S(),
		fsm.Events{
			{
				Name: proto.States.Welcome.S(),
				Dst:  proto.States.Welcome.S(),
			},
			//TAXI
			{
				Name: proto.States.Taxi.Order.CreateDraft.S(),
				Src: []string{
					proto.States.Welcome.S(),
					proto.States.Food.Order.Create.S(),
				},
				Dst: proto.States.Taxi.Order.CreateDraft.S(),
			},
			{
				Name: proto.States.Taxi.Order.Departure.S(),
				Src: []string{
					proto.States.Taxi.Order.CreateDraft.S(),
					proto.States.Taxi.Order.FixDeparture.S(),
				},
				Dst: proto.States.Taxi.Order.Departure.S(),
			},
			{
				Name: proto.States.Taxi.Order.FixDeparture.S(),
				Src: []string{
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.Departure.S(),
					proto.States.Taxi.Order.ChangeService.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.FixDeparture.S(),
			},
			{
				Name: proto.States.Taxi.Order.FixArrival.S(),
				Src: []string{
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.Departure.S(),
					proto.States.Taxi.Order.ChangeService.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.FixArrival.S(),
			},
			{
				Name: proto.States.Taxi.Order.Arrival.S(),
				Src: []string{
					proto.States.Taxi.Order.Departure.S(),
					proto.States.Taxi.Order.FixArrival.S(),
					proto.States.Taxi.Order.ChangeService.S(),
				},
				Dst: proto.States.Taxi.Order.Arrival.S(),
			},
			{
				Name: proto.States.Taxi.Order.PaymentMethod.S(),
				Src: []string{
					proto.States.Taxi.Order.Departure.S(),
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.ChangeService.S(),
				},
				Dst: proto.States.Taxi.Order.PaymentMethod.S(),
			},
			{
				Name: proto.States.Taxi.Order.PaymentByCash.S(),
				Src: []string{
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.PaymentByCash.S(),
			},
			{
				Name: proto.States.Taxi.Order.ChangeService.S(),
				Src: []string{
					proto.States.Taxi.Order.Departure.S(),
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.ChangeService.S(),
			},
			{
				Name: proto.States.Taxi.Order.NeedPhone.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.NeedPhone.S(),
			},
			{
				Name: proto.States.Taxi.Order.OrderCreated.S(),
				Src: []string{
					proto.States.Taxi.Order.PaymentByCash.S(),
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
					proto.States.Taxi.Order.ChangeService.S(),
					proto.States.Taxi.Order.NeedPhone.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.OrderCreated.S(),
			},
			//
			{
				Name: proto.States.Taxi.Order.FindingDriver.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OrderCreated.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.OfferCancelled.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
				},
				Dst: proto.States.Taxi.Order.FindingDriver.S(),
			},
			{
				Name: proto.States.Taxi.Order.SmartDistribution.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OrderCreated.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.OfferCancelled.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.Arrival.S(),
					proto.States.Taxi.Order.NeedPhone.S(),
					proto.States.Taxi.Order.PaymentMethod.S(),
					proto.States.Taxi.Order.ChangeService.S(),
				},
				Dst: proto.States.Taxi.Order.SmartDistribution.S(),
			},
			{
				Name: proto.States.Taxi.Order.OfferOffered.S(),
				Src: []string{
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
					proto.States.Taxi.Order.OfferCancelled.S(),
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OrderCreated.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
				},
				Dst: proto.States.Taxi.Order.OfferOffered.S(),
			},
			{
				Name: proto.States.Taxi.Order.DriverAccepted.S(),
				Src: []string{
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.OfferCancelled.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
				},
				Dst: proto.States.Taxi.Order.DriverAccepted.S(),
			},
			{
				Name: proto.States.Taxi.Order.OfferCancelled.S(),
				Src: []string{
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
				},
				Dst: proto.States.Taxi.Order.OfferCancelled.S(),
			},
			{
				Name: proto.States.Taxi.Order.OrderStart.S(),
				Src: []string{
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.DriverAccepted.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
				},
				Dst: proto.States.Taxi.Order.OrderStart.S(),
			},
			{
				Name: proto.States.Taxi.Order.DriverFounded.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.DriverAccepted.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.OfferCancelled.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
				},
				Dst: proto.States.Taxi.Order.OrderStart.S(),
			},
			{
				Name: proto.States.Taxi.Order.OnTheWay.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OfferOffered.S(),
					proto.States.Taxi.Order.Waiting.S(),
					proto.States.Taxi.Order.DriverAccepted.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.OnPlace.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
				},
				Dst: proto.States.Taxi.Order.OnTheWay.S(),
			},
			{
				Name: proto.States.Taxi.Order.OnPlace.S(),
				Src: []string{
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.OrderPayment.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
				},
				Dst: proto.States.Taxi.Order.OnPlace.S(),
			},
			{
				Name: proto.States.Taxi.Order.Waiting.S(),
				Src: []string{
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.OnPlace.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
				},
				Dst: proto.States.Taxi.Order.Waiting.S(),
			},
			{
				Name: proto.States.Taxi.Order.OrderPayment.S(),
				Src: []string{
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.Waiting.S(),
					proto.States.Taxi.Order.OnPlace.S(),
				},
				Dst: proto.States.Taxi.Order.OrderPayment.S(),
			},
			{
				Name: proto.States.Taxi.Order.Finished.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderPayment.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.OnPlace.S(),
					proto.States.Taxi.Order.Waiting.S(),
				},
				Dst: proto.States.Taxi.Order.Finished.S(),
			},
			//
			{
				Name: proto.States.Taxi.Order.Cancelled.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderPayment.S(),
					proto.States.Taxi.Order.Waiting.S(),
					proto.States.Taxi.Order.OnPlace.S(),
					proto.States.Taxi.Order.OnTheWay.S(),
					proto.States.Taxi.Order.DriverFounded.S(),
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.DriverAccepted.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
					proto.States.Taxi.Order.OrderCreated.S(),
				},
				Dst: proto.States.Taxi.Order.Cancelled.S(),
			},
			{
				Name: proto.States.Taxi.Order.DriverNotFound.S(),
				Src: []string{
					proto.States.Taxi.Order.OrderStart.S(),
					proto.States.Taxi.Order.DriverAccepted.S(),
					proto.States.Taxi.Order.SmartDistribution.S(),
					proto.States.Taxi.Order.FindingDriver.S(),
				},
				Dst: proto.States.Taxi.Order.DriverNotFound.S(),
			},
			//
			//FOOD
			{
				Name: proto.States.Food.Order.Create.S(),
				Src:  []string{proto.States.Welcome.S()},
				Dst:  proto.States.Food.Order.Create.S(),
			},
		},
		fsm.Callbacks{
			"before_" + proto.States.Taxi.Order.OrderCreated.S(): func(e *fsm.Event) {
				var err error
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}
				ctx := context.Background()

				//получаем пользователя
				user, err := h.DB.GetUser(ctx, msg.ClientMsgID, msg.Source)
				if err != nil {
					e.Cancel(errors.Wrap(err, "failed to get user from DB"))
				}
				// Номера нет, кидаем запрос на получение номера
				if user.Phone == "" {
					custError := fmt.Sprintf("$%s$ Just warning. User doesnt have phone", noUserPhoneErrorCode)
					e.Cancel(errors.New(custError))
					return
				}

				err = h.FillTariff(&msg.Order.OrderJSON)
				if err != nil {
					e.Cancel(errors.Wrap(err, "failed to fill tariff"))
				}

				msg.Order.OrderJSON.Client.MainPhone = "+" + user.Phone
				msg.Order.OrderJSON.CallbackPhone = "+" + user.Phone
				//msg.Order.OrderPrefs.TelegramOrderMsgID

				////сохраняем ID сообщения что бы потом его обновлять
				//update, ok := msg.Payload.(*tgbotapi.Update)
				//if ok {
				//	if update.Message != nil {
				//		//msg.Order.OrderPrefs.TelegramOrderMsgID = update.Message.MessageID
				//	} else if update.CallbackQuery != nil {
				//		//msg.Order.OrderPrefs.TelegramOrderMsgID = update.CallbackQuery.Message.MessageID
				//	}
				//}

				err = h.DB.SaveLocalOrder(ctx, &msg.Order)
				if err != nil {
					e.Cancel(errors.Wrap(err, "failed to save updated order"))
				}

				err = h.Pub.StartOrder(&msg.Order.OrderJSON)
				if err != nil {
					e.Cancel(errors.Wrap(err, "failed to start order"))
				}

				//добавляем заказ в буфер
				//h.Buffers.WIPOrdersFull[msg.OrderUUID] = msg.Order
				h.Buffers.WIPOrders[msg.OrderUUID] = msg.Order.State

			},
			"enter_" + proto.States.Taxi.Order.OrderCreated.S(): func(e *fsm.Event) {
				msg, err := validateInput(e)
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event":  "entring " + proto.States.Taxi.Order.OrderCreated.S() + " state",
						"reason": "failed to valide callback input data",
					}).Error(err)
				}
				val, ok := msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.Arrival.S()]
				if !ok {
					logs.Eloger.WithFields(logrus.Fields{
						"event": "getting arrival msg id",
					}).Error(errors.New("msg id for arrival order are empty"))
					return
				}
				//fmt.Printf("\n\nVALUE: %v\n\n", val)
				err = h.Telegram.UpdateKeyboard(msg.ChatMsgID, val, proto.GetCancelOrderButton())
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event": "clearing keyboard on entering state",
					}).Error(err)
				}
			},
			"before_" + proto.States.Taxi.Order.Departure.S(): func(e *fsm.Event) {
				var err error
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}

				//если полученный адрес это координаты
				if msg.Type == proto.MsgTypes.Coordinates.S() {
					var route structures.Route

					updateMsg, _ := msg.Payload.(*tgbotapi.Update)

					err = tool.SendRequest(
						http.MethodPost,
						h.Config.ClientURL+proto.System.CRMAdresses.FindAddress.S(),
						nil,
						structures.PureCoordinates{
							Lat:  updateMsg.Message.Location.Latitude,
							Long: updateMsg.Message.Location.Longitude,
						},
						&route)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to find address by coordinates"))
					}
					msg.Order.OrderJSON.Routes = append([]structures.Route{}, route)

					err = h.DB.SaveLocalOrder(context.Background(), &msg.Order)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to save local order after saving route"))
					}
				}

				ctx := context.Background()
				//если адрес из кнопки-колбека при исправлении адреса
				if msg.Type == proto.MsgTypes.TelegramCallback.S() && msg.Order.OrderPrefs.FixAddressVariant != 0 {
					if msg.Order.OrderPrefs.FixAddressVariant == 99 {
						return
					}
					address := msg.Order.OrderPrefs.DepartureVariants[msg.Order.OrderPrefs.FixAddressVariant-1]
					msg.Order, err = h.DB.SaveOrderRoute(ctx, msg.OrderUUID, proto.System.RouteTypes.Departure, address)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to save new order route on callback"))
					}
				}

				if msg.Type == proto.MsgTypes.TelegramMessage.S() {
					//получаем и пытаемся сохранить роут в заказ
					msg.Order, _, err = h.setOrderRoute(ctx, &msg.DFAnswer, proto.System.RouteTypes.Departure, msg.OrderUUID)
					if err != nil {
						e.Cancel(err)
					}
				}
			},
			"before_" + proto.States.Taxi.Order.Arrival.S(): func(e *fsm.Event) {
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}
				ctx := context.Background()

				//если адрес из кнопки-колбека при исправлении адреса
				if msg.Type == proto.MsgTypes.TelegramCallback.S() && msg.Order.OrderPrefs.FixAddressVariant != 0 {
					if msg.Order.OrderPrefs.FixAddressVariant == 99 {
						return
					}
					address := msg.Order.OrderPrefs.ArrivalVariants[msg.Order.OrderPrefs.FixAddressVariant-1]
					msg.Order, err = h.DB.SaveOrderRoute(ctx, msg.OrderUUID, proto.System.RouteTypes.Arrival, address)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to save new order route on callback"))
						return
					}
					//return
				}
				//получаем и пытаемся сохранить роут в заказ

				//если новый адрес введен ручками
				if msg.Type == proto.MsgTypes.TelegramMessage.S() && msg.DFAnswer.Intent != "" {
					msg.Order, _, err = h.setOrderRoute(ctx, &msg.DFAnswer, proto.System.RouteTypes.Arrival, msg.OrderUUID)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to change order route"))
						return
					}
				}

				if msg.Type == proto.MsgTypes.Coordinates.S() {

					var route structures.Route

					updateMsg, _ := msg.Payload.(*tgbotapi.Update)

					err = tool.SendRequest(
						http.MethodPost,
						h.Config.ClientURL+proto.System.CRMAdresses.FindAddress.S(),
						nil,
						structures.PureCoordinates{
							Lat:  updateMsg.Message.Location.Latitude,
							Long: updateMsg.Message.Location.Longitude,
						},
						&route)
					if err != nil {
						e.Cancel(errors.Wrap(err, "filed to find address by coordinates"))
						return
					}

					if len(msg.Order.OrderJSON.Routes) == 1 {
						msg.Order.OrderJSON.Routes = append(msg.Order.OrderJSON.Routes, route)
					} else if len(msg.Order.OrderJSON.Routes) == 2 {
						msg.Order.OrderJSON.Routes[1] = route
					}
				}

				//если сервис пустой, заполняем
				if msg.Order.OrderJSON.ServiceUUID == "" {
					msg.Order.OrderJSON.ServiceUUID = h.Config.Preferences.DefaultServiceUUID
					//если мы пришли со смены тарифа, то надо посмотреть заполнен ли поле выбранного тарифа и не равно ли уже выбранному
				} else if msg.Order.OrderPrefs.Service != "" && msg.Order.OrderPrefs.Service != msg.Order.OrderJSON.ServiceUUID {
					msg.Order.OrderJSON.ServiceUUID = msg.Order.OrderPrefs.Service
				}

				err = h.FillTariff(&msg.Order.OrderJSON)
				if err != nil {
					e.Cancel(errors.Wrap(err, "filed to fill tariff"))
					return
				}

				// записываем варианты тарифов
				if len(msg.Order.OrderJSON.Routes) >= 2 {
					var tariffButs []proto.TariffProto
					tariffs, err := h.GetTariffs(msg.Order.OrderJSON)
					if err != nil {
						logs.Eloger.WithFields(logrus.Fields{
							"event": "failed to get tarriff",
						}).Error(err)
					}
					for _, v := range tariffs {
						tariffButs = append(tariffButs, proto.TariffProto{
							ServiceUUID:     v.ServiceUUID,
							ServiceImage:    v.ServiceImage,
							Name:            v.Name,
							Currency:        v.Currency,
							BonusPayment:    v.BonusPayment,
							MaxBonusPayment: v.MaxBonusPayment,
							TotalPrice:      v.TotalPrice,
						})
					}
					msg.Order.OrderPrefs.Tariffs = tariffButs
				}

				//очищаем кнопку - "адрес подачи не верный"
				val, ok := msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.Departure.S()]
				if ok {
					err = h.Telegram.UpdateKeyboard(msg.ChatMsgID, val, proto.ButtonsSet{})
					if err != nil {
						logs.Eloger.WithFields(logrus.Fields{
							"event": "clearing keyboard on entering state",
						}).Error(err)
					}
					delete(msg.Order.OrderPrefs.MsgsIDs, proto.States.Taxi.Order.Departure.S())
				}

				err = h.DB.SaveLocalOrder(ctx, &msg.Order)
				if err != nil {
					e.Cancel(errors.Wrap(err, "filed to save local order"))
				}
			},

			"before_" + proto.States.Taxi.Order.Cancelled.S(): func(e *fsm.Event) {
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}

				// отправить action на отмену заказа в кролик
				err = h.Pub.ActionOnOrder(&structures.ActionOnOrder{
					OrderUUID: msg.OrderUUID,
					Action:    structures.ActionOnOrderCancelOrder,
				})
				if err != nil {
					e.Cancel(errors.Wrap(err, "failed to send cancel command"))
				}

			},
			"before_" + proto.States.Taxi.Order.FixDeparture.S(): func(e *fsm.Event) {
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}

				routes, err := h.GetCRMAdresses(msg.Order.OrderJSON.Routes[0].UnrestrictedValue)
				if err != nil {
					e.Cancel(errors.Wrap(err, "cant find adresses for departure"))
				}
				if len(routes) >= 5 {
					routes = routes[:5]
				}

				msg.Order.OrderPrefs.DepartureVariants = routes

				err = h.DB.SaveLocalOrder(context.Background(), &msg.Order)
				if err != nil {
					e.Cancel(err)
				}
			},
			"before_" + proto.States.Taxi.Order.FixArrival.S(): func(e *fsm.Event) {
				msg, err := validateInput(e)
				if err != nil {
					e.Cancel(err)
				}

				if len(msg.Order.OrderJSON.Routes) < 2 {
					e.Cancel(errors.Wrap(err, "less then 2 routes"))
				}
				routes, err := h.GetCRMAdresses(msg.Order.OrderJSON.Routes[1].UnrestrictedValue)
				if err != nil {
					e.Cancel(errors.Wrap(err, "cant find adresses for departure"))
				}
				if len(routes) >= 5 {
					routes = routes[:5]
				}

				msg.Order.OrderPrefs.ArrivalVariants = routes

				err = h.DB.SaveLocalOrder(context.Background(), &msg.Order)
				if err != nil {
					e.Cancel(err)
				}
			},
			//
			"leave_" + proto.States.Taxi.Order.FixArrival.S(): func(e *fsm.Event) {
				var err error
				msg, err := validateInput(e)
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event": "validating input on enter state",
					}).Error(err)
					return
				}
				if msg.Type == proto.MsgTypes.TelegramMessage.S() {
					val, ok := msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.FixArrival.S()]
					if !ok {
						return
					}
					err = h.Telegram.UpdateKeyboard(msg.ChatMsgID, val, proto.ButtonsSet{})
					if err != nil {
						logs.Eloger.WithFields(logrus.Fields{
							"event":  "deleting FixDeparture keyboard",
							"msgID":  msg.MsgID,
							"chatID": msg.ChatMsgID,
						}).Error(err)
					}
				}
			},
			"leave_" + proto.States.Taxi.Order.FixDeparture.S(): func(e *fsm.Event) {
				var err error
				msg, err := validateInput(e)
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event": "validating input on enter state",
					}).Error(err)
					return
				}
				if msg.Type == proto.MsgTypes.TelegramCallback.S() {
					err = h.Telegram.DeleteMsg(msg.ChatMsgID, msg.MsgID)
					if err != nil {
						logs.Eloger.WithFields(logrus.Fields{
							"event":  "deleting msg",
							"msgID":  msg.MsgID,
							"chatID": msg.ChatMsgID,
						}).Error(err)
					}
				}
			},
			//
			"enter_state": func(e *fsm.Event) {
				var err error
				msg, err := validateInput(e)
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event": "validating input on enter state",
					}).Error(err)
					return
				}
				state := structures.OfferStates{
					OrderUUID: msg.OrderUUID,
					State:     e.Dst,
				}

				msg.Order, err = h.DB.SetOrderState(context.Background(), state)
				if err != nil {
					logs.Eloger.WithFields(logrus.Fields{
						"event":  "setting order state",
						"reason": "db error",
					}).Error(err)
					return
				}
				msg.State = msg.Order.State

				//удалеям из буфера действующиз заказов
				if variables.InactiveOrderStates(msg.State) {
					delete(h.Buffers.WIPOrders, msg.OrderUUID)
				} else {
					h.Buffers.WIPOrders[msg.OrderUUID] = msg.Order.State
				}
			},
		},
	)
}
