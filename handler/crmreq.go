//CRM requests handlers must be here
package handler

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/faemproject/backend/faem/pkg/structures"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"net/http"
)

func (h *Handler) GetCRMAdresses(req string) ([]structures.Route, error) {
	var (
		request struct {
			Address string `json:"name"`
		}
		response []structures.Route
	)
	request.Address = req
	url := fmt.Sprintf("%s/addresses", h.Config.CRMURL)
	err := h.RPC(http.MethodPost, url, request, &response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addresses via rpc")
	}
	return response, nil
}

//GetTariffs возвращает стоимости по все доступным тарифам
func (h *Handler) GetTariffs(order models.OrderCRM) ([]models.ShortTariff, error) {
	// /orders/tariffs
	url := fmt.Sprintf("%s/orders/tariffs", h.Config.CRMURL)
	var response []models.ShortTariff
	err := h.RPC(http.MethodPost, url, order, &response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tariffs via rpc")
	}
	return response, nil
}

//FillTariff заполняет стоимость тарифа
func (h *Handler) FillTariff(order *models.OrderCRM) error {
	// /orders/tariffs
	if order.ServiceUUID == "" {
		return errors.New("Service UUID empty")
	}
	if len(order.Routes) < 2 {
		return errors.New("Need more the 1 route")
	}
	url := fmt.Sprintf("%s/orders/tariff", h.Config.CRMURL)
	var response structures.Tariff

	fmt.Printf("UUID= %s\n", order.ServiceUUID)
	err := h.RPC(http.MethodPost, url, order, &response)
	if err != nil {
		return errors.Wrap(err, "failed to get single tariff via rpc")
	}
	order.Tariff = response

	return nil
}
