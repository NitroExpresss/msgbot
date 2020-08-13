package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/tool"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"math"
	"math/rand"
	"strconv"
	"time"
)

func (h *Handler) NotificateWhenRideWillStart(ctx context.Context, state structures.OfferStates) {

	//skip if not ride staring
	if state.State != variables.OrderStates["OnPlace"] {
		return
	}

	log := logs.Eloger.WithFields(logrus.Fields{
		"event":     "notificate user in whatsapp",
		"orderUUID": state.OrderUUID,
	})

	log.Info("Start sending notification to WhatsApp")

	//msg := "Желаем вам приятной поездки и поздравляем с праздником Победы! ⭐💥🎇 \n --- \nВесь май для вас скидка 7% на все поездки через новое суперприложение, ссылка для скачивания > https://faem.ru/wapp \n\n🚕 заказывайте такси, доставку, а скоро 🛵 еду и продукты \n💳 можете оплачивать картой \n🚗 цену устанавливаете вы \n💬 чат с водителем \n ...и многое другое"

	var startPhrase []string
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. Вас ожидает:")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. К Вам подъехала.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. Авто на месте.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. Авто подъехало.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. Водитель Вас ожидает.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. Вас ожидает водитель.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. водитель подъехал к адресу.")
	startPhrase = append(startPhrase, "Здравствуйте, это служба такси. водитель на месте.")

	var endPhrase []string
	endPhrase = append(endPhrase, "\n\nА вы знаете что через приложение можно заказывать дешевле? \nДарим 100 бонусов за регистрацию (1 бонус = 1 рубль), скачайте по ссылке 👉 faem.ru/wapp ")
	endPhrase = append(endPhrase, "\n\nЖелаем приятной поездки.\nА вы уже скачали наше приложение? Лучшие цены на поездки только в приложении, проверьте 👉 faem.ru/wapp")
	endPhrase = append(endPhrase, "\n\nСкачайте наше новое приложение и получите 100 бонусов (1 бонус = 1 рубль). Скачатей по ссылке и проверьте бонусный баланс 👉 faem.ru/wapp")
	endPhrase = append(endPhrase, "\n\nСкачайте новое приложение и получите 150 бонусов (1 бонус = 1 рубль) на поездки. Ваш промокод: faem-wapp. Установить 👉 faem.ru/wapp")
	endPhrase = append(endPhrase, "\n\nА вы знаете что в приложении цены еще дешевле, а при установке вы получите 100 бонусов (1 бонус = 1 рубль) + 50 рублей по промокоду: faem-wapp. Скачайте по ссылке 👉 faem.ru/wapp")
	endPhrase = append(endPhrase, "\n\nУстановите приложении и полуите 100 бонусов за регистрацию (1 бонус = 1 рубль). Скачайте по ссылке 👉 faem.ru/wapp")
	endPhrase = append(endPhrase, "\n\nСкачате наше приложении и сможете заказывать еду и продукты. Дарим 100 рублей при регистрации. Жмите на ссылку 👉 faem.ru/wapp")

	var feedbackPhrase []string
	feedbackPhrase = append(feedbackPhrase, "\n\nПо окончанию поездки будем признательны за ваш отзыв 🙏 желаем крепкого здоровья, Вам и Вашим близким")
	feedbackPhrase = append(feedbackPhrase, "\n\nБудем рады вашему отзыву 🙏 а Вам и Вашим близким всего самого хорошего")
	feedbackPhrase = append(feedbackPhrase, "\n\nМы хотим быть лучше, помогите своим отзывом по завершению поездки 🙏 будьте здоровы Вы и Ваши близкие")

	rand.Seed(time.Now().Unix())
	n1 := rand.Int() % len(startPhrase)
	n2 := rand.Int() % len(endPhrase)
	n3 := rand.Int() % len(feedbackPhrase)

	taxi, ok := h.Buffers.DriverFounded[state.OrderUUID]
	if !ok {
		log.Error("driver not founded in buffer")
		return
	}

	phrase := fmt.Sprintf("%s %s %s %s", startPhrase[n1], taxi, endPhrase[n2], feedbackPhrase[n3])

	val, err := h.NumberFromBufferByUUID(state.OrderUUID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"reason": "seems like phone is wrong",
		}).Error(err)
		return
	}

	err = h.SendChatApiMsg(val, phrase)
	if err != nil {
		log.Error(err)
	}
	log.Info("Msg sended!")
	delete(h.Buffers.CRMOrders, state.OrderUUID)
	delete(h.Buffers.DriverFounded, state.OrderUUID)
}

func (h *Handler) NumberFromBufferByUUID(uuid string) (string, error) {
	val, ok := h.Buffers.CRMOrders[uuid]
	if !ok {
		return "", errors.New("number not found")
	}

	if len(val) == 0 || len(val) != 12 {
		return "", errors.New("seems like number is wrong")
	}
	return val[1:], nil
}

//SendChatApiMsg send message to WhatsApp
func (h *Handler) SendChatApiMsg(number, msgToSend string) error {

	bucketURL, bucketToken, phone, err := h.getBucket(number)
	url := fmt.Sprintf("%s/sendMessage?token=%s", bucketURL, bucketToken)

	sendMsg := proto.WhAppMsg{
		Phone: phone,
		Body:  msgToSend,
	}

	var resp proto.WhAppMsgResp
	err = tool.SendRequest("POST", url, nil, sendMsg, &resp)
	if err != nil {
		return err
	}

	logs.Eloger.WithFields(logrus.Fields{
		"event":  "send to whatsapp",
		"number": phone,
		"bucket": bucketURL,
	}).Info(msgToSend)

	return nil
}

func (h *Handler) getBucket(number string) (string, string, int, error) {
	phone, err := strconv.Atoi(number)
	if err != nil {
		return "", "", 0, err
	}
	if divisibleBy(phone, 3) {
		return h.Config.ChatApi.URL1, h.Config.ChatApi.Token1, phone, nil
	}
	if divisibleBy(phone, 2) {
		return h.Config.ChatApi.URL2, h.Config.ChatApi.Token2, phone, nil
	}
	return h.Config.ChatApi.URL3, h.Config.ChatApi.Token3, phone, nil
}

func divisibleBy(num int, divisor int) bool {
	if math.Mod(float64(num), float64(divisor)) == 0 {
		return true
	}
	return false
}
