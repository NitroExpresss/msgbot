package repository

import (
	"context"
	"github.com/pkg/errors"
	"gitlab.com/faemproject/backend/faem/services/msgbot/proto"
	"strconv"
	"time"

	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
)

func (p *Pg) GetUser(ctx context.Context, clientMsgID, source string) (models.LocalUsers, error) {
	var users []models.LocalUsers

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := p.Db.ModelContext(ctx, &users).
		Where("source = ? AND client_msg_id = ?", source, clientMsgID).Select()

	if err != nil {
		return models.LocalUsers{}, errors.Wrap(err, "Error getting Users")
	}

	if len(users) == 0 {
		return models.LocalUsers{}, errors.Wrap(err, "User not found")
	}

	return users[0], nil
}

func (p *Pg) SaveUserContact(ctx context.Context, contact proto.MessangerContact, source string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var locUser = models.LocalUsers{
		Phone:                contact.PhoneNumber,
		SourceMsgContactdata: contact,
	}

	_, err := p.Db.ModelContext(ctx, &locUser).
		Set("phone = ?phone").
		Set("source_msg_contactdata = ?source_msg_contactdata").
		Where("client_msg_id = ? AND source = ?", strconv.Itoa(contact.UserID), source).
		Update()
	if err != nil {
		return errors.Wrap(err, "Error updating ")
	}
	return nil
}
