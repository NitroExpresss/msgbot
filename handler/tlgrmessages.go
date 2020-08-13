package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/pkg/structures/tool"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/dialogflow"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

const (
	//Интенты
	DepartureIntent = "DepartureAddressIntent"
	ArrivalIntent   = "ArrivalAddressIntent"
	//Параметры entities
	//Откуда
	DepartAddressHouse    = "departure-address-house"    // номер дома
	DepartAddressBuilding = "departure-address-building" // номер корпуса
	DepartAddressStreet   = "departure-address-street"   // название улицы
	DepartAddressLetter   = "departure-address-letter"   // название литера
	DepartPublicPlace     = "departure-public-place"     //название публичного места
	DepartCity            = "departure-city"             // название города

	//Куда
	ArrivalAddressHouse    = "arrival-address-house"    // номер дома
	ArrivalAddressBuilding = "arrival-address-building" // номер корпуса
	ArrivalAddressStreet   = "arrival-address-street"   // название улицы
	ArrivalAddressLetter   = "arrival-address-letter"   // название литера
	ArrivalPublicPlace     = "arrival-public-place"     // название публичного места
	ArrivalCity            = "arrival-city"             // название города

	//Фразы - ответы
	//TODO перенести эти фразы в DialogFlow, и вызывать определенный Intent ошибки
	AddressNotFound        = "Адреса: %s, нет у меня в базе. Пожалуйста, попробуйте другой адресс"
	ErrorOccurs            = "Произошка какая-то ошибка, мне очень жаль 😢"
	EnterYourPhone         = "Мне нужен ваш номер телефона, что бы оформить заказ 😝"
	SendContactButton      = "Поделиться моим контактом"
	ContactDataSaved       = "Отлично, номерок записан..."
	OrderCreated           = "Заказ создан. Начинаю искать свободную машину в вашем районе"
	OrderStateUpdatedMsg   = "Статус вашего заказа изменен.\n\nТекущий статус: %s"
	CarFoundedMsg          = "Ура, мы нашли для вас машину!\n\n %s \nПожалуйста, ожидайте..."
	WillArrivePhraseNoTime = "%s %s номер %s"
	WillArrivePhrase       = "Через %s к вам подъедет\n%s цвет %s номер %s"
)

func (h *Handler) HandleNewTelegramMsg(ctx context.Context, msg *tgbotapi.Message) {
	var (
		err         error
		address     string
		errResponse string
		tariffs     []models.ShortTariff
	)

	log := logs.Eloger.WithFields(logrus.Fields{
		"event":  "handling new message",
		"userID": msg.From.ID,
	})
	log.Debug("Handling New Message")

	// если получаем координаты, находим адрес и записываем его в первый роут
	if msg.Text == "" && msg.Location != nil {
		var route structures.Route
		err = tool.SendRequest(
			http.MethodPost,
			h.Config.ClientURL+string(proto.Consts.EndPoints.Client.FindAdress),
			nil,
			structures.PureCoordinates{
				Lat:  msg.Location.Latitude,
				Long: msg.Location.Longitude,
			},
			&route)
		if err != nil {
			log.Errorln(errpath.Err(err))
			return
		}
		msg.Text = route.UnrestrictedValue
	}

	//если получаем контакт, записываем его в бэдэшечку
	if msg.Contact != nil {
		getMyContact := proto.MessangerContact{
			PhoneNumber: msg.Contact.PhoneNumber,
			FirstName:   msg.Contact.FirstName,
			LastName:    msg.Contact.LastName,
			UserID:      msg.Contact.UserID,
		}
		err = h.DB.SaveUserContact(ctx, getMyContact, "telegram")
		if err != nil {
			log.WithField("reason", "saving contact datat").Error(errpath.Err(err))
		}
		_, err = h.Telegram.SendMessage(msg.Chat.ID, string(proto.Consts.BotSend.Answers.ContactDataSaved))
		if err != nil {
			log.WithField("reason", "error sending contact saving confirm message").Error(errpath.Err(err))
		}
		msg.Text = string(proto.Consts.BotSend.Answers.ContactDataSaved)
		// return
		//TODO: првоерять есть ли заказ, и запускать если он ожидает...
	}

	chatMsg := structures.MessageFromBot{
		Source:       "telegram",
		UserLogin:    msg.From.UserName,
		Text:         msg.Text,
		CreatedAt:    time.Now(),
		ChatMsgID:    msg.Chat.ID,
		ClientMsgID:  strconv.Itoa(msg.From.ID),
		CreatedAtMsg: time.Unix(int64(msg.Date), 0),
	}

	currentOrder, err := h.GetMsgOrder(ctx, chatMsg)
	if err != nil {
		log.WithField("reason", "can't get order uuid").Error(errpath.Err(err))
		return
	}
	if currentOrder.OrderUUID == "" {
		lastOrder, err := h.DB.GetLastOrder(ctx, chatMsg.ClientMsgID, chatMsg.Source)
		if err != nil {
			log.WithField("reason", "can't get last order uuid").Error(errpath.Err(err))
			return
		}
		if lastOrder.OrderUUID != "" {
			if lastOrder.State != variables.OrderStates["Finished"] && lastOrder.State != variables.OrderStates["OrderCancelledState"] {
				chatMsg.OrderUUID = lastOrder.OrderUUID
				err = h.Pub.NewMsg(&chatMsg)
				if err != nil {
					log.Errorln(errpath.Err(err))
					return
				}
				err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, string(proto.Consts.BotSend.Answers.WaitForCompletion))
				if err != nil {
					log.Errorln(errpath.Err(err))
				}
			}
			return
		}

		_, err = h.Telegram.SendMessage(msg.Chat.ID, string(proto.Consts.BotSend.Answers.WaitForCompletion))
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
		return
	}
	chatMsg.OrderUUID = currentOrder.OrderUUID
	log.WithField("orderUUID", chatMsg.OrderUUID).Debug("Local Order Founded")

	err = h.Pub.NewMsg(&chatMsg)
	if err != nil {
		log.Errorln(errpath.Err(err))
		return
	}

	intentAnswer, err := h.DF.DetectIntentText(msg.Text, string(msg.From.ID)) // DetectIntentText(msg.Text, chatMsg.OrderUUID)
	if err != nil {
		// log.Errorln(errpath.Err(err))
		// return
		log.Warnln(errpath.Err(err))
	}
	if chatMsg.Text == string(proto.Consts.Intents.Skip) {
		intentAnswer.Intent = string(proto.Consts.Intents.Skip)
		intentAnswer.Answer = string(proto.Consts.Intents.Skip)
	}

	// если не идет переписка с оператором
	if currentOrder.State != string(proto.Consts.Order.CreationStates.ProcessingWithOperator) { // && currentOrder.State != string(proto.Consts.Order.CreationStates.ServiceChoice) // оператор нажал обновить

		switch intentAnswer.Intent {
		case string(proto.Consts.Intents.Skip):
			break
		case string(proto.Consts.Intents.DefaultFallbackIntent):
			if intentAnswer.Intent == string(proto.Consts.Intents.DefaultFallbackIntent) {
				err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, intentAnswer.Answer)
				if err != nil {
					log.Errorln(errpath.Err(err))
				}
				return
			}
		case string(proto.Consts.Intents.Welcome):
			stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.StartChatting)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = stOrder.State

		case string(proto.Consts.Intents.TaxiCallIntent):
			upOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetDepartureAddress)})
			if err != nil {
				log.Errorln(errpath.Err(err))
			}
			currentOrder.State = upOrder.State

		// 	//Разбираем Intent откуда и куда
		// case DepartureIntent, ArrivalIntent:
		case string(proto.Consts.Intents.AddressIntent):
			// if currentOrder.State != string(proto.Consts.Order.CreationStates.SetArrivalAddress) && currentOrder.State != string(proto.Consts.Order.CreationStates.SetDepartureAddress) {
			// 	break
			// }

			if currentOrder.State == string(proto.Consts.Order.CreationStates.StartChatting) {
				upOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetDepartureAddress)})
				if err != nil {
					log.Errorln(errpath.Err(err))
				}
				currentOrder.State = upOrder.State
			}

			if currentOrder.State == string(proto.Consts.Order.CreationStates.SetArrivalAddress) {
				order, adr, err := h.setOrderRoute(ctx, &intentAnswer, proto.Consts.Order.SetRoute.Arrival, currentOrder.OrderUUID)
				if err != nil {
					log.Errorln(errpath.Err(err))
					break
				}
				currentOrder.OrderJSON.Routes = order.OrderJSON.Routes
				address = adr

				//Пересчитываем тариф
				if len(order.OrderJSON.Routes) >= 2 {

					upOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.ServiceChoice)})
					if err != nil {
						log.Errorln(errpath.Err(err))
						break
					}
					currentOrder.State = upOrder.State

				}
				break
			}
			//
			if currentOrder.State == string(proto.Consts.Order.CreationStates.SetDepartureAddress) {
				order, adr, err := h.setOrderRoute(ctx, &intentAnswer, proto.Consts.Order.SetRoute.Departure, currentOrder.OrderUUID)
				if err != nil {
					log.Errorln(errpath.Err(err))
				}
				currentOrder.OrderJSON.Routes = order.OrderJSON.Routes
				address = adr

				stOrder, err := h.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: currentOrder.OrderUUID, State: string(proto.Consts.Order.CreationStates.SetArrivalAddress)})
				if err != nil {
					log.Errorln(errpath.Err(err))
				}
				currentOrder.State = stOrder.State
			}

		default:

		}
	}

	switch currentOrder.State {
	// case string(proto.Consts.Intents.DefaultFallbackIntent):

	case string(proto.Consts.Order.CreationStates.StartChatting):
		err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, intentAnswer.Answer, getWelcomeButtons())
		if err != nil {
			log.Errorln(errpath.Err(err))
		}

	case string(proto.Consts.Order.CreationStates.SetDepartureAddress):
		err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, intentAnswer.Answer, getSendLocationButtons())
		if err != nil {
			log.Errorln(errpath.Err(err))
		}

	case string(proto.Consts.Order.CreationStates.SetArrivalAddress):
		// сообщение дабы убрать кнопку локации оставшуюся от статуса departure
		err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, string(proto.Consts.BotSend.Answers.Received))
		if err != nil {
			err = errpath.Err(err, "сообщение не отправленно")
			log.Errorln(err)
			errResponse = err.Error()
			break
		}
		err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, fmt.Sprintf(string(proto.Consts.BotSend.Answers.DepartureAddress), address), getAnotherDepartureAddressButtons(address))
		if err != nil {
			err = errpath.Err(err, "сообщение не отправленно")
			log.Errorln(err)
			errResponse = err.Error()
			break
		}

	case string(proto.Consts.Order.CreationStates.ServiceChoice):
		//Пересчитываем тариф
		if len(currentOrder.OrderJSON.Routes) >= 2 {
			tariffs, err = h.GetTariffs(currentOrder.OrderJSON)
			if err != nil {
				err = errpath.Err(err, "Error getting tariff")
				log.Errorln(err)
				errResponse = err.Error()
				break
			}
			buildmsg := fmt.Sprintf(string(proto.Consts.BotSend.Answers.ArrivalAddress), currentOrder.OrderJSON.Routes[0].UnrestrictedValue, currentOrder.OrderJSON.Routes[1].UnrestrictedValue)

			if buildmsg == "" {
				log.Warnln(errpath.Err(err, "отправленное сообщение пустое"))
			}

			err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, buildmsg, getTariffButtons(tariffs, address))
			if err != nil {
				err = errpath.Err(err, "сообщение не отправленно")
				log.Errorln(err)
				errResponse = err.Error()
				break
			}
		}

	default:
		// tlgMsgSender.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		// h.SendMessageToClientAndCRM(&chatMsg,msg.Chat.ID, intentAnswer.Answer)
	}

	if errResponse != "" {
		log.Warnln(errpath.Errorf(errResponse))
		err = h.SendMessageToClientAndCRM(&chatMsg, msg.Chat.ID, errResponse)
		if err != nil {
			log.Errorln(errpath.Err(err))
		}
	}
	return
}

//setOrderRoute получает данные из интента по адресу и возвращает заказ, имя роута и ошибку
//также функци сохраняет полученный роут в тело самого заказа
func (h *Handler) setOrderRoute(ctx context.Context, intentAnswer *dialogflow.NLPResponse, routeType proto.Constant, orderUUID string) (models.LocalOrders, string, error) {
	var order models.LocalOrders
	var route string

	if intentAnswer.Intent == "" {
		return order, route, errors.New("Intent are empty")
	}
	//берет адрес из интента
	adress, err := addressFromIntet(*intentAnswer)
	if err != nil {
		return order, route, errpath.Err(err)
	}

	//ищем роут
	routes, err := h.GetCRMAdresses(adress)
	if err != nil {
		return order, route, errpath.Err(err)
	}

	if len(routes) == 0 {
		return order, route, errpath.Errorf("routes list is empty")
	}
	route = routes[0].UnrestrictedValue

	//Сохраняем промежуточные точки
	order, err = h.DB.SaveOrderRoute(ctx, orderUUID, routeType, routes[0])
	if err != nil {
		return order, route, errpath.Err(err)
	}

	return order, route, nil
}

func routeType(routeType string) string {
	switch routeType {
	case DepartureIntent:
		return "departure"
	case ArrivalIntent:
		return "arrival"
	default:
		return "departure"
	}
}

//Getting address from
func addressFromIntet(answ dialogflow.NLPResponse) (string, error) {
	var full_address, street, building, house, letter, publicPlace, city string
	switch answ.Intent {
	case proto.States.Taxi.Order.Departure.S():
		street = DepartAddressStreet
		building = DepartAddressBuilding
		house = DepartAddressHouse
		letter = DepartAddressLetter
		publicPlace = DepartPublicPlace
		city = DepartCity
	case proto.States.Taxi.Order.Arrival.S():
		street = ArrivalAddressStreet
		building = ArrivalAddressBuilding
		house = ArrivalAddressHouse
		letter = ArrivalAddressLetter
		publicPlace = ArrivalPublicPlace
		city = ArrivalCity
	case string(proto.Consts.Intents.AddressIntent):
		street = DepartAddressStreet
		building = DepartAddressBuilding
		house = DepartAddressHouse
		letter = DepartAddressLetter
		publicPlace = DepartPublicPlace
		city = DepartCity
	}

	if val, ok := answ.Entities[city]; ok {
		full_address = full_address + val + " "
	}

	if val, ok := answ.Entities[publicPlace]; ok {
		full_address = full_address + val + " "
	}

	if val, ok := answ.Entities[street]; ok {
		full_address = full_address + val
		//номер дома
		if bval, ok := answ.Entities[house]; ok {
			cutDot(&bval)
			if bval != "" {
				full_address = full_address + ", " + bval
			}
		}
		//номер корпуса
		if bval, ok := answ.Entities[building]; ok {
			if bval != "" {
				full_address = full_address + " " + bval
			}
		}
		//литер
		if bval, ok := answ.Entities[letter]; ok {
			if bval != "" {
				full_address = full_address + " " + bval
			}
		}
		if full_address == "" {
			return "", errors.New("Address is empty")
		}

		return full_address, nil
	} else {
		return "", errors.New("Can't find address entiti in intent answer")
	}
}

func cutDot(s *string) {
	ival := strings.Split(*s, ".")
	*s = ival[0]
}

// SendMessageToClientAndCRM - отправляет в црмку сообщения написанные ботом
func (h *Handler) SendMessageToClientAndCRM(msgfb *structures.MessageFromBot, chatID int64, msg string, keyboard ...proto.ButtonsSet) error {
	var err error

	msgfb.Source = string(structures.TelegramBotMember)
	msgfb.Text = msg
	err = h.Pub.NewMsg(msgfb)
	if err != nil {
		return errpath.Err(err)
	}
	_, err = h.Telegram.SendMessage(chatID, msg, keyboard...)
	if err != nil {
		return errpath.Err(err)
	}

	return nil
}

//=========================================================================================

// Готовим кнопки из списка тарифов
func getTariffButtons(t []models.ShortTariff, curentAddress string) proto.ButtonsSet {
	var buttonsRows []proto.MsgKeyboardRows

	buttonsRows = append(buttonsRows, proto.MsgKeyboardRows{MsgButtons: []proto.MsgButton{
		{
			Text: string(proto.Consts.BotSend.Buttons.Taxi.AnotherArrivalAdress),
			Data: cutButtonData(ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.AnotherArrivalAdress), curentAddress)),
		},
	}})

	for _, v := range t {
		var buttons []proto.MsgButton
		b := proto.MsgButton{
			Text: fmt.Sprintf("%s - %v ₽", v.Name, v.TotalPrice),
			Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.PaymentChoice), v.ServiceUUID),
		}
		buttons = append(buttons, b)
		buttonsRows = append(buttonsRows, proto.MsgKeyboardRows{MsgButtons: buttons})
	}

	//buttonsRows = append(buttonsRows, proto.MsgKeyboardRows{MsgButtons: []proto.MsgButton{
	//	{
	//		Text: "предложить свою цену [#]",
	//		Data: ButtonDataAddValues("", "[#]", "[#]"),
	//	},
	//}})

	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons:         buttonsRows,
	}

}

//
//

func getWelcomeButtons() proto.ButtonsSet {

	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Reply,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Welcome.GetTaxi),
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.OrderStart)),
					},
				},
			},
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Welcome.OrderFood),
						Type: proto.Consts.ButtonsTypes.Regular,
					},
				},
			},
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Welcome.TakingFireNeedAssistance),
						Type: proto.Consts.ButtonsTypes.Regular,
					},
				},
			},
		},
	}
}

func getSendLocationButtons() proto.ButtonsSet {
	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Reply,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Taxi.SendMyLocation),
						Type: proto.Consts.ButtonsTypes.Location,
					},
				},
			},
		},
	}
}

func getAnotherDepartureAddressButtons(curentAddress string) proto.ButtonsSet {
	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Taxi.AnotherDepartureAddress),
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.AnotherDepartureAdress)),
					},
				},
			},
		},
	}
}

func getAnotherArrivalAddressButtons(curentAddress string) proto.ButtonsSet {
	return proto.ButtonsSet{
		DisplayLocation: proto.Consts.ButtonsDisplayLocation.Inline,
		Buttons: []proto.MsgKeyboardRows{
			{
				MsgButtons: []proto.MsgButton{
					{
						Text: string(proto.Consts.BotSend.Buttons.Taxi.AnotherArrivalAdress),
						Data: ButtonDataAddValues("", string(proto.Consts.BotSend.ButtonsActions.AnotherArrivalAdress)),
					},
				},
			},
		},
	}
}
