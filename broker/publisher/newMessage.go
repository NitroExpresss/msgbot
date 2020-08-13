package publisher

import (
	"github.com/pkg/errors"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
)

const (
	channelNewChatMsg = "newChatMsg"
)

func (p *Publisher) NewMsg(msg *structures.MessageFromBot) error {
	chatMsgChannel, err := p.Rabbit.GetSender(channelNewChatMsg)
	if err != nil {
		return errors.Wrapf(err, "failed to get a sender channel")
	}
	return p.Publish(chatMsgChannel, rabbit.ChatExchange, rabbit.NewIncomingMsgKey, msg)
}

func (p *Publisher) initNewMsgExchange() error {
	newMsgChannel, err := p.Rabbit.GetSender(channelNewChatMsg)
	if err != nil {
		return errors.Wrapf(err, "failed to get a sender channel")
	}

	err = newMsgChannel.ExchangeDeclare(
		rabbit.ChatExchange, // name
		"topic",             // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	return errors.Wrap(err, "failed to create an exchange")
}
