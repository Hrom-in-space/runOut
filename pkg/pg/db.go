package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Config struct {
	User     string
	Password string
	HostPort string
	DBName   string
}

func New(ctx context.Context, conf Config) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(fmt.Sprintf(
		"postgres://%v:%v@%v/%v",
		conf.User, conf.Password, conf.HostPort, conf.DBName,
	))
	if err != nil {
		return nil, fmt.Errorf("error parsing pgxpool config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error creating pgxpool: %w", err)
	}

	return pool, nil
}
