package publisher

import (
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
)

// ActionOnOrderChannel -
const ActionOnOrderChannel = "ActionOnOrderChannel"

// ActionOnOrder -
func (p *Publisher) ActionOnOrder(action *structures.ActionOnOrder) error {

	actionOnOrderChannel, err := p.Rabbit.GetSender(ActionOnOrderChannel)
	if err != nil {
		return errpath.Err(err, "failed to get a sender channel")
	}

	err = p.Publish(actionOnOrderChannel, rabbit.OrderExchange, rabbit.ActionOnOrderKey, action)
	if err != nil {
		return errpath.Err(err, "failed to get a sender channel")
	}
	return nil
}
