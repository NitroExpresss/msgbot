package repository

import (
	"context"
	"github.com/go-pg/pg"
	"gitlab.com/faemproject/backend/faem/pkg/variables"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"time"
)

func (p *Pg) GetActiveOrders(ctx context.Context) ([]models.LocalOrders, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var localOrders []models.LocalOrders

	err := p.Db.ModelContext(ctx, &localOrders).
		Where("state in (?)", pg.In(variables.ActiveOrderStatesList())).
		Select()

	if err != nil {
		return localOrders, err
	}

	return localOrders, nil
}
