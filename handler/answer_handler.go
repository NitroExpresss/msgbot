package handler

import (
	"errors"
	"fmt"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"math"
	"time"
)

//mixedOrderDataToAnswer - подмешиваем данные из заказа если надо
func mixedOrderDataToAnswer(template string, msg *models.ChatMsgFull) (string, error) {
	switch msg.State {
	case proto.States.Taxi.Order.Departure.S():
		routes := msg.Order.OrderJSON.Routes
		if len(routes) > 0 {
			return fmt.Sprintf(template, msg.Order.OrderJSON.Routes[0].UnrestrictedValue), nil
		} else {
			return proto.GetErrorText(msg.State), errors.New("Tryinig to output route that does not exists")
		}
	case proto.States.Taxi.Order.Arrival.S(), proto.States.Taxi.Order.ChangeService.S():
		routes := msg.Order.OrderJSON.Routes
		if len(routes) > 0 {
			return fmt.Sprintf(
				template,
				msg.Order.OrderJSON.Routes[0].UnrestrictedValue,
				msg.Order.OrderJSON.Routes[1].UnrestrictedValue,
				msg.Order.OrderJSON.Tariff.TotalPrice,
				msg.Order.OrderJSON.Tariff.Name,
				msg.Order.OrderJSON.Tariff.PaymentType,
			), nil
		} else {
			return proto.GetErrorText(msg.State), errors.New("Tryinig to output route that does not exists")
		}
	case proto.States.Taxi.Order.OrderStart.S():
		return carWillArrive(speechData{
			CarColor:  msg.Order.OrderJSON.Driver.Color,
			CarNumber: msg.Order.OrderJSON.Driver.RegNumber,
			CarBrand:  msg.Order.OrderJSON.Driver.Car,
			ArriveIn:  msg.Order.OrderJSON.ArrivalTime,
		}, template)
	case proto.States.Taxi.Order.OnTheWay.S():
		return fmt.Sprintf(
			template,
			msg.Order.OrderJSON.Routes[1].UnrestrictedValue,
		), nil
	case proto.States.Taxi.Order.FixDeparture.S():
		return fixAddressVariants(template, msg.Order.OrderJSON.Routes[0].UnrestrictedValue, msg.Order.OrderPrefs.DepartureVariants), nil
	case proto.States.Taxi.Order.FixArrival.S():
		return fixAddressVariants(template, msg.Order.OrderJSON.Routes[1].UnrestrictedValue, msg.Order.OrderPrefs.ArrivalVariants), nil
	default:
		return template, nil
	}
}

func fixAddressVariants(template string, originalAddress string, routes []structures.Route) string {
	var variants string
	for i, rt := range routes {
		var num string
		switch i + 1 {
		case 1:
			num = proto.Buttons.Actions.WrongAddress1.T()
		case 2:
			num = proto.Buttons.Actions.WrongAddress2.T()
		case 3:
			num = proto.Buttons.Actions.WrongAddress3.T()
		case 4:
			num = proto.Buttons.Actions.WrongAddress4.T()
		case 5:
			num = proto.Buttons.Actions.WrongAddress5.T()

		}
		variants += fmt.Sprint(num, " ", rt.Value, "\n")
	}
	return fmt.Sprintf(template, originalAddress, variants)
}

type speechData struct {
	CarColor  string
	CarNumber string
	CarBrand  string
	ArriveIn  int64
}

func carWillArrive(sp speechData, template string) (string, error) {

	sp.CarNumber = removeLetters(sp.CarNumber)
	nowTime := time.Now()
	endTime := time.Unix(sp.ArriveIn, 0)

	var res string
	minutes := int(math.Ceil(endTime.Sub(nowTime).Minutes()))
	if minutes < 1 {
		res = fmt.Sprintf(template, 1, sp.CarColor, sp.CarBrand, sp.CarNumber)
	} else {
		// Склоняем числительное
		spellMinutes := spellMinutes(minutes)
		// Формируем строку
		res = fmt.Sprintf(template, spellMinutes, sp.CarColor, sp.CarBrand, sp.CarNumber)
	}
	return res, nil
}

func removeLetters(text string) string {
	var res string
	for i := range text {
		if text[i] >= 48 && text[i] <= 57 {
			res += string(text[i])
		}
	}
	return res
}

func spellMinutes(minutes int) string {
	// Process special cases
	div := minutes % 100
	if div >= 10 && div <= 20 {
		return fmt.Sprintf("%v минут", minutes)
	}

	div = minutes % 10
	if div == 1 {
		return fmt.Sprintf("%v минуту", minutes)
	}
	if div >= 2 && div <= 4 {
		return fmt.Sprintf("%v минуты", minutes)
	}
	return fmt.Sprintf("%v минут", minutes)
}
