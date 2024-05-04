package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"runout/internal/domain"
	"runout/pkg/pg"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (n *Repo) ListNeeds(ctx context.Context) ([]domain.Need, error) {
	const query = "SELECT id, name FROM needs ORDER BY name"
	rows, err := pg.MustTxFromCtx(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying needs: %w", err)
	}
	needs, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Need])
	if err != nil {
		return nil, err
	}

	return needs, nil
}

func (n *Repo) AddNeed(ctx context.Context, need string) error {
	const query = "INSERT INTO needs (name) VALUES ($1)"
	_, err := pg.MustTxFromCtx(ctx).Exec(ctx, query, need)
	if err != nil {
		return fmt.Errorf("error adding need: %w", err)
	}

	return nil
}

func (n *Repo) ClearNeeds(ctx context.Context) error {
	const query = "DELETE FROM needs"
	_, err := pg.MustTxFromCtx(ctx).Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error adding need: %w", err)
	}

	return nil
}
