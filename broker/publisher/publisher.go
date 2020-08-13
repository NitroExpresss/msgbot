package publisher

import (
	"fmt"
	"go.uber.org/multierr"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/services/msgbot/broker"
)

const (
	headerPublisher = "msgbot"
)

type Publisher struct {
	Rabbit  *rabbit.Rabbit
	Encoder broker.Encoder

	userCreatedChannel *rabbit.Channel
	wg                 sync.WaitGroup
}

func (p *Publisher) Init() error {
	// call all the initializers here, multierr package might be useful
	return multierr.Append(
		p.initNewMsgExchange(),
		p.initNewOrderExchange(),
	)
}

func (p *Publisher) Wait(shutdownTimeout time.Duration) {
	// try to shutdown the listener gracefully
	stoppedGracefully := make(chan struct{}, 1)
	go func() {
		p.wg.Wait()
		stoppedGracefully <- struct{}{}
	}()

	// wait for a graceful shutdown and then stop forcibly
	select {
	case <-stoppedGracefully:
		logs.Eloger.Info("publisher stopped gracefully")
	case <-time.After(shutdownTimeout):
		logs.Eloger.Info("publisher stopped forcibly")
	}
}

func (p *Publisher) Publish(channel *rabbit.Channel, exchange, routingKey string, payload interface{}) error {
	p.wg.Add(1)
	defer p.wg.Done()

	headers := make(amqp.Table)
	headers["publisher"] = headerPublisher

	body, err := p.Encoder.Encode(payload)
	if err != nil {
		return errors.Wrap(err, "failed to encode the message")
	}

	err = channel.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Headers:      headers,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to send a message, exchange = %s, routing key = %s", exchange, routingKey)
	}

	logs.Eloger.WithFields(logrus.Fields{
		"event": "Send JSON to RabbitMQ",
		"value": fmt.Sprintf("exchange = %s; key = %s", exchange, routingKey),
	}).Debug("Sent to RabbitMQ")
	return nil
}
