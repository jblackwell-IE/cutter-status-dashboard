package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/IdeaEvolver/cutter-pkg/cuterr"
)

type Health struct {
	db *sql.DB
}

func NewHealthChecker(db *sql.DB) *Health {
	return &Health{db}
}

func (h *Health) Health(ctx context.Context) error {
	var t time.Time
	if err := h.db.QueryRowContext(ctx, "select current_timestamp").Scan(&t); err != nil {
		return cuterr.FromDatabaseError("database healthcheck", err)
	}
	return nil
}
