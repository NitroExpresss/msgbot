package proto

import (
	"errors"
	"fmt"
	"math/rand"
)

var Answers struct {
	Errors  []Constant
	Intents map[Constant][]string
}

var errorsAnswers = []string{
	"Что-то пошло не так, мне очень жаль.\nМожет попробуем еще раз.\n\n%s",
	"Не навижу ошибки а-а-а-а, но они случаются, ведь я пока учусь.\nЧто если попробовать еще раз:\n\n%s",
	"Не ошибается только тот, кто ничего не делает (( вообщем у нас проблема\nДумаю стоит попробовать еще раз:\n\n%s",
}

var oldErrorsAnswers = []string{
	"К сожалению произошла ошибка (( \nВсе данные я передал разработчикам, скоро мы это исправим. \n",
	"Очень жаль, но произошла ошибка. #$~$@%$%^&*()!@#$%^&*()_ \nВся информация уже у разработчиков, я прослежу что бы ее решили\n",
	"Оооо нет, - ошибка. Больше всего на свете не навижу ошибки.\nЯ передал разработчикам все информацию, они наверно уже исправляют\n",
}

var TaxiOrderCreated = []string{
	"Создаем заказ такси 🚕\n\nОтправьте мне адрес откуда вас забрать или пришлите свою гео-позицию",
	"🚕 🚕 вези, вези...\n\nПришлите адрес, ткуда вас забрать? А еще можно прислать гео-позицию.",
	"Заказ такси 🚕\n\n...отличный выбор, на такси, у нас лучшие цены в городе.\n\nНа какой адрес прислать авто?",
}

var tempUnavailable = []string{
	"Мне жаль но эта функция пока не доступна. 🧘",
	"Мы еще не успели это сделать. 🤔",
	"Это пока не работает 😒 скоро заработает",
}

var taxiOrderStarted = []string{
	"Понеслась... ищу для вас ближайшее авто 😇",
	"Заказ создан 😉 ищу авто",
}

//var TaxiOrderChangeService = []string{
//	"Вам доступны следующие тарифы:\n",
//}

var TaxiOrderPaymentMethod = tempUnavailable

var ArrivalFixed = []string{
	"🚕 Адрес подачи: %s\n\n🚕 Адрес назначения: %s\n\nСтоимость поездки: %v₽ (%s)\n\nСпособ оплаты: %s\n",
}
var DepartureFixed = []string{
	"Адрес теперь исправлен, как насчет:\n\n%s\n\nЕсли все верно пришлите адрес назначения:",
	"Новый адрес подачи:\n\n%s\n\nКуда направимся:",
}

var TaxiFixDepartureAddress = []string{
	"Адрес: %s не верный?\n\nВведите адрес подачи повторно, или выберете один из предлагаемых:\n\n%s",
}

var TaxiFixArrivalAddress = []string{
	"Адрес: %s не верный?\n\nВведите адрес назначения повторное, или выберете один из предлагаемых:\n\n%s",
}

var FoodOrderCreate = []string{
	"Мне жаль 😔 но этот раздел еще в разработке. \nМы планируем его запустить 15 августа. Я напишу, как он будет готов.\n\nПока можно заказать такси.",
	"эээээ 😶 заказ еды еще не включили.\nОн запуститься в начале августа, я пришлю уведомление как он будет готов.\nА пока можно заказать такси)",
}

var NeedPhone = []string{
	"У меня нет вашего номера телефона. Пришлите его пожалуйста что бы начать поиск авто",
}

var UnexpectedContact = []string{
	"Дико извиняюсь, но я не понимаю что мне делать с этим контактом?\nВернемся к диалогу?",
	"А зачем мне этот контакт, я не понимаю... 😬",
}

var UnexpectedCoordinates = []string{
	"Я не очень понимаю, что мне делать с этими координатами...😬",
	"Коориднаты? А зачем? Не пойму что с ними делать...",
}

var driverFounded = []string{
	"Ура, мы нашли для вас машину!\n\nЧерез %v минут к вам подъедет %s %s номера %s \n\nПожалуйста, ожидайте...",
	"Получилось!\n\nЧерез %v минут к вам подъедет %s %s номера %s \n\nПожалуйста, ожидайте...",
}
var onTheWay = []string{
	"Поехали! Следующая остановка %s.\n\nХорошей поездки.",
	"Следующая остановка %s.\n\nПоехали!",
}

var driverOnPlace = []string{
	"Вас ожидает водитель, пожалуйста выходите",
	"Водитель на месте. Выходите",
}

var orderFinished = []string{
	"Ваш заказ завершен! Спасибо",
	"Приехали, будем рады видеть вас снова.",
}

var orderCancelled = []string{
	"Заказ отменен, будем рады видеть вас снова.\n",
	"Жаль конечно, но ваш заказ отменен 🤷\n",
}

var driverNotFounded = []string{
	"Очень жаль, но мы не нашли водителя и отменили ваш заказ",
	"О, нет! Ваш заказ отменен, мы просто не нашли водителя 😔",
}

///

//фразы и словари
type PhrasesDict struct {
	dict   []string
	phrase Constant
}

func initAnswers() {
	Answers.Intents = make(map[Constant][]string)

	//ошибки
	for _, v := range errorsAnswers {
		Answers.Errors = append(Answers.Errors, Constant(v))
	}

	pDic := []PhrasesDict{
		{
			//создание заказа такси
			dict:   TaxiOrderCreated,
			phrase: States.Taxi.Order.CreateDraft,
		},
		{
			//исправление адреса откуда забрать
			dict:   TaxiFixDepartureAddress,
			phrase: States.Taxi.Order.FixDeparture,
		},
		{
			dict:   TaxiFixArrivalAddress,
			phrase: States.Taxi.Order.FixArrival,
		},
		{
			dict:   ArrivalFixed,
			phrase: States.Taxi.Order.Arrival,
		},
		{
			dict:   DepartureFixed,
			phrase: States.Taxi.Order.Departure,
		},
		{
			//изменить услугу
			dict:   ArrivalFixed,
			phrase: States.Taxi.Order.ChangeService,
		},
		{
			//изменение способа оплаты
			dict:   TaxiOrderPaymentMethod,
			phrase: States.Taxi.Order.PaymentMethod,
		},
		{
			//мы ждем номер для авторизации
			dict:   NeedPhone,
			phrase: States.Taxi.Order.NeedPhone,
		},
		{
			//внезапно прислали контакт
			dict:   UnexpectedContact,
			phrase: States.Unknown.UnexpectedContact,
		},
		{
			//внезапно прислали координаты
			dict:   UnexpectedCoordinates,
			phrase: States.Unknown.UnexpectedCoordinates,
		},

		{
			//создание заказа еды
			dict:   FoodOrderCreate,
			phrase: States.Food.Order.Create,
		},
		{
			dict:   taxiOrderStarted,
			phrase: States.Taxi.Order.OrderCreated,
		},
		{
			dict:   driverFounded,
			phrase: States.Taxi.Order.OrderStart,
		},

		{
			dict:   onTheWay,
			phrase: States.Taxi.Order.OnTheWay,
		},
		{
			dict:   driverOnPlace,
			phrase: States.Taxi.Order.OnPlace,
		},
		{
			dict:   orderFinished,
			phrase: States.Taxi.Order.Finished,
		},
		{
			dict:   orderCancelled,
			phrase: States.Taxi.Order.Cancelled,
		},
		{
			dict:   driverNotFounded,
			phrase: States.Taxi.Order.DriverNotFound,
		},
	}
	for _, d := range pDic {
		for _, e := range d.dict {
			Answers.Intents[d.phrase] = append(Answers.Intents[d.phrase], e)
		}
	}
}

func GetIntentText(state Constant) (string, error) {
	if val, ok := Answers.Intents[state]; ok {
		if len(Answers.Intents[state]) == 1 {
			return val[0], nil
		}
		return val[rand.Intn(len(Answers.Intents[state])-1)], nil
	}
	ans := fmt.Sprintf("intent data to founded in dictionary. State: %s", state.S())
	return "Я не знаю как ответить (( мне очень жаль", errors.New(ans)
}

//GetErrorText возвращает случайные текст с ошибкой
func GetErrorText(currentState string) string {

	template := Answers.Errors[rand.Intn(len(errorsAnswers)-1)].S()
	result := fmt.Sprintf(template, expectState(currentState))

	return result
}

//возращает текст который уведомляет о том что мы ждем от клиента
func expectState(state string) string {
	switch state {
	case States.Welcome.S():
		return "Нажмите на кноку заказа Такси или Еда или нажмите на /start"
	case States.Taxi.Order.CreateDraft.S():
		return "Давайте начнем с адреса, откуда вас забрать?"
	case States.Taxi.Order.Departure.S():
		return "Пожалуйста, пришлите мне текстом адрес куда поедем?"
	case States.Taxi.Order.FixDeparture.S():
		return "Веберете один из предложеных адресов или пришлите новый"
	case States.Taxi.Order.FixArrival.S():
		return "Веберете один из предложеных адресов или пришлите новый"
	case States.Taxi.Order.Arrival.S():
		return "Вы можете вызвать такси, изменить адрес назначения или тариф. Для отмены заказа вы можете создать заказ а уже потом его отменить"
	case States.Taxi.Order.ChangeService.S():
		return "Попробуйте выбрать одну из предложенных выше классов обслуживания"
	case States.Taxi.Order.NeedPhone.S():
		return "Я ожидаю что вы пришлете свой контакт что бы я мог записать адрес"
	case States.Taxi.Order.OrderCreated.S():
		return "Ваш заказ создан. Возможо какая то техническая проблема, я все передам разработчикам"
	case States.Taxi.Order.FindingDriver.S(), States.Taxi.Order.SmartDistribution.S(), States.Taxi.Order.OfferOffered.S():
		return "Мы ищем для вас авто. Если хотите отменить, нажмите `❌ Отменить заказ`, в одном из предыдущих сообщениях"
	default:
		return ""
	}
}
