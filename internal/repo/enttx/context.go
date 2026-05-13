package enttx

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
)

type txContextKey struct{}

// WithTx stores an ent transaction in context so nested repository calls can
// join the same transaction instead of opening a partial nested write.
func WithTx(ctx context.Context, tx *ent.Tx) context.Context {
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txContextKey{}, tx)
}

func FromContext(ctx context.Context) (*ent.Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txContextKey{}).(*ent.Tx)
	return tx, ok && tx != nil
}

func Client(ctx context.Context, base *ent.Client) *ent.Client {
	if tx, ok := FromContext(ctx); ok {
		return tx.Client()
	}
	return base
}

type Session struct {
	tx     *ent.Tx
	client *ent.Client
	owned  bool
}

func Begin(ctx context.Context, base *ent.Client) (context.Context, *Session, error) {
	if tx, ok := FromContext(ctx); ok {
		return ctx, &Session{tx: tx, client: tx.Client(), owned: false}, nil
	}
	tx, err := base.Tx(ctx)
	if err != nil {
		return ctx, nil, fmt.Errorf("start ent transaction: %w", err)
	}
	return WithTx(ctx, tx), &Session{tx: tx, client: tx.Client(), owned: true}, nil
}

func (s *Session) Client() *ent.Client {
	if s == nil {
		return nil
	}
	return s.client
}

func (s *Session) Tx() *ent.Tx {
	if s == nil {
		return nil
	}
	return s.tx
}

func (s *Session) Commit() error {
	if s == nil || !s.owned || s.tx == nil {
		return nil
	}
	return s.tx.Commit()
}

func (s *Session) Owned() bool {
	if s == nil {
		return false
	}
	return s.owned
}

func (s *Session) Rollback() {
	if s == nil || !s.owned || s.tx == nil {
		return
	}
	_ = s.tx.Rollback()
}
