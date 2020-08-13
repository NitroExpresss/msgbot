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
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/handler"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
)

const (
	channelNewOrderStates = "newChatMsgForBot"
)

func (s *Subscriber) initOrderStatesSubscription() error {
	orderStatesChannel, err := s.Rabbit.GetReceiver(channelNewOrderStates)
	if err != nil {
		return errors.Wrapf(err, "failed to get a receiver channel")
	}

	// Declare an exchange first
	err = orderStatesChannel.ExchangeDeclare(
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

	queue, err := orderStatesChannel.QueueDeclare(
		rabbit.BotOrderStatesQueue, // name
		true,                       // durable
		false,                      // delete when unused
		false,                      // exclusive
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	err = orderStatesChannel.QueueBind(
		queue.Name,           // queue name
		"state.*",            // routing key
		rabbit.OrderExchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to bind a queue")
	}

	msgs, err := orderStatesChannel.Consume(
		queue.Name,                   // queue
		rabbit.BotOrderStateConsumer, // consumer
		true,                         // auto-ack
		false,                        // exclusive
		false,                        // no-local
		false,                        // no-wait
		nil,                          // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to consume from a channel")
	}

	s.wg.Add(1)
	go s.orderStateHadling(msgs) // handle incoming messages
	return nil
}

func (s *Subscriber) orderStateHadling(messages <-chan amqp.Delivery) {
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
					if err := s.HandleNewOrderState(context.Background(), msg); err != nil {
						logs.Eloger.Errorln(errpath.Err(err, "failed to handle new user"))
					}
				},
			))
		}
	}
}

func (s *Subscriber) HandleNewOrderState(ctx context.Context, msg amqp.Delivery) error {
	// Decode incoming message
	var incomeState structures.OfferStates
	if err := s.Encoder.Decode(msg.Body, &incomeState); err != nil {
		return errors.Wrap(err, "failed to decode new order state")
	}

	//скипаем статус когда он - OrderCreated , потому что мы сами его вызвали
	if incomeState.State == variables.OrderStates["OrderCreated"] {
		return nil
	}

	//обновляем статус заказа
	//s.Handler.UpdateOrderState(ctx, incomeState)

	var localOrder models.LocalOrders

	_, ok := s.Handler.Buffers.WIPOrders[incomeState.OrderUUID]
	if !ok {
		//заказа нет в буфере активных заказов- ливаем
		return nil
	}
	localOrder, err := s.Handler.DB.GetLocalOrderByUUID(ctx, incomeState.OrderUUID)
	if err != nil {
		return errors.Wrap(err, "failed to get last order")
	}

	//получаем объект статуса для перевода в новый статус
	newState := s.Handler.GetStateObjectFromText(incomeState.State)

	//
	msgJob := s.Handler.Jobs.GetJobQueue(handler.JobQueueNameMsgs, handler.JobQueueLimitMsgs)

	err = msgJob.Execute(crypto.FNV(incomeState.OrderUUID), func() error {
		err = s.Handler.BrokerMsgHandler(ctx, &localOrder, newState)
		return err
	})

	return err
}
