package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
)

type LocalUsers struct {
	tableName            struct{}               `sql:"chat_users"`
	ID                   int                    `json:"id"`
	ClientMsgID          string                 `json:"client_msg_id"`
	Source               string                 `json:"source"`
	Phone                string                 `json:"phone"`
	CrmUUID              string                 `json:"crm_uuid"`
	SourceMsgContactdata proto.MessangerContact `json:"source_msg_contactdata"` //контактные данные прямо из мессенджера
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

type LocalOrders struct {
	tableName   struct{}         `sql:"chat_orders"`
	ID          int              `json:"id"`
	OrderUUID   string           `json:"order_uuid"`
	OrderJSON   OrderCRM         `json:"order_json"`
	State       string           `json:"state"`
	ClientMsgID string           `json:"client_msg_id"` // ChatMsgId(значение от)
	ClientID    int              `json:"client_id"`
	Source      string           `json:"source"`
	ChatMsgId   string           `json:"chat_msg_id"` // id чата (переписки)
	OrderPrefs  OrderPreferences `json:"order_prefs"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type OrderPreferences struct {
	TelegramOrderMsgID int `json:"tlgrm_order_msg_id"` //ID сообщения с данными о заказе
	//в некоторых случаях использую эту структуру что бы передавать из брокера данные
	//в нужный нам handler
	NewState          string              `json:"new_state"`
	DepartureVariants []structures.Route  `json:"departure_variants"`
	FixAddressVariant int                 `json:"fix_address_variant"` //когда отработался коллбек сюда запишем результат
	ArrivalVariants   []structures.Route  `json:"arrival_variants"`    //варианты назначения
	MsgsIDs           map[string]int      `json:"msgs_ids"`            //мапа в которой храним IDшники сообщений определенных статусов
	Tariffs           []proto.TariffProto `json:"tariffs"`             //при смене адреса результаты тарифа пишем сюда
	Service           string              `json:"service"`             //если пользователем выбран сервис
	Coordinates       Coordinates         `json:"coordinates"`
}

type OrderCRM struct {
	structures.Order
	ServiceUUID string `json:"service_uuid"`
}

func NewTelegramOrder(user LocalUsers, chatId string) LocalOrders {
	uuid, _ := uuid.NewV4()
	var locOrder = LocalOrders{
		Source:      string(proto.Consts.MsgSources.Telegram),
		OrderUUID:   uuid.String(),
		State:       string(proto.States.Welcome),
		ClientID:    user.ID,
		ClientMsgID: user.ClientMsgID,
		ChatMsgId:   chatId,
	}
	locOrder.OrderJSON = newGlobalOrder(locOrder.OrderUUID, &user)

	return locOrder
}

func newGlobalOrder(uuid string, user *LocalUsers) OrderCRM {
	var newOrder OrderCRM
	newOrder.UUID = uuid
	newOrder.Source = "msgbot"
	newOrder.CallbackPhone = user.Phone
	com := "Заказ инициирован сервисом чат-бота"
	newOrder.Comment = &com
	newOrder.Client.TelegramID = user.ClientMsgID
	newOrder.Client.MainPhone = user.Phone
	newOrder.Client.UUID = user.CrmUUID
	newOrder.Owner.Name = "default"
	newOrder.CreatedAt = time.Now()
	newOrder.Source = variables.OrderSources["Telegram"]

	return newOrder
}

// ShortTariff short tariff data from CRM
type ShortTariff struct {
	proto.TariffProto
}
