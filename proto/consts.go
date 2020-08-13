package proto

const (
	// ButtonTextSize int = 64 // не проверенно но точно больше 64
	// ButtonDataSize - максимальное количество символов для записи мета данных для кнопки
	ButtonDataSize int = 64
)

// Constant
type Constant string

//S returns string
func (c *Constant) S() string {
	return string(*c)
}

//System системные константы
var System struct {
	RouteTypes struct {
		Departure Constant
		Arrival   Constant
	}
	CRMAdresses struct {
		FindAddress Constant
	}
}

// Consts - объек группирующий и хранящий в себе константы
var Consts struct {
	Intents struct {
		Skip                  Constant
		DefaultFallbackIntent Constant
		Welcome               Constant
		TaxiCallIntent        Constant
		AddressIntent         Constant
		DepartureIntent       Constant
		ArrivalIntent         Constant
	}
	// ButtonActions struct {
	// 	ButtonActions map[Constant]Constant
	// 	OrderStart    Constant
	// 	AnotherAdress Constant
	// }
	BotSend struct {
		Answers struct {
			Welcome struct {
				WhatYouWant Constant
			}
			// обработка статусов заказа приходящих с CRM
			ForStates struct {
				OrderStateUpdatedMsg   Constant
				CarFoundedMsg          Constant
				WillArrivePhraseNoTime Constant
				WillArrivePhrase       Constant
				OnPlace                Constant
				OnTheWay               Constant
				OrderPayment           Constant
				Finished               Constant
				OrderCanceled          Constant
			}
			OrderCreated               Constant
			Received                   Constant
			WhereToGoing               Constant
			AddressNotFound            Constant
			WaitingForOperator         Constant
			ErrorOccurs                Constant
			EnterYourPhone             Constant
			AdressChoices              Constant
			ContactDataSaved           Constant
			DepartureAddress           Constant
			ArrivalAddress             Constant
			RewriteAddress             Constant
			MsgForCreatingWithOperator Constant
			PaymentChoice              Constant
			WaitForCompletion          Constant
			Symbol                     struct {
				Star Constant
			}
		}
		Buttons struct {
			Welcome struct {
				GetTaxi                  Constant
				OrderFood                Constant
				OrderDelivery            Constant
				TakingFireNeedAssistance Constant
			}
			Taxi struct {
				SendMyLocation          Constant
				AnotherDepartureAddress Constant
				AnotherArrivalAdress    Constant
				SendContactButton       Constant
			}
		}
		ButtonsActions struct {
			ChoiceTariff           Constant
			OrderStart             Constant
			PaymentChoice          Constant
			CreatingWithOperator   Constant
			AnotherDepartureAdress Constant
			AnotherArrivalAdress   Constant
			SetDepartureAdress     Constant
			SetArrivalAdress       Constant
			RewriteAddress         Constant
			CancelOrder            Constant
			SetRating              Constant
		}
	}
	ButtonsDisplayLocation struct {
		Inline Constant
		Reply  Constant
	}
	ButtonsTypes struct {
		Regular  Constant
		Contact  Constant
		Location Constant
	}
	MsgSources struct {
		Telegram Constant
		Bot      Constant
		Operator Constant
	}

	Order struct {
		CreationStates struct {
			StartChatting          Constant
			SetDepartureAddress    Constant
			SetArrivalAddress      Constant
			ServiceChoice          Constant
			PaymentChoice          Constant
			ProcessingWithOperator Constant
			OrderCanceled          Constant
		}
		SetRoute struct {
			Departure Constant
			Arrival   Constant
		}
	}

	EndPoints struct {
		CRM    struct{}
		Client struct {
			FindAdress Constant
		}
	}
}

func init() {
	initStates()
	initButtons()
	initAnswers()
	initSystemVars()

	Consts.Intents.Skip = "skip"
	Consts.Intents.DefaultFallbackIntent = "Default Fallback Intent"
	Consts.Intents.Welcome = "Welcome Intent"
	Consts.Intents.TaxiCallIntent = "Taxi call intent"
	Consts.Intents.AddressIntent = "Address intent"
	Consts.Intents.ArrivalIntent = "ArrivalAddressIntent"
	Consts.Intents.DepartureIntent = "DepartureAddressIntent"

	Consts.BotSend.Answers.Welcome.WhatYouWant = "чего желает мой повелитель?"

	Consts.BotSend.Answers.DepartureAddress = "🚕 Адрес подачи: %s\n\nКуда едем?"
	Consts.BotSend.Answers.ArrivalAddress = "🚕 Адрес подачи: %s\n🚕 Адрес назначения: %s\n\nВыбирайте тариф"
	Consts.BotSend.Answers.Received = "полученно"
	Consts.BotSend.Answers.WhereToGoing = "куда поедем?"
	Consts.BotSend.Answers.AddressNotFound = "Адреса: %s, нет у меня в базе. Пожалуйста, попробуйте другой адресс"
	Consts.BotSend.Answers.WaitingForOperator = "Ожидайте ответа от оператора"
	Consts.BotSend.Answers.ErrorOccurs = "Произошка какая-то ошибка, мне очень жаль 😢"
	Consts.BotSend.Answers.EnterYourPhone = "Мне нужен ваш номер телефона, что бы оформить заказ 😝"
	Consts.BotSend.Answers.ContactDataSaved = "Отлично, номерок записан..."
	Consts.BotSend.Answers.AdressChoices = "есть вот такие варианты адресов:"
	Consts.BotSend.Answers.RewriteAddress = "введите новый адрес"
	Consts.BotSend.Answers.MsgForCreatingWithOperator = "созданно оператором"
	Consts.BotSend.Answers.OrderCreated = "Заказ создан. Начинаю искать свободную машину в вашем районе"
	Consts.BotSend.Answers.PaymentChoice = "Если тариф не верный просто выберите другой из предыдущего меню.\n\nОплатить заказ пока можно только наличными:"
	Consts.BotSend.Answers.WaitForCompletion = "Дождитесь окончания выполнения заказа"

	Consts.BotSend.Answers.ForStates.WillArrivePhrase = "Через %s к вам подъедет\n%s цвет %s номер %s"
	Consts.BotSend.Answers.ForStates.OrderStateUpdatedMsg = "Статус вашего заказа изменен.\n\nТекущий статус: %s"
	Consts.BotSend.Answers.ForStates.CarFoundedMsg = "Ура, мы нашли для вас машину!\n\nЧерез %v минут к вам подъедет %s %s номера %s \n\nПожалуйста, ожидайте..."
	Consts.BotSend.Answers.ForStates.WillArrivePhraseNoTime = "%s %s номер %s"
	Consts.BotSend.Answers.ForStates.OnPlace = "Такси на месте, пожалуйста выходите.\n\nБесплатное ожидание: %v мин"
	Consts.BotSend.Answers.ForStates.OnTheWay = "Поехали!\nСледующая остановка: %s\n\nПриятной поездки!"
	Consts.BotSend.Answers.ForStates.OrderPayment = "Ваша поездка окончена.\n\nСтоимость поездки: %v\n\nСпасибо! Будем ждать вас снова"
	Consts.BotSend.Answers.ForStates.Finished = "Мы постоянно работаем над качеством наших услуг.\n\nПожалуйста оцените вашу последнюю поездку:"
	Consts.BotSend.Answers.ForStates.OrderCanceled = "Ваш заказ отменен"

	Consts.BotSend.Answers.Symbol.Star = "*"

	Consts.BotSend.Buttons.Welcome.GetTaxi = "Заказать Такси 🚕"
	Consts.BotSend.Buttons.Welcome.OrderFood = "Заказать Еду 🍕"
	Consts.BotSend.Buttons.Welcome.OrderDelivery = "Заказать Доставку 🚚"
	Consts.BotSend.Buttons.Welcome.TakingFireNeedAssistance = "написать в поддержку"

	// Consts.ButtonActions.OrderStart = "start_order"
	// Consts.ButtonActions.AnotherAdress = "another_adress"
	// Consts.ButtonActions.ButtonActions = map[Constant]Constant{
	// 	Consts.BotSend.Buttons.Welcome.GetTaxi:                  Consts.ButtonActions.OrderStart,
	// 	Consts.BotSend.Buttons.Welcome.OrderFood:                ConsSetDepartureAdressts.ButtonActions.OrderStart,
	// 	Consts.BotSend.Buttons.Welcome.OrderDelivery:            Consts.ButtonActions.OrderStart,
	// 	Consts.BotSend.Buttons.Welcome.TakingFireNeedAssistance: Consts.ButtonActions.OrderStart,
	// }

	Consts.BotSend.Buttons.Taxi.SendMyLocation = "отправить мою геолокацию"
	Consts.BotSend.Buttons.Taxi.AnotherDepartureAddress = "адрес подачи не верный?"
	Consts.BotSend.Buttons.Taxi.AnotherArrivalAdress = "адрес назначения не верный?"
	Consts.BotSend.Buttons.Taxi.SendContactButton = "Поделиться моим контактом"

	Consts.BotSend.ButtonsActions.ChoiceTariff = "choice_tariff"
	Consts.BotSend.ButtonsActions.OrderStart = "start_order"
	Consts.BotSend.ButtonsActions.PaymentChoice = "payment_choice"
	Consts.BotSend.ButtonsActions.CreatingWithOperator = "creating_with_operator"
	Consts.BotSend.ButtonsActions.AnotherDepartureAdress = "another_departure_address"
	Consts.BotSend.ButtonsActions.AnotherArrivalAdress = "another_arrival_address"
	Consts.BotSend.ButtonsActions.SetDepartureAdress = "set_departure_address"
	Consts.BotSend.ButtonsActions.SetArrivalAdress = "set_arrival_address"
	Consts.BotSend.ButtonsActions.RewriteAddress = "rewrite_address"
	Consts.BotSend.ButtonsActions.CancelOrder = "order_cancel"
	Consts.BotSend.ButtonsActions.SetRating = "set_rating"

	Consts.ButtonsDisplayLocation.Inline = "]"
	Consts.ButtonsDisplayLocation.Reply = "reply"

	Consts.ButtonsTypes.Regular = "regular"
	Consts.ButtonsTypes.Contact = "contact"
	Consts.ButtonsTypes.Location = "location"

	Consts.MsgSources.Telegram = "telegram"
	Consts.MsgSources.Bot = "Бот"
	Consts.MsgSources.Operator = "Оператор"

	//
	Consts.EndPoints.Client.FindAdress = "/findaddress"

	//
	Consts.Order.CreationStates.StartChatting = "just_start_chatting" // TODO: возможно для црмки надо отправить order_draft
	Consts.Order.CreationStates.SetDepartureAddress = "set_departure_address"
	Consts.Order.CreationStates.SetArrivalAddress = "set_arrival_address"
	Consts.Order.CreationStates.ServiceChoice = "service_choice"
	Consts.Order.CreationStates.PaymentChoice = "payment_choice"
	Consts.Order.CreationStates.ProcessingWithOperator = "order_processing_with_operator"
	Consts.Order.CreationStates.OrderCanceled = "order_canceled"

	Consts.Order.SetRoute.Departure = "departure"
	Consts.Order.SetRoute.Arrival = "arrival"

}

//объявление системных переменных
func initSystemVars() {
	System.RouteTypes.Departure = "departure"
	System.RouteTypes.Arrival = "arrival"
	System.CRMAdresses.FindAddress = "/findaddress"
}
