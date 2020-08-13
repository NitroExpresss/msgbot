package publisher

import (
	"github.com/pkg/errors"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
)

const (
	channelNewOrder   = "newDraftOrderChanel"
	channelStartOrder = "newOrderChanel"
)

func (p *Publisher) NewDraftOrder(order *models.OrderCRM) error {
	chatMsgChannel, err := p.Rabbit.GetSender(channelNewOrder)
	if err != nil {
		return errors.Wrapf(err, "failed to get a sender channel")
	}
	return p.Publish(chatMsgChannel, rabbit.OrderExchange, rabbit.NewChatOrder, order)
}

func (p *Publisher) StartOrder(order *models.OrderCRM) error {
	chatMsgChannel, err := p.Rabbit.GetSender(channelStartOrder)
	if err != nil {
		return errpath.Err(err, "failed to get a sender channel")
	}
	err = p.Publish(chatMsgChannel, rabbit.OrderExchange, rabbit.NewKey, order)
	if err != nil {
		return errpath.Err(err, "publish failed")
	}
	return nil
}

func (p *Publisher) initNewOrderExchange() error {
	newMsgChannel, err := p.Rabbit.GetSender(channelNewOrder)
	if err != nil {
		return errors.Wrapf(err, "failed to get a sender channel")
	}

	err = newMsgChannel.ExchangeDeclare(
		rabbit.OrderExchange, // name
		"topic",              // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	return errors.Wrap(err, "failed to create an exchange")
}
