package handler

import (
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

func getAnswerButtons(msg *models.ChatMsgFull) proto.ButtonsSet {
	switch msg.State {
	case proto.States.Taxi.Order.Arrival.S():
		return proto.GetArrivalButtons()
	case proto.States.Food.Order.Create.S():
		return proto.ButtonsSet{
			DisplayLocation: proto.Buttons.Display.Inline,
			Buttons: []proto.MsgKeyboardRows{
				{
					MsgButtons: []proto.MsgButton{
						{
							Text: proto.Buttons.Menu.CallTaxi.T(),
							Data: proto.Buttons.Menu.CallTaxi.D(),
						}}}},
		}
	case proto.States.Taxi.Order.NeedPhone.S():
		return proto.GetContactButton()

	case proto.States.Taxi.Order.OfferCancelled.S():
		return proto.GetWellcomeButtons()

	case proto.States.Taxi.Order.DriverNotFound.S():
		return proto.GetWellcomeButtons()
	case proto.States.Taxi.Order.Departure.S():
		return proto.GetFixDepartureAddress()
	case proto.States.Taxi.Order.FixDeparture.S(), proto.States.Taxi.Order.FixArrival.S():
		return proto.GetFixAddressButtons()
	case proto.States.Taxi.Order.ChangeService.S():
		return proto.ChangeServiceButtons(msg.Order.OrderPrefs.Tariffs)
	}

	return proto.ButtonsSet{}
}
