package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Tx string

const TxKey = Tx("tx")

func TxToCtx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

func TxFromCtx(ctx context.Context) (pgx.Tx, bool) {
	v := ctx.Value(TxKey)
	if v == nil {
		return nil, false
	}

	tx, ok := v.(pgx.Tx)
	if !ok {
		return nil, false
	}

	return tx, true
}

func MustTxFromCtx(ctx context.Context) pgx.Tx {
	tx, ok := TxFromCtx(ctx)
	if !ok {
		panic("transaction not found in context")
	}

	return tx
}
