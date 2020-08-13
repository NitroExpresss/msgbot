package handler

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/structures/errpath"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

//MsgBotRepository тут события с БД
type MsgBotRepository interface {
	//GetCurrentOrder возращает текущий заказ (чат) для пользователя в заданнои канале, "" если его нет
	GetCurrentOrder(ctx context.Context, user_id, source string) (models.LocalOrders, error)
	// GetLastOrder - получение последнего заказа по id
	GetLastOrder(ctx context.Context, user_id, source string) (models.LocalOrders, error)
	//GetLocalUser возвращает локального юзера
	GetLocalUser(ctx context.Context, user_id, source string) (models.LocalUsers, error)
	//Сохраняет локальный заказ в БД
	SaveLocalOrder(ctx context.Context, order *models.LocalOrders) error
	//Получаем ID chatа для исходящих сообщений
	GetLocalOrderByUUID(ctx context.Context, orderUUID string) (models.LocalOrders, error)
	//Сохраняем роуты в БД, и если есть оба роута то считаем тарифы
	SaveOrderRoute(ctx context.Context, orderUUID string, routeType proto.Constant, route structures.Route) (models.LocalOrders, error)
	// UpdateOrderRoute - routeNumber["set_departure_address","set_arrival_address"]
	UpdateOrderRoute(ctx context.Context, orderUUID string, routeNumber proto.Constant, route structures.Route) (models.LocalOrders, error)
	//Получаем пользователя по MsgId
	GetUser(ctx context.Context, clientMsgID, source string) (models.LocalUsers, error)
	//Сохраняем данные клиента
	SaveUserContact(ctx context.Context, contact proto.MessangerContact, source string) error
	//Сохраняем статус заказа и возвращаем экземпляр
	SetOrderState(ctx context.Context, newState structures.OfferStates) (models.LocalOrders, error)
	//Инициализируем буферизованные данные
	GetActiveOrders(ctx context.Context) ([]models.LocalOrders, error)
	//Инициализируем буферизованные данные
	SaveDriverData(ctx context.Context, order structures.Order) (models.LocalOrders, error)
}

//GetMsgOrder возвращает заказ для текущего сообщения
func (h *Handler) GetMsgOrder(ctx context.Context, chatMsg structures.MessageFromBot) (models.LocalOrders, error) {

	// Проверяем есть ли чат для данного пользователя. В нашем случае chatUUID = orderUUID
	currentOrder, err := h.DB.GetCurrentOrder(ctx, chatMsg.ClientMsgID, chatMsg.Source)
	if err != nil {
		return models.LocalOrders{}, errpath.Err(err, "Error getting current order")
	}
	if currentOrder.OrderUUID != "" {
		return currentOrder, nil
	}
	// Проверяем является ли последний заказ завершенным
	lastOrder, err := h.DB.GetLastOrder(ctx, chatMsg.ClientMsgID, chatMsg.Source)
	if err != nil {
		return models.LocalOrders{}, errpath.Err(err, "Error getting last order")
	}
	if lastOrder.OrderUUID != "" {
		if !variables.InactiveOrderStates(lastOrder.State) {
			log.Warnln(errpath.Errorf("новый заказ не создан т.к. текущий не является завершенным"))
			return currentOrder, nil
		}
	}
	//Если нет, то создаем новый заказ

	//Для начало надо понять есть ли локальный пользователь для этого userID
	localUser, err := h.DB.GetLocalUser(ctx, chatMsg.ClientMsgID, "telegram")
	if err != nil {
		return currentOrder, errors.Wrap(err, "Error getting local user")
	}

	chatID := strconv.FormatInt(chatMsg.ChatMsgID, 10)
	//Создаем и сохраняем новый заказ
	localOrder := models.NewTelegramOrder(localUser, chatID)
	err = h.DB.SaveLocalOrder(ctx, &localOrder)
	if err != nil {
		return localOrder, errors.Wrap(err, "Error saving local user")
	}

	//Отправляем этот заказ по рэбиту
	//err = h.Pub.NewDraftOrder(&localOrder.OrderJSON)
	//if err != nil {
	//	return localOrder, err
	//}

	return localOrder, nil
}
