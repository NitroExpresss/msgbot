package subscriber

import (
	"context"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/korovkin/limiter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/faemproject/backend/faem/pkg/lang"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

const channelOrderUpdate = "orderUpdateChannel"

func (s *Subscriber) initOrderUpdate() error {
	autoCallChannel, err := s.Rabbit.GetReceiver(channelOrderUpdate)
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
		rabbit.BotOrderUpdate, // name
		true,                  // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	err = autoCallChannel.QueueBind(
		queue.Name, // queue name
		rabbit.UpdateKey,
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
	go s.handleOrderUpdate(msgs) // handle incoming messages

	return nil
}

func (s *Subscriber) handleOrderUpdate(messages <-chan amqp.Delivery) {
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
					if err := s.handleOrderUpdateMsg(context.Background(), msg); err != nil {
						logs.Eloger.Errorf("Failed to handle driver found request: %v", err)
					}
				},
			))
		}
	}
}

func (s *Subscriber) handleOrderUpdateMsg(ctx context.Context, msg amqp.Delivery) error {
	var newOrder structures.Order

	//log := logs.Eloger.WithFields(logrus.Fields{
	//	"event":      "handling driver found message",
	//	"orderUUID":  newOrder.UUID,
	//	"driverUUID": newOrder.Driver.UUID,
	//})
	if err := s.Encoder.Decode(msg.Body, &newOrder); err != nil {
		return errors.Wrap(err, "failed to decode an Driver found request")
	}

	//// Создан ли этот заказ вне этого сервиса
	//if newOrder.Source != variables.OrderSources["Telegram"] && newOrder.Source != variables.OrderSources["Whatsapp"] {
	//	log.Warnln("заказ вне этого сервиса Telegram или Whatsapp")
	//	return nil
	//}

	// Проверяем есть ли тег о назначении водителя
	state, ok := msg.Headers["tag"].(string)
	if !ok || state != "offer_accepted" {
		return nil
	}

	// сменяется статус на выпадение кнопок с сервисами-услугами
	locOrder, err := s.Handler.DB.SetOrderState(ctx, structures.OfferStates{OrderUUID: newOrder.UUID, State: string(proto.Consts.Order.CreationStates.ServiceChoice)})
	if err != nil {
		return errpath.Err(err)
	}

	// апдейтим заказ (роуты введенные оператором)
	if len(newOrder.Routes) > 0 {
		_, err = s.Handler.DB.SaveOrderRoute(ctx, newOrder.UUID, proto.Consts.Order.SetRoute.Departure, newOrder.Routes[0])
		if err != nil {
			return errpath.Err(err)
		}
	}
	if len(newOrder.Routes) > 1 {
		_, err = s.Handler.DB.SaveOrderRoute(ctx, newOrder.UUID, proto.Consts.Order.SetRoute.Arrival, newOrder.Routes[1])
		if err != nil {
			return errpath.Err(err)
		}
	}

	// инициировать вывод сообщения с кнопками
	chatID, err := strconv.ParseInt(locOrder.ChatMsgId, 10, 64)
	if err != nil {
		return errors.Wrap(err, "Cant convert chat id to int64")
	}
	_, err = s.Handler.Telegram.SendMessage(chatID, string(proto.Consts.BotSend.Answers.MsgForCreatingWithOperator))
	if err != nil {
		return errors.Wrap(err, "Cant send message to Telegram client")
	}

	emptyMsg := tgbotapi.Message{}
	emptyMsg.Chat = &tgbotapi.Chat{ID: chatID}
	emptyMsg.From = &tgbotapi.User{ID: int(chatID)}
	emptyMsg.Text = string(proto.Consts.Intents.Skip)
	s.Handler.HandleNewTelegramMsg(context.Background(), &emptyMsg)

	return nil
}
