package limiter

import (
	"context"
	"fmt"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type Limiter struct {
	inner *limiter.Limiter
}

func New(formatted string) (*Limiter, error) {
	rate, err := limiter.NewRateFromFormatted(formatted)
	if err != nil {
		return nil, fmt.Errorf("invalid rate %q: %w", formatted, err)
	}
	return &Limiter{
		inner: limiter.New(memory.NewStore(), rate),
	}, nil
}

func (l *Limiter) Allow(ctx context.Context, key string) error {
	ctxRate, err := l.inner.Get(ctx, key)
	if err != nil {
		return err
	}
	if ctxRate.Reached {
		return fmt.Errorf("rate limit reached")
	}
	return nil
}
