//nolint:varnamelen
package pg_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"runout/pkg/pg"
	pg_mocks "runout/pkg/pg/mocks"
)

type CreaterService struct {
	trm pg.Manager
}

func (s *CreaterService) Create(ctx context.Context) error {
	return s.trm.Do(ctx, func(ctx context.Context) error {
		_, err := pg.MustTxFromCtx(ctx).Exec(ctx, "INSERT")
		return err
	})
}

// TestTxManagerDo проверяет работу менеджера транзакций
// в контексте использования внутри сервиса.
func TestTxManagerDo(t *testing.T) {
	t.Parallel()
	clearCtx := context.Background()
	tx := pg_mocks.NewMockTx(t)
	tx.EXPECT().Exec(mock.Anything, "INSERT").RunAndReturn(
		func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			_, ok := pg.TxFromCtx(ctx)
			assert.True(t, ok)
			return pgconn.NewCommandTag(""), nil
		},
	)
	tx.EXPECT().Commit(mock.Anything).Return(nil)
	db := pg_mocks.NewMockDB(t)
	db.EXPECT().Begin(clearCtx).Return(tx, nil)
	trm := pg.NewTxManager(db)
	service := &CreaterService{trm: trm}

	err := service.Create(clearCtx)

	assert.Len(t, tx.Calls, 2)
	assert.Len(t, db.Calls, 1)
	require.NoError(t, err)
}

// TestTxManagerDoWithError проверяет работу менеджера транзакций
// в контексте использования внутри сервиса с ошибкой.
func TestTxManagerDoWithError(t *testing.T) {
	t.Parallel()
	clearCtx := context.Background()
	errTest := errors.New("test")

	tx := pg_mocks.NewMockTx(t)
	tx.EXPECT().Exec(mock.Anything, "INSERT").RunAndReturn(
		func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			_, ok := pg.TxFromCtx(ctx)
			assert.True(t, ok)
			return pgconn.NewCommandTag(""), errTest
		},
	)
	tx.EXPECT().Rollback(mock.Anything).Return(nil)
	db := pg_mocks.NewMockDB(t)
	db.EXPECT().Begin(clearCtx).Return(tx, nil)
	trm := pg.NewTxManager(db)
	service := &CreaterService{trm: trm}

	err := service.Create(clearCtx)

	assert.Len(t, tx.Calls, 2)
	assert.Len(t, db.Calls, 1)
	require.ErrorIs(t, err, errTest)
}

type DeleterService struct {
	trm pg.Manager
}

func (s *DeleterService) Delete(ctx context.Context) error {
	return s.trm.Do(ctx, func(ctx context.Context) error {
		_, err := pg.MustTxFromCtx(ctx).Exec(ctx, "DELETE")
		return err
	})
}

type BlinkerUseCase struct {
	trm        pg.Manager
	createrSvc CreaterService
	deleterSvc DeleterService
}

func (s *BlinkerUseCase) Blink(ctx context.Context) error {
	return s.trm.Do(ctx, func(ctx context.Context) error {
		err := s.createrSvc.Create(ctx)
		if err != nil {
			return fmt.Errorf("error in create service: %w", err)
		}
		err = s.deleterSvc.Delete(ctx)
		if err != nil {
			return fmt.Errorf("error in delete service: %w", err)
		}
		return nil
	})
}

// TestTxManagerDoUseCase проверяет работу менеджера транзакций
// в контексте использования внутри UseCase
// тоесть вызов менеджера транзакций внутри другого менеджера транзакций.
func TestTxManagerDoUseCase(t *testing.T) {
	t.Parallel()
	clearCtx := context.Background()

	// ожидаем что будут вызваны оба сервиса
	tx := pg_mocks.NewMockTx(t)
	tx.EXPECT().Exec(mock.Anything, "INSERT").RunAndReturn(
		func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			_, ok := pg.TxFromCtx(ctx)
			assert.True(t, ok)
			return pgconn.NewCommandTag(""), nil
		},
	)
	tx.EXPECT().Exec(mock.Anything, "DELETE").RunAndReturn(
		func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			_, ok := pg.TxFromCtx(ctx)
			assert.True(t, ok)
			return pgconn.NewCommandTag(""), nil
		},
	)
	tx.EXPECT().Commit(mock.Anything).Return(nil)

	// эта проверка гарантирует что транзакции начнутся в самом верхнем выхове менеджера
	dbMain := pg_mocks.NewMockDB(t)
	dbMain.EXPECT().Begin(clearCtx).Return(tx, nil)
	trmMain := pg.NewTxManager(dbMain)

	dbFake := pg_mocks.NewMockDB(t)
	trmFake := pg.NewTxManager(dbFake)

	createrService := &CreaterService{trm: trmFake}
	deleterService := &DeleterService{trm: trmFake}
	BlinkerUseCase := &BlinkerUseCase{
		trm:        trmMain,
		createrSvc: *createrService,
		deleterSvc: *deleterService,
	}

	err := BlinkerUseCase.Blink(clearCtx)

	assert.Len(t, tx.Calls, 3)
	assert.Len(t, dbMain.Calls, 1)
	assert.Empty(t, dbFake.Calls)
	require.NoError(t, err)
}

// TestTxManagerDoUseCaseWithError проверяет работу менеджера транзакций
// в контексте использования внутри UseCase
// тоесть вызов менеджера транзакций внутри другого менеджера транзакций
// с ошибкой в первом же сервисе.
func TestTxManagerDoUseCaseWithError(t *testing.T) {
	t.Parallel()
	clearCtx := context.Background()
	errTest := errors.New("test")

	tx := pg_mocks.NewMockTx(t)
	tx.EXPECT().Exec(mock.Anything, "INSERT").RunAndReturn(
		func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			_, ok := pg.TxFromCtx(ctx)
			assert.True(t, ok)
			return pgconn.NewCommandTag(""), errTest
		},
	)
	tx.EXPECT().Rollback(mock.Anything).Return(nil)

	// эта проверка гарантирует что транзакции начнутся в самом верхнем выхове менеджера
	dbMain := pg_mocks.NewMockDB(t)
	dbMain.EXPECT().Begin(clearCtx).Return(tx, nil)
	trmMain := pg.NewTxManager(dbMain)

	dbFake := pg_mocks.NewMockDB(t)
	trmFake := pg.NewTxManager(dbFake)

	createrService := &CreaterService{trm: trmFake}
	deleterService := &DeleterService{trm: trmFake}
	BlinkerUseCase := &BlinkerUseCase{
		trm:        trmMain,
		createrSvc: *createrService,
		deleterSvc: *deleterService,
	}

	err := BlinkerUseCase.Blink(clearCtx)

	assert.Len(t, tx.Calls, 3)
	assert.Len(t, dbMain.Calls, 1)
	assert.Empty(t, dbFake.Calls)
	require.ErrorIs(t, err, errTest)
}
