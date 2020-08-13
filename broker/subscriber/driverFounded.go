package subscriber

import (
	"context"
	"github.com/korovkin/limiter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/faemproject/backend/faem/pkg/crypto"
	"gitlab.com/faemproject/backend/faem/pkg/lang"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/handler"
)

const (
	channelDriverFounded = "driverFoundedChannel"
	//maxOrderStateChangesAllowed = 10
)

var (
	arrivePhraseNoTime, arrivePhrase string
)

//func (s *Subscriber) handleDriverFoundedMsg(ctx context.Context, msg amqp.Delivery) error {
//	var newOrder structures.Order
//
//	log := logs.Eloger.WithFields(logrus.Fields{
//		"event":      "handling driver found message",
//		"orderUUID":  newOrder.UUID,
//		"driverUUID": newOrder.Driver.UUID,
//	})
//	if err := s.Encoder.Decode(msg.Body, &newOrder); err != nil {
//		return errors.Wrap(err, "failed to decode an Driver found request")
//	}
//
//	// Создан ли этот заказ вне этого сервиса
//	if newOrder.Source != variables.OrderSources["Telegram"] && newOrder.Source != variables.OrderSources["Whatsapp"] {
//		//log.Warnln("Заказ вне этого сервиса Telegram или Whatsapp")
//
//		if newOrder.Source == "crm" {
//			s.Handler.Buffers.DriverFounded[newOrder.UUID] = fmt.Sprintf("%s, цвет - %s, номер %s", newOrder.Driver.Car, newOrder.Driver.Color, newOrder.Driver.RegNumber)
//		}
//		//	sText, _ := carWillArrive(speechData{
//		//		CarColor:  newOrder.Driver.Color,
//		//		CarNumber: newOrder.Driver.RegNumber,
//		//		CarBrand:  newOrder.Driver.Car,
//		//		ArriveIn:  newOrder.ArrivalTime,
//		//	})
//		//	val, err := s.Handler.NumberFromBufferByUUID(newOrder.UUID)
//		//	if err != nil {
//		//		return errors.Wrap(err, "error getting number")
//		//	}
//		//	err = s.Handler.SendChatApiMsg(val, sText)
//		//	if err != nil {
//		//		return errors.Wrap(err, "cant send msg to whatsapp")
//		//	}
//		return nil
//	}
//
//	// Проверяем есть ли тег о назначении водителя
//	state, ok := msg.Headers["tag"].(string)
//	if !ok || state != "offer_accepted" {
//		return nil
//	}
//
//	log.Debug("Saving to DB...")
//
//	locOrder, err := s.Handler.DB.SaveDriverData(ctx, newOrder)
//	if err != nil {
//		return errors.Wrap(err, "failed to Save Driver Data from CRM")
//	}
//	log.Info("Driver data updated...")
//
//	arrivePhrase = handler.WillArrivePhrase
//	arrivePhraseNoTime = handler.WillArrivePhraseNoTime
//
//	sayText, _ := carWillArrive(speechData{
//		CarColor:  newOrder.Driver.Color,
//		CarNumber: newOrder.Driver.RegNumber,
//		CarBrand:  newOrder.Driver.Car,
//		ArriveIn:  newOrder.ArrivalTime,
//	})
//
//	chatID, err := strconv.ParseInt(locOrder.ChatMsgId, 10, 64)
//	if err != nil {
//		return errors.Wrap(err, "Cant convert chat id to int64")
//	}
//
//	err = s.Handler.Telegram.SendMessage(chatID, sayText)
//	if err != nil {
//		return errors.Wrap(err, "Cant send message to Telegram client")
//	}
//
//	return nil
//}
//
//type speechData struct {
//	CarColor  string
//	CarNumber string
//	CarBrand  string
//	ArriveIn  int64
//}
//
//func carWillArrive(sp speechData) (string, error) {
//
//	sp.CarNumber = removeLetters(sp.CarNumber)
//	nowTime := time.Now()
//	endTime := time.Unix(sp.ArriveIn, 0)
//
//	var res string
//	minutes := int(math.Ceil(endTime.Sub(nowTime).Minutes()))
//	if minutes < 1 {
//		res = fmt.Sprintf(string(proto.Consts.BotSend.Answers.ForStates.CarFoundedMsg), 1, sp.CarColor, sp.CarBrand, sp.CarNumber)
//	} else {
//		// Склоняем числительное
//		spellMinutes := spellMinutes(minutes)
//		// Формируем строку
//		res = fmt.Sprintf(string(proto.Consts.BotSend.Answers.ForStates.CarFoundedMsg), spellMinutes, sp.CarColor, sp.CarBrand, sp.CarNumber)
//	}
//	return res, nil
//}
//
//func removeLetters(text string) string {
//	var res string
//	for i := range text {
//		if text[i] >= 48 && text[i] <= 57 {
//			res += string(text[i])
//		}
//	}
//	return res
//}
//
//func spellMinutes(minutes int) string {
//	// Process special cases
//	div := minutes % 100
//	if div >= 10 && div <= 20 {
//		return fmt.Sprintf("%v минут", minutes)
//	}
//
//	div = minutes % 10
//	if div == 1 {
//		return fmt.Sprintf("%v минуту", minutes)
//	}
//	if div >= 2 && div <= 4 {
//		return fmt.Sprintf("%v минуты", minutes)
//	}
//	return fmt.Sprintf("%v минут", minutes)
//}

func (s *Subscriber) initDriverFounded() error {
	autoCallChannel, err := s.Rabbit.GetReceiver(channelDriverFounded)
	if err != nil {
		return errors.Wrapf(err, "failed to get a receiver channel %s", channelDriverFounded)
	}

	// Declare an exchange first
	err = autoCallChannel.ExchangeDeclare(
		rabbit.OrderExchange, // name
		"topic",              // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to create an exchange")
	}

	queue, err := autoCallChannel.QueueDeclare(
		rabbit.BotDriverFoundedQueue, // name
		true,                         // durable
		false,                        // delete when unused
		false,                        // exclusive
		false,                        // no-wait
		nil,                          // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	err = autoCallChannel.QueueBind(
		queue.Name, // queue name
		rabbit.OrderUpdateFromCRMKey,
		rabbit.OrderExchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to bind [DriverAccepted] queue")
	}

	msgs, err := autoCallChannel.Consume(
		queue.Name,                      // queue
		rabbit.BotDriverFoundedConsumer, // consumer
		true,                            // auto-ack
		false,                           // exclusive
		false,                           // no-local
		false,                           // no-wait
		nil,                             // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to consume from a channel")
	}

	s.wg.Add(1)
	go s.handleDriverFounded(msgs) // handle incoming messages

	return nil
}

func (s *Subscriber) handleDriverFounded(messages <-chan amqp.Delivery) {
	defer s.wg.Done()

	limit := limiter.NewConcurrencyLimiter(maxNewUsersAllowed)
	defer limit.Wait()

	for {
		select {
		case <-s.closed:
			return
		case msg := <-messages:
			// Start new goroutine to handle multiple requests at the same time
			limit.Execute(lang.Recover(
				func() {
					//if err := s.handleDriverFoundedMsg(context.Background(), msg); err != nil {
					//	logs.Eloger.Errorf("Failed to handle driver found request: %v", err)
					//}

					if err := s.handleNewDriverFoundedMsg(context.Background(), msg); err != nil {
						logs.Eloger.Errorf("Failed to handle driver found request: %v", err)
					}

				},
			))
		}
	}
}

func (s *Subscriber) handleNewDriverFoundedMsg(ctx context.Context, msg amqp.Delivery) error {
	var newOrder structures.Order

	if err := s.Encoder.Decode(msg.Body, &newOrder); err != nil {
		return errors.Wrap(err, "failed to decode an Driver found request")
	}

	// Создан ли этот заказ вне этого сервиса
	if newOrder.Source != variables.OrderSources["Telegram"] && newOrder.Source != variables.OrderSources["Whatsapp"] {
		return nil
	}

	// Проверяем есть ли тег о назначении водителя
	state, ok := msg.Headers["tag"].(string)
	if !ok || state != "offer_accepted" {
		return nil
	}

	//сохраняем данные о назначениии водителя, - конструкция является некой очередью
	msgJob := s.Handler.Jobs.GetJobQueue(handler.JobQueueNameMsgs, handler.JobQueueLimitMsgs)
	err := msgJob.Execute(crypto.FNV(newOrder.Driver.UUID), func() error {
		_, err := s.Handler.DB.SaveDriverData(ctx, newOrder)
		//err = s.Handler.BrokerMsgHandler(ctx, &locOrder, proto.States.Taxi.Order.DriverFounded)
		return err
	})

	if err != nil {
		return errors.Wrap(err, "failed to Save Driver Data from CRM")
	}
	return err
}
