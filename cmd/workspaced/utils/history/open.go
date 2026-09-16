package history

import (
	"context"

	"github.com/lucasew/workspaced/internal/db"
	"github.com/lucasew/workspaced/pkg/logging"
)

func open(ctx context.Context) (*db.DB, func(), error) {
	if database, ok := db.FromContext(ctx); ok {
		return database, func() {}, nil
	}
	database, err := db.Open(ctx)
	if err != nil {
		return nil, nil, err
	}
	return database, func() { logging.Close(ctx, database) }, nil
}
