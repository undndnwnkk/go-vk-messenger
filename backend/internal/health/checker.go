package health

import (
	"context"
	"time"
)

type Checker struct {
	Checks []func(context.Context) error
}

func (c Checker) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for _, check := range c.Checks {
		if err := check(ctx); err != nil {
			return err
		}
	}
	return nil
}
