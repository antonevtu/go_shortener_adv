// Package handlers processes http requests for URL shortening, extension, and deletion
package handlers

import (
	"context"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
)

type Repositorier interface {

	//AddEntity adds new row Entity in DB. If long URL already exists, returns ErrUniqueViolation
	AddEntity(ctx context.Context, entity db.Entity) error

	//SelectByLongURL returns row Entity for known long URL
	SelectByLongURL(ctx context.Context, longURL string) (db.Entity, error)

	//SelectByShortID returns row Entity for known short ID
	SelectByShortID(ctx context.Context, shortURL string) (db.Entity, error)

	//SelectByUser returns all Entity rows for given userID
	SelectByUser(ctx context.Context, userID string) ([]db.Entity, error)

	//AddEntityBatch fast adds BatchInput in transaction mode
	AddEntityBatch(ctx context.Context, userID string, input db.BatchInput) error

	//Ping checks DB connection is alive
	Ping(ctx context.Context) error

	//SetDeletedBatch fast delete several Entities in transaction mode
	//Doesn't remove rows, only sets deleted flags = true
	SetDeletedBatch(ctx context.Context, userID string, shortIDs []string) error

	//SetDeleted delete one row Entity.
	//Doesn't remove row, only sets deleted flag = true
	SetDeleted(ctx context.Context, item pool.ToDeleteItem) error
}
