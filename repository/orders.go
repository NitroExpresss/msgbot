package repository

import (
	"context"
	"github.com/go-pg/pg"
	"gitlab.com/faemproject/backend/faem/pkg/variables"

	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

func (p *Pg) SaveDriverData(ctx context.Context, order structures.Order) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var localOrder models.LocalOrders

	updatedOrder := models.OrderCRM{
		Order:       order,
		ServiceUUID: order.Service.UUID,
	}

	_, err := p.Db.ModelContext(ctx, &localOrder).
		Set("order_json = ?", updatedOrder).
		Where("order_uuid = ?", order.UUID).
		Returning("*").
		Update()

	if err != nil {
		return localOrder, err
	}
	return localOrder, nil
}

func (p *Pg) SetOrderState(ctx context.Context, newState structures.OfferStates) (models.LocalOrders, error) {
	var order models.LocalOrders
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := p.Db.ModelContext(ctx, &order).
		Set("state = ?", newState.State).
		Set("updated_at = ?", time.Now()).
		Where("order_uuid = ?", newState.OrderUUID).
		Returning("*").
		Update()

	if err != nil {
		return models.LocalOrders{}, err
	}
	return order, nil
}

//SaveOrderRoute - refactored
//TODO в самом конце можно убрать proto.Consts.Order.SetRoute.Arrival из case
func (p *Pg) SaveOrderRoute(ctx context.Context, orderUUID string, routeType proto.Constant, route structures.Route) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var localOrders []models.LocalOrders

	err := p.Db.ModelContext(ctx, &localOrders).
		Where("order_uuid = ?", orderUUID).Select()
	if err != nil {
		return models.LocalOrders{}, errors.Wrap(err, "Error getting Order")
	}
	if len(localOrders) == 0 {
		return models.LocalOrders{}, errors.Wrap(err, "Local order not found")
	}
	order := localOrders[0]
	orderRoute := order.OrderJSON.Routes

	switch routeType {
	case proto.Consts.Order.SetRoute.Departure, proto.System.RouteTypes.Departure:
		if len(orderRoute) == 0 {
			order.OrderJSON.Routes = append(order.OrderJSON.Routes, route)
		} else {
			order.OrderJSON.Routes[0] = route
		}
	case proto.Consts.Order.SetRoute.Arrival, proto.System.RouteTypes.Arrival:
		if len(orderRoute) == 0 {
			order.OrderJSON.Routes = append(order.OrderJSON.Routes, structures.Route{}, route)
		} else if len(orderRoute) == 1 {
			order.OrderJSON.Routes = append(order.OrderJSON.Routes, route)
		} else {
			order.OrderJSON.Routes[1] = route
		}
	}

	_, err = p.Db.ModelContext(ctx, &order).Column("order_json").
		Where("order_uuid = ?", orderUUID).Returning("*").Update()

	if err != nil {
		return models.LocalOrders{}, errors.Wrap(err, "Error updating Order")
	}

	return order, nil
}

func (p *Pg) UpdateOrderRoute(ctx context.Context, orderUUID string, routeNumber proto.Constant, route structures.Route) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var localOrders []models.LocalOrders
	var order models.LocalOrders

	err := p.Db.ModelContext(ctx, &localOrders).
		Where("order_uuid = ?", orderUUID).Select()
	if err != nil {
		return order, errpath.Err(err, "Error getting Order")
	}
	if len(localOrders) == 0 {
		return order, errpath.Err(err, "Local order not found")
	}
	order = localOrders[0]

	update := func() error {
		_, err = p.Db.ModelContext(ctx, &order).Column("order_json").
			Where("order_uuid = ?", orderUUID).Returning("*").Update()
		return err
	}

	if routeNumber == proto.Consts.BotSend.ButtonsActions.SetDepartureAdress {
		if len(order.OrderJSON.Routes) < 1 {
			return order, errpath.Errorf("нельзя обновить не существующий адрес подачи")
		}
		order.OrderJSON.Order.Routes[0] = route
		err = update()
		if err != nil {
			return models.LocalOrders{}, errpath.Err(err, "Error updating Order")
		}
	}
	if routeNumber == proto.Consts.BotSend.ButtonsActions.SetArrivalAdress {
		if len(order.OrderJSON.Routes) < 2 {
			return order, errpath.Errorf("нельзя обновить не существующий адрес назначения")
		}
		order.OrderJSON.Order.Routes[1] = route
		err = update()
		if err != nil {
			return models.LocalOrders{}, errpath.Err(err, "Error updating Order")
		}
	}

	return order, nil
}

func (p *Pg) GetCurrentOrder(ctx context.Context, user, source string) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var curOrders []models.LocalOrders
	var crmOrderStates []string
	//crmOrderStates = append(crmOrderStates, variables.ActiveOrderStatesList()...)
	crmOrderStates = append(crmOrderStates, variables.InactiveOrderStatesList()...)
	//crmOrderStates = append(crmOrderStates, variables.OrderStates["OrderCreated"])

	err := p.Db.ModelContext(ctx, &curOrders).
		Where("client_msg_id = ? AND source = ?", user, source).
		Where("state not in (?)", pg.In(crmOrderStates)).
		Order("created_at DESC").
		Select()
	if err != nil {
		return models.LocalOrders{}, err
	}
	if len(curOrders) == 0 {
		return models.LocalOrders{}, nil // errpath.Errorf("order not found")
	}
	return curOrders[0], nil // TODO:? почему не сортируется по desc? // для того чтобы понимать какая ошибка // переделать с проверкой pg-шной ошибки
}

func (p *Pg) GetLastOrder(ctx context.Context, user, source string) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var curOrders []models.LocalOrders

	err := p.Db.ModelContext(ctx, &curOrders).
		Where("client_msg_id = ? AND source = ?", user, source).
		Order("created_at DESC").
		Select()
	if err != nil {
		return models.LocalOrders{}, errpath.Err(err)
	}
	if len(curOrders) == 0 {
		return models.LocalOrders{}, nil // errpath.Errorf("order not found")
	}
	return curOrders[0], nil
}

func (p *Pg) SaveLocalOrder(ctx context.Context, order *models.LocalOrders) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := p.Db.ModelContext(ctx, order).
		OnConflict("(order_uuid) DO UPDATE").Insert()
	if err != nil {
		return err
	}

	return nil
}

func (p *Pg) GetLocalUser(ctx context.Context, userMsgId, source string) (models.LocalUsers, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	locUser := models.LocalUsers{
		ClientMsgID: userMsgId,
		Source:      source,
	}

	_, err := p.Db.ModelContext(ctx, &locUser).
		Where("client_msg_id = ?client_msg_id AND source = ?source").
		OnConflict("DO NOTHING").
		SelectOrInsert()

	if err != nil {
		return models.LocalUsers{}, err
	}

	return locUser, nil
}

func (p *Pg) GetLocalOrderByUUID(ctx context.Context, orderUUID string) (models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var locOrder []models.LocalOrders

	err := p.Db.ModelContext(ctx, &locOrder).
		Where("order_uuid = ?", orderUUID).Select()
	if err != nil {
		return models.LocalOrders{}, err
	}
	if len(locOrder) == 0 {
		return models.LocalOrders{}, errors.New("Chat not found")
	}

	return locOrder[0], nil
}
