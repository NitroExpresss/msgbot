//Здесь мы отслеживание новые заказы из CRM-ки
package subscriber

import (
	"github.com/korovkin/limiter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/faemproject/backend/faem/pkg/lang"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
)

const (
	channelNewOrders = "newOrdersChannel"
)

//handleNewOrderMsg сохраняет новый заказ
func (s *Subscriber) handleNewOrderMsg(msg amqp.Delivery) error {

	var newOrder structures.Order

	// Если паблишер не crm игнорим
	publisher, ok := msg.Headers["publisher"].(string)
	if !ok || publisher != "crm" {
		return nil
	}

	if err := s.Encoder.Decode(msg.Body, &newOrder); err != nil {
		return errors.Wrap(err, "failed to decode an newOrder request")
	}

	// проверяем на всякий
	if newOrder.Source != "crm" {
		return nil
	}

	//сохраняем в буфер заказ о котором нужно
	s.Handler.Buffers.CRMOrders[newOrder.UUID] = newOrder.CallbackPhone

	return nil
}

func (s *Subscriber) initNewOrders() error {
	autoCallChannel, err := s.Rabbit.GetReceiver(channelNewOrders)
	if err != nil {
		return errors.Wrapf(err, "failed to get a receiver channel %s", channelNewOrders)
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
		rabbit.BotNewOrders, // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	err = autoCallChannel.QueueBind(
		queue.Name, // queue name
		rabbit.NewKey,
		rabbit.OrderExchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to bind [NewOrder] queue")
	}

	msgs, err := autoCallChannel.Consume(
		queue.Name,              // queue
		rabbit.NewOrderConsumer, // consumer
		true,                    // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to consume from a channel")
	}

	s.wg.Add(1)
	go s.handleNewOrders(msgs) // handle incoming messages
	return nil
}

func (s *Subscriber) handleNewOrders(messages <-chan amqp.Delivery) {
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
					if err := s.handleNewOrderMsg(msg); err != nil {
						logs.Eloger.Errorf("Failed to handle New Order request: %v", err)
					}
				},
			))
		}
	}
}
