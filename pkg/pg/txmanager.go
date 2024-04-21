package pg

import (
	"context"
	"fmt"

	"runout/pkg/logger"
)

type Manager interface {
	Do(ctx context.Context, callback func(context.Context) error) error
}

var _ = (*TxManager)(nil)

type TxManager struct {
	db DB
}

func NewTxManager(db DB) *TxManager {
	return &TxManager{
		db: db,
	}
}

func (m *TxManager) Do(ctx context.Context, callback func(context.Context) error) error {
	var err error
	log := logger.FromCtx(ctx)

	// получение транзакции из контекста или создание новой
	trx, isTxFromCtx := TxFromCtx(ctx)
	if !isTxFromCtx {
		trx, err = m.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("error starting transaction: %w", err)
		}
	}

	// если транзакция не из контекста, то кладем ее в контекст
	if !isTxFromCtx {
		ctx = TxToCtx(ctx, trx)
	}

	err = callback(ctx)
	if err != nil {
		rollbackErr := trx.Rollback(ctx)
		if rollbackErr != nil {
			log.Error("Error rolling back transaction", logger.Error(rollbackErr))
		}

		return fmt.Errorf("error in callback: %w", err)
	}

	// если транзакция не из контекста, то фиксируем ее
	if !isTxFromCtx {
		return trx.Commit(ctx)
	}
	// иначе ничего не делаем, т.к. транзакция была создана вне этого метода и будет зафиксирована вне этого метода
	return nil
}
