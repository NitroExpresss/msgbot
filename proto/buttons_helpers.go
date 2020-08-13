package proto

import (
	"fmt"
)

type TariffProto struct {
	ServiceUUID     string `json:"service_uuid"`
	ServiceImage    string `json:"service_image"`
	Name            string `json:"name"`
	Currency        string `json:"currency"`
	BonusPayment    int    `json:"bonus_payment"`
	MaxBonusPayment int    `json:"max_bonus_payment"`
	TotalPrice      int    `json:"total_price"`
}

func GetCancelOrderButton() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.CancelOrder.T(),
						Data: Buttons.Actions.CancelOrder.D(),
					},
				},
			},
		},
	}
}

//GetArrivalButtons возращает кнопки для статусу orders_arrival
func GetArrivalButtons() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.WrongArrivalAddress.T(),
						Data: Buttons.Actions.WrongArrivalAddress.D(),
					},
				},
			},
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.ChangeTarrif.T(),
						Data: Buttons.Actions.ChangeTarrif.D(),
					},
					{
						Text: Buttons.Actions.ChangePaymentType.T(),
						Data: Buttons.Actions.ChangePaymentType.D(),
					},
				},
			},
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.StartOrder.T(),
						Data: Buttons.Actions.StartOrder.D(),
					},
				},
			},
		},
	}
}

func GetFixAddressButtons() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.WrongAddress1.T(),
						Data: Buttons.Actions.WrongAddress1.D(),
					},
					{
						Text: Buttons.Actions.WrongAddress2.T(),
						Data: Buttons.Actions.WrongAddress2.D(),
					},
					{
						Text: Buttons.Actions.WrongAddress3.T(),
						Data: Buttons.Actions.WrongAddress3.D(),
					},
					{
						Text: Buttons.Actions.WrongAddress4.T(),
						Data: Buttons.Actions.WrongAddress4.D(),
					},
					{
						Text: Buttons.Actions.WrongAddress5.T(),
						Data: Buttons.Actions.WrongAddress5.D(),
					},
				},
			},
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.BackButton.T(),
						Data: Buttons.Actions.BackButton.D(),
					},
				},
			},
		},
	}
}

func GetFixDepartureAddress() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Actions.WrongDepartureAddress.T(),
						Data: Buttons.Actions.WrongDepartureAddress.D(),
					},
				},
			},
		},
	}
}

func GetWellcomeButtons() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Menu.CallTaxi.T(),
						Data: Buttons.Menu.CallTaxi.D(),
					},
				},
			},
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Menu.OrderFood.T(),
						Data: Buttons.Menu.OrderFood.D(),
					}}}},
	}
}

func GetContactButton() ButtonsSet {
	return ButtonsSet{
		DisplayLocation: Buttons.Display.Reply,
		Buttons: []MsgKeyboardRows{
			{
				MsgButtons: []MsgButton{
					{
						Text: Buttons.Reply.NeedPhone.T(),
						Type: Buttons.Type.Contact,
					}}}}}
}

func ChangeServiceButtons(tarrifs []TariffProto) ButtonsSet {
	var buttons []MsgKeyboardRows
	for _, v := range tarrifs {
		buttons = append(buttons, MsgKeyboardRows{
			MsgButtons: []MsgButton{
				{
					Text: fmt.Sprintf("%s - %v₽", v.Name, v.TotalPrice),
					Data: v.ServiceUUID,
				},
			},
		})
	}

	buttons = append(buttons, MsgKeyboardRows{
		MsgButtons: []MsgButton{
			{
				Text: Buttons.Actions.BackButton.T(),
				Data: Buttons.Actions.BackButton.D(),
			},
		},
	})

	return ButtonsSet{
		DisplayLocation: Buttons.Display.Inline,
		Buttons:         buttons,
	}
}
