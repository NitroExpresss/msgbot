package proto

var MsgTypes struct {
	TelegramMessage  Constant
	TelegramCallback Constant
	TelegramContact  Constant
	Coordinates      Constant
	BrokerMessage    Constant
}

type ButtonType struct {
	Text Constant
	Data Constant
}

//D(ata) возращает ключ кнопки
func (b *ButtonType) D() string {
	return b.Data.S()
}

//T(ext) возращает текст кнопки
func (b *ButtonType) T() string {
	return b.Text.S()
}

var Buttons struct {
	Menu struct {
		CallTaxi  ButtonType
		OrderFood ButtonType
	}
	Actions struct {
		WrongDepartureAddress ButtonType
		WrongArrivalAddress   ButtonType
		ChangeTarrif          ButtonType
		ChangePaymentType     ButtonType
		StartOrder            ButtonType
		CancelOrder           ButtonType
		WrongAddress1         ButtonType
		WrongAddress2         ButtonType
		WrongAddress3         ButtonType
		WrongAddress4         ButtonType
		WrongAddress5         ButtonType
		BackButton            ButtonType
	}
	Reply struct {
		NeedPhone ButtonType
	}
	Display struct {
		Inline Constant
		Reply  Constant
	}
	Type struct {
		Regular  Constant
		Contact  Constant
		Location Constant
	}
}

func initButtons() {
	//Action
	Buttons.Actions.WrongDepartureAddress.Text = "🚕 Адрес подачи не верный"
	Buttons.Actions.WrongDepartureAddress.Data = States.Taxi.Order.FixDeparture
	Buttons.Actions.WrongArrivalAddress.Text = "✏️ Изменить адрес назначения"
	Buttons.Actions.WrongArrivalAddress.Data = States.Taxi.Order.FixArrival

	Buttons.Actions.WrongAddress1.Text = "1️⃣"
	Buttons.Actions.WrongAddress1.Data = "1"
	Buttons.Actions.WrongAddress2.Text = "2️⃣"
	Buttons.Actions.WrongAddress2.Data = "2"
	Buttons.Actions.WrongAddress3.Text = "3️⃣"
	Buttons.Actions.WrongAddress3.Data = "3"
	Buttons.Actions.WrongAddress4.Text = "4️⃣"
	Buttons.Actions.WrongAddress4.Data = "4"
	Buttons.Actions.WrongAddress5.Text = "5️⃣"
	Buttons.Actions.WrongAddress5.Data = "5"

	Buttons.Actions.BackButton.Text = "🔙 Назад"
	Buttons.Actions.BackButton.Data = "back_button"

	Buttons.Actions.ChangeTarrif.Text = "🚘 Изменить тариф"
	Buttons.Actions.ChangeTarrif.Data = States.Taxi.Order.ChangeService

	Buttons.Actions.ChangePaymentType.Text = "💳 Изменить cпособ оплаты"
	Buttons.Actions.ChangePaymentType.Data = States.Taxi.Order.PaymentMethod
	Buttons.Actions.StartOrder.Text = "🚕 Вызвать такси"
	Buttons.Actions.StartOrder.Data = States.Taxi.Order.OrderCreated

	Buttons.Actions.CancelOrder.Text = "❌ Отменить заказ"
	Buttons.Actions.CancelOrder.Data = States.Taxi.Order.Cancelled

	//Текстовые кнопки
	Buttons.Menu.CallTaxi.Text = "🚕 Заказать Такси"
	Buttons.Menu.CallTaxi.Data = States.Taxi.Order.CreateDraft

	Buttons.Menu.OrderFood.Text = "🍕 Заказать Еду"
	Buttons.Menu.OrderFood.Data = States.Food.Order.Create

	//Reply кнопки
	Buttons.Reply.NeedPhone.Text = "Поделиться моим контактом"

	//Типы кнопок
	Buttons.Type.Regular = "regular"
	Buttons.Type.Contact = "contact"
	Buttons.Type.Location = "location"

	//Расположение кнопок
	Buttons.Display.Inline = "inline"
	Buttons.Display.Reply = "reply"

	//Message Types
	MsgTypes.TelegramMessage = "tlgrm_msg"
	MsgTypes.TelegramCallback = "tlgrm_callback"
	MsgTypes.TelegramContact = "tlgrm_contact"
	MsgTypes.BrokerMessage = "broker_msg"
	MsgTypes.Coordinates = "coordinates_msg"
}
