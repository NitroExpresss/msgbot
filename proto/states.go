package proto

import (
	"gitlab.com/faemproject/backend/faem/pkg/variables"
)

//States описываю все возможные типы контекста разговора, именно эти значения должны
//принимать входящий контексты и сами названия интентов, и они же используются в стейт машине
var States struct {
	Welcome        Constant
	FallBackIntent Constant
	Taxi           struct { //такси
		Order struct { //этапы заказа
			CreateDraft   Constant //создание нового заказа
			Departure     Constant //откуда забрать
			Arrival       Constant //куда поедем
			FixDeparture  Constant //исправляем адрес
			FixArrival    Constant //исправляем адрес
			PaymentMethod Constant //изменить способ оплаты
			PaymentByCash Constant //оплата картой (?)
			ChangeService Constant //изменить тариф

			OrderCreated      Constant //запуск заказа
			SmartDistribution Constant //распределение
			FindingDriver     Constant //поиск авто
			OfferOffered      Constant //заказ предложен
			OfferCancelled    Constant //заказ отклонен водителем
			DriverAccepted    Constant //заказ принят
			DriverFounded     Constant //специальный статус в которые переводится того как нашли водителя

			OrderStart Constant //поиск авто
			Waiting    Constant //ожидание

			OnPlace  Constant //на месте
			OnTheWay Constant //выполнение заказа

			OrderPayment   Constant //оплата заказа
			Finished       Constant //заказ завершен
			Cancelled      Constant //заказ отменен
			DriverNotFound Constant //водитель не найден
			Unknown        Constant //заказ завершен

			NeedPhone Constant //запус заказа
		}
	}
	Unknown struct { //непонятные ситуации
		UnexpectedContact     Constant //когда внезапно получил контакт
		UnexpectedCoordinates Constant //когда внезапно получил контакт
	}

	Food struct {
		Order struct {
			Create Constant
		}
	}
}

func initStates() {
	//States taxi
	States.Welcome = "welcome"
	States.FallBackIntent = "Default Fallback Intent"
	States.Taxi.Order.CreateDraft = "taxi_order_create"
	States.Taxi.Order.Departure = "taxi_order_departure"
	States.Taxi.Order.Arrival = "taxi_order_arrival"
	States.Taxi.Order.FixDeparture = "taxi_order_fix-departure"
	States.Taxi.Order.FixArrival = "taxi_order_fix-arrival"
	States.Taxi.Order.PaymentMethod = "taxi_order_payment-method"
	States.Taxi.Order.PaymentByCash = "taxi_order_payment_cash"
	States.Taxi.Order.ChangeService = "taxi_order_change-service"
	States.Taxi.Order.NeedPhone = "taxi_order_need-phone"
	States.Taxi.Order.DriverFounded = "taxi_order_driver_founded"
	States.Taxi.Order.OrderCreated = Constant(variables.OrderStates["OrderCreated"])
	States.Taxi.Order.SmartDistribution = Constant(variables.OrderStates["SmartDistribution"])
	States.Taxi.Order.OfferOffered = Constant(variables.OrderStates["Offered"])
	States.Taxi.Order.OfferCancelled = Constant(variables.OrderStates["OrderCancelledState"])
	States.Taxi.Order.DriverAccepted = Constant(variables.OrderStates["DriverAccepted"])
	States.Taxi.Order.FindingDriver = Constant(variables.OrderStates["FindingDriver"])
	States.Taxi.Order.OrderStart = Constant(variables.OrderStates["Start"])
	States.Taxi.Order.OnTheWay = Constant(variables.OrderStates["OnTheWay"])
	States.Taxi.Order.OnPlace = Constant(variables.OrderStates["OnPlace"])
	States.Taxi.Order.Waiting = Constant(variables.OrderStates["Waiting"])
	States.Taxi.Order.OrderPayment = Constant(variables.OrderStates["OrderPayment"])
	States.Taxi.Order.Finished = Constant(variables.OrderStates["Finished"])
	States.Taxi.Order.Cancelled = Constant(variables.OrderStates["OrderCancelledState"])
	States.Taxi.Order.DriverNotFound = Constant(variables.OrderStates["DriverNotFound"])

	States.Taxi.Order.Unknown = "unknown_state"

	States.Unknown.UnexpectedContact = "unexpected_contact"
	States.Unknown.UnexpectedCoordinates = "unexpected_coordinates"

	States.Food.Order.Create = "food_order_create"

	//States taxi
	States.Food.Order.Create = "food_order_create"
}
