package status

import (
	"context"
	"database/sql"
	"time"

	"github.com/IdeaEvolver/cutter-pkg/cuterr"
)

type StatusStore struct {
	db *sql.DB
}

type AllStatuses struct {
	StatusId string
	Service  string `json:"service"`
	Status   string `json:"status"`
}

type Status struct {
	Status string `json:"status"`
}

type StatusReport struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func New(db *sql.DB) *StatusStore {
	return &StatusStore{
		db: db,
	}
}

func (s *StatusStore) UpdateStatus(ctx context.Context, service, status string) error {
	var query = `UPDATE statuses SET status = $1 WHERE service = $2`

	_, err := s.db.ExecContext(ctx, query, status, service)
	if err != nil {
		return cuterr.FromDatabaseError("UpdateStatus", err)
	}

	return nil
}

func (s *StatusStore) GetAllStatuses(ctx context.Context) ([]*AllStatuses, error) {
	var query = `SELECT * FROM statuses`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, cuterr.FromDatabaseError("GetAllStatuses", err)
	}
	defer rows.Close()

	ret := []*AllStatuses{}
	for rows.Next() {
		r := &AllStatuses{}
		if err := rows.Scan(
			&r.StatusId,
			&r.Service,
			&r.Status,
		); err != nil {
			return nil, err
		}
		ret = append(ret, r)
	}

	return ret, nil
}

// might be useful to get individual service status

func (s *StatusStore) GetStatus(ctx context.Context, service string) (*Status, error) {
	var query = `SELECT status FROM statuses WHERE service = $1`

	ret := &Status{}
	err := s.db.QueryRowContext(ctx, query, service).
		Scan(
			&ret.Status,
		)

	if err != nil {
		return nil, cuterr.FromDatabaseError("GetStatus", err)
	}

	return ret, nil
}

func (s *StatusStore) UpdateServiceDown(ctx context.Context, service, status string, timestamp time.Time) error {
	var query = `INSERT INTO service_down (service, status, timestamp) VALUES ($1, $2, $3)`

	_, err := s.db.ExecContext(ctx, query, service, status, timestamp)
	if err != nil {
		return cuterr.FromDatabaseError("UpdateStatusDown", err)
	}

	return nil
}

func (s *StatusStore) GetServiceDown(ctx context.Context, service string) ([]*StatusReport, error) {
	var query = `SELECT service, status, timestamp FROM service_down WHERE service = $1`

	rows, err := s.db.QueryContext(ctx, query, service)
	if err != nil {
		return nil, cuterr.FromDatabaseError("GetServiceDown", err)
	}
	defer rows.Close()

	ret := []*StatusReport{}
	for rows.Next() {
		r := &StatusReport{}
		if err := rows.Scan(
			&r.Service,
			&r.Status,
			&r.Timestamp,
		); err != nil {
			return nil, err
		}
		ret = append(ret, r)
	}
	return ret, nil
}
