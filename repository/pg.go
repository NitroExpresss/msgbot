package repository

import (
	"github.com/go-pg/pg"
)

type Pg struct {
	Db *pg.DB
}
