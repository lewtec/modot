package history

import (
	"context"

	"github.com/lucasew/workspaced/internal/db"
	"github.com/lucasew/workspaced/pkg/logging"
)

func open(ctx context.Context) (*db.DB, func(), error) {
	_, borrowed := db.FromContext(ctx)
	database, err := db.OpenFromCtx(ctx)
	if err != nil {
		return nil, nil, err
	}
	if borrowed {
		return database, func() {}, nil
	}
	return database, func() { logging.Close(ctx, database) }, nil
}
