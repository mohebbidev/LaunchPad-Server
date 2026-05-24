package infrastructure

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgconn"
)

func Retry(ctx context.Context, attempts int, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		backoff := time.Duration(100*(1<<i)) * time.Millisecond
		jitter := time.Duration(rand.Intn(50)) * time.Millisecond

		wait := backoff + jitter

		select {
			case <- time.After(wait):

			case <- ctx.Done():
				return ctx.Err()

		}

	}
	return err
}


func isRetryable(err error) bool {

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "40001",
			"40P01":
			return true
			
		}
	}

	return false

}

