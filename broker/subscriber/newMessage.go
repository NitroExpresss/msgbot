package subscriber

import (
	"context"

	"github.com/korovkin/limiter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/faemproject/backend/faem/pkg/lang"
	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
)

const (
	channelNewChatMsg = "newChatMsgForBot"
)

func (s *Subscriber) HandleNewIncomeMsg(ctx context.Context, msg amqp.Delivery) error {
	// Decode incoming message
	var incomeMsg structures.ChatMessages
	if err := s.Encoder.Decode(msg.Body, &incomeMsg); err != nil {
		return errors.Wrap(err, "failed to decode new user")
	}
	// Handle incoming message somehow
	if err := s.Handler.SendToBotClient(ctx, incomeMsg); err != nil {
		return errors.Wrap(err, "failed to handle new income msg")
	}
	return nil
}

//
func (s *Subscriber) newMsgRabbitHandler(messages <-chan amqp.Delivery) {
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
					if err := s.HandleNewIncomeMsg(context.Background(), msg); err != nil {
						logs.Eloger.Errorln(errpath.Err(err, "failed to handle new user"))
					}
				},
			))
		}
	}
}

func (s *Subscriber) initNewMessage() error {
	newMsgChannel, err := s.Rabbit.GetReceiver(channelNewChatMsg)
	if err != nil {
		return errors.Wrapf(err, "failed to get a receiver channel")
	}

	// Declare an exchange first
	err = newMsgChannel.ExchangeDeclare(
		rabbit.FCMExchange, // name
		"topic",            // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to create an exchange")
	}

	queue, err := newMsgChannel.QueueDeclare(
		rabbit.NewMsg2BotQueue, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	err = newMsgChannel.QueueBind(
		queue.Name,                    // queue name
		rabbit.ChatMessageToClientKey, // routing key
		rabbit.FCMExchange,            // exchange
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to bind a queue")
	}

	msgs, err := newMsgChannel.Consume(
		queue.Name,               // queue
		rabbit.BotNewMsgConsumer, // consumer
		true,                     // auto-ack
		false,                    // exclusive
		false,                    // no-local
		false,                    // no-wait
		nil,                      // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to consume from a channel")
	}

	s.wg.Add(1)
	go s.newMsgRabbitHandler(msgs) // handle incoming messages
	return nil
}
