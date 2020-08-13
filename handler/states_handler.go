// основной файл управления статусами диалога и вызова функций генерации ответа и кнопок

package handler

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/looplab/fsm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"strings"
)

type TaxiOrderStates struct {
	From string
	To   string
	FSM  *fsm.FSM
}

//структура для ответа
type answerType struct {
	textAnswer    string           //сам текст
	buttonsAnswer proto.ButtonsSet //инлайн кнопки
	msgId         int              //id сообщение, 0 - значит новое
}

var statesToSkip = []string{
	proto.States.Taxi.Order.SmartDistribution.S(),
	proto.States.Taxi.Order.OfferOffered.S(),
	proto.States.Taxi.Order.FindingDriver.S(),
	proto.States.Taxi.Order.Waiting.S(),
	proto.States.Taxi.Order.OrderPayment.S(),
	proto.States.Taxi.Order.DriverAccepted.S(),
	proto.States.Taxi.Order.DriverFounded.S(),
}

//getTextAnswer возвращает ответ и кноки для мессенджера
func (h *Handler) getTextAnswer(msg *models.ChatMsgFull) (answerType, error) {

	//если welcome то показываем приветствие
	if msg.Type == proto.MsgTypes.TelegramMessage.S() && msg.DFAnswer.Intent == proto.States.Welcome.S() {
		return answerType{
			textAnswer:    msg.DFAnswer.Answer,
			buttonsAnswer: proto.GetWellcomeButtons(),
			msgId:         0,
		}, nil
	}

	//Fall Back Intent
	if msg.DFAnswer.Intent == proto.States.FallBackIntent.S() {
		return answerType{
			textAnswer:    proto.GetErrorText(msg.State),
			buttonsAnswer: proto.ButtonsSet{},
			msgId:         0,
		}, nil
	}

	//обрабатываем обращение в зависимости от его типа (сообщение или callback)
	var action string
	var err error
	switch msg.Type {
	case proto.MsgTypes.TelegramMessage.S():
		action = msg.DFAnswer.Intent
	case proto.MsgTypes.TelegramCallback.S():
		action, err = getCallbackActionData(msg)
		if err != nil {
			txt, but, err := getErrorData(msg, err)
			return answerType{
				textAnswer:    txt,
				buttonsAnswer: but,
				msgId:         0,
			}, err
		}
	case proto.MsgTypes.Coordinates.S():
		switch msg.State {
		case proto.States.Taxi.Order.CreateDraft.S():
			action = proto.States.Taxi.Order.Departure.S()
		case proto.States.Taxi.Order.Departure.S():
			action = proto.States.Taxi.Order.Arrival.S()
		default:
			msg.State = proto.States.Unknown.UnexpectedCoordinates.S()
			answ, but, msgID := outputTextAndButtons(msg)
			return answerType{
				textAnswer:    answ,
				buttonsAnswer: but,
				msgId:         msgID,
			}, nil
		}

	case proto.MsgTypes.TelegramContact.S():
		if msg.State == proto.States.Taxi.Order.NeedPhone.S() {
			err = h.saveClientContact(msg)
			if err != nil {
				handleEventErrors(msg, err)
			}
			action = proto.States.Taxi.Order.OrderCreated.S()
		} else {
			//если мы получаем контакт не том статусе то отвечаем
			msg.State = proto.States.Unknown.UnexpectedContact.S()
			answ, but, msgID := outputTextAndButtons(msg)
			return answerType{
				textAnswer:    answ,
				buttonsAnswer: but,
				msgId:         msgID,
			}, nil
		}
	case proto.MsgTypes.BrokerMessage.S():
		action = msg.Order.OrderPrefs.NewState

	default:
		return answerType{
			textAnswer:    "",
			buttonsAnswer: proto.ButtonsSet{},
			msgId:         0,
		}, errors.New("Unknownd message type")
	}
	log := logs.Eloger.WithFields(logrus.Fields{
		"reason":    "debbuging FSM",
		"action":    action,
		"OrderUUID": msg.OrderUUID,
	})

	//устанавливаем текущий статус в стейт машину
	msg.FSM.SetState(msg.State)

	log.WithFields(logrus.Fields{
		"Msg state": msg.State,
		"FSM State": msg.FSM.Current(),
		"DF":        msg.DFAnswer.Intent,
		"OrderUUID": msg.OrderUUID,
	}).Debug("Before")

	//переводим в новый статус и проводим различные манипуляции с данными
	if msg.FSM.Current() != action {
		err = msg.FSM.Event(action, msg)
		if err != nil {
			//перехватываем некоторые типы ошибок, например если нет номера при создании заказа
			//в этом например случае это не ошибка, а просто новый сценарий
			answ, but, newErr := handleEventErrors(msg, err)
			if newErr != nil {
				return answerType{
					textAnswer:    answ,
					buttonsAnswer: but,
					msgId:         0,
				}, newErr
			}
		}
	}

	//не на все статусы надо отвечать, некоторые пропускаем
	if skipState(msg.State) {
		return answerType{
			textAnswer:    skipStateConstant,
			buttonsAnswer: proto.ButtonsSet{},
			msgId:         0,
		}, nil
	}
	//генерируем ответ
	answ, but, msgId := outputTextAndButtons(msg)
	return answerType{
		textAnswer:    answ,
		buttonsAnswer: but,
		msgId:         msgId,
	}, nil
}

func skipState(state string) bool {
	for _, v := range statesToSkip {
		if v == state {
			return true
		}
	}
	return false
}
func (h *Handler) saveClientContact(msg *models.ChatMsgFull) error {
	fullMsg, ok := msg.Payload.(*tgbotapi.Update)
	if !ok {
		return errors.New(fmt.Sprintf("Cant transform interface to ChatMsgFull on get user data functuon"))
	}
	getMyContact := proto.MessangerContact{
		PhoneNumber: fullMsg.Message.Contact.PhoneNumber,
		FirstName:   fullMsg.Message.Contact.FirstName,
		LastName:    fullMsg.Message.Contact.LastName,
		UserID:      fullMsg.Message.Contact.UserID,
	}
	if getMyContact.PhoneNumber == "" {
		return errors.New(fmt.Sprintf("Empty phone number cant register"))
	}
	ctx := context.Background()
	err := h.DB.SaveUserContact(ctx, getMyContact, "telegram")
	if err != nil {
		return errors.Wrap(err, "failed to save user contact")
	}
	return nil
}

func outputTextAndButtons(msg *models.ChatMsgFull) (string, proto.ButtonsSet, int) {
	log := logs.Eloger.WithFields(logrus.Fields{
		"event": "generate text",
	})
	log.Debug(msg.FSM.Current())

	var templateText string
	var err error

	//берем шаблон из интента DF или справочника внутри системы
	if msg.Type == proto.MsgTypes.TelegramMessage.S() {
		templateText = msg.DFAnswer.Answer
	} else {
		//если это колбек то берет из справочника
		templateText, err = proto.GetIntentText(proto.Constant(msg.State))
		if err != nil {
			log.WithFields(logrus.Fields{
				"reason": "cant get intent text",
			}).Error(err)
		}
	}

	//смешиваем полученный ответ с данными заказа
	answerText, err := mixedOrderDataToAnswer(templateText, msg)
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason": "using template to get answer",
			"type":   "",
		}).Error(err)
	}

	//получаем кнопки для данного сообщения
	answerButtons := getAnswerButtons(msg)

	//новое сообщение или обновляем существующее
	msgId := newOrUpdateMessage(msg)

	return answerText, answerButtons, msgId
}

func newOrUpdateMessage(msg *models.ChatMsgFull) int {
	switch msg.State {
	case proto.States.Taxi.Order.FixArrival.S():
		return msg.MsgID
	case proto.States.Taxi.Order.FixDeparture.S():
		return msg.MsgID
	case proto.States.Taxi.Order.ChangeService.S():
		return msg.MsgID
	case proto.States.Taxi.Order.Arrival.S():
		//если мы пришли на этап из этого же сообщения
		val, ok := msg.Order.OrderPrefs.MsgsIDs[proto.States.Taxi.Order.Arrival.S()]
		if ok && val == msg.MsgID {
			return msg.MsgID
		}
		return 0
	default:
		return 0
	}
}

//handleEventErrors перехватываем ошибки ивентов
func handleEventErrors(msg *models.ChatMsgFull, err error) (string, proto.ButtonsSet, error) {
	//пытаемся получить код
	splitError := strings.Split(err.Error(), "$")

	if len(splitError) < 2 {
		return getErrorData(msg, err)
	}

	switch splitError[1] {
	//у клиента нет номера
	case noUserPhoneErrorCode:
		newErr := msg.FSM.Event(proto.States.Taxi.Order.NeedPhone.S(), msg)
		if newErr != nil {
			return getErrorData(msg, newErr)
		}
		return "", proto.ButtonsSet{}, nil
	default:
		return getErrorData(msg, err)
	}
}

//getErrorData возращает типовую ошибку
func getErrorData(msg *models.ChatMsgFull, err error) (string, proto.ButtonsSet, error) {
	return proto.GetErrorText(msg.State), proto.ButtonsSet{}, err
}

//getCallbackActionData обрабатывает полученные коллбеки, в некоторых случаях нужно провести некоторые манипуляции
//например когда получаем один из номеров для исправления адреса то action для всех один а саму структуру расширяем
//нужным параметром
func getCallbackActionData(msg *models.ChatMsgFull) (string, error) {
	cbData, ok := msg.Payload.(*tgbotapi.Update)
	if !ok {
		return "", errors.New("unknown datatype while parsing callback data")
	}
	data := cbData.CallbackQuery.Data

	//если мы на этапе исправления адреса
	if (msg.State == proto.States.Taxi.Order.FixDeparture.S()) || (msg.State == proto.States.Taxi.Order.FixArrival.S()) {
		//сначала запомним на какую кнопку нажали
		switch data {
		case proto.Buttons.Actions.WrongAddress1.D():
			msg.Order.OrderPrefs.FixAddressVariant = 1
		case proto.Buttons.Actions.WrongAddress2.D():
			msg.Order.OrderPrefs.FixAddressVariant = 2
		case proto.Buttons.Actions.WrongAddress3.D():
			msg.Order.OrderPrefs.FixAddressVariant = 3
		case proto.Buttons.Actions.WrongAddress4.D():
			msg.Order.OrderPrefs.FixAddressVariant = 4
		case proto.Buttons.Actions.WrongAddress5.D():
			msg.Order.OrderPrefs.FixAddressVariant = 5
		case proto.Buttons.Actions.BackButton.D():
			msg.Order.OrderPrefs.FixAddressVariant = 99
		}
		//теперь вернем следующий этап
		if msg.State == proto.States.Taxi.Order.FixDeparture.S() {
			return proto.States.Taxi.Order.Departure.S(), nil
		} else if msg.State == proto.States.Taxi.Order.FixArrival.S() {
			return proto.States.Taxi.Order.Arrival.S(), nil
		}
	}

	if msg.State == proto.States.Taxi.Order.ChangeService.S() {
		//Если на этапе смены сервиса и нажали назад
		if data == proto.Buttons.Actions.BackButton.D() {
			return proto.States.Taxi.Order.Arrival.S(), nil
		}
		msg.Order.OrderPrefs.Service = cbData.CallbackQuery.Data
		fmt.Printf("DATA = %s SERVICE = %s\n\n", cbData.CallbackQuery.Data, msg.Order.OrderPrefs.Service)
		return proto.States.Taxi.Order.Arrival.S(), nil
	}

	return cbData.CallbackQuery.Data, nil
}

func validateInput(e *fsm.Event) (*models.ChatMsgFull, error) {
	if len(e.Args) < 1 {
		er := fmt.Sprintf("Didn't get ChatMsgFull as argument on `leave_state`. From %s to %s", e.Src, e.Dst)
		return &models.ChatMsgFull{}, errors.New(er)
	}
	msg, ok := e.Args[0].(*models.ChatMsgFull)
	if !ok {
		er := fmt.Sprintf("Cant transform interface to ChatMsgFull on `leave_state`. From %s to %s", e.Src, e.Dst)
		return &models.ChatMsgFull{}, errors.New(er)
	}
	return msg, nil
}
