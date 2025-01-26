package retry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/downloaderbot/pkg/retry"
)

func TestRetry_Do(t *testing.T) {
	t.Run("linear retry", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return nil
		})
		require.NoError(t, err)
	})
	t.Run("backoff retry", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("unsupported retry policy", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.Policy(3)), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return nil
		})
		require.Error(t, err)
	})

	t.Run("linear retry failed", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
	})

	t.Run("backoff retry failed", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
	})

	t.Run("linear retry failed after 3 attempts", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "linear retry failed after 3 attempts", err.Error())
	})

	t.Run("backoff retry failed after 3 attempts", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(3), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "backoff retry failed after 3 attempts", err.Error())
	})

	t.Run("linear retry failed after 3 attempts with custom delay", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(3), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "linear retry failed after 3 attempts", err.Error())
	})

	t.Run("backoff retry failed after 3 attempts with custom delay", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(3), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "backoff retry failed after 3 attempts", err.Error())
	})

	t.Run("linear retry failed after 3 attempts with custom max attempts", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(5), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "linear retry failed after 5 attempts", err.Error())
	})

	t.Run("backoff retry failed after 3 attempts with custom max attempts", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(5), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "backoff retry failed after 5 attempts", err.Error())
	})

	t.Run("linear retry failed after 3 attempts with custom max attempts and delay", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(5), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "linear retry failed after 5 attempts", err.Error())
	})

	t.Run("backoff retry failed after 3 attempts with custom max attempts and delay", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(5), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "backoff retry failed after 5 attempts", err.Error())
	})

	t.Run("linear retry failed after 3 attempts with custom max attempts and delay and custom retry function", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithMaxAttempts(5), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "linear retry failed after 5 attempts", err.Error())
	})

	t.Run("backoff retry failed after 3 attempts with custom max attempts and delay and custom retry function", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyBackoff), retry.WithMaxAttempts(5), retry.WithDelay(400*time.Millisecond)).SetDebug(true)
		err := r.Do(func() error {
			return retry.ErrRetry
		})
		require.Error(t, err)
		require.Equal(t, "backoff retry failed after 5 attempts", err.Error())
	})

	t.Run("linear retry failed with exit error", func(t *testing.T) {
		r := retry.New(retry.WithPolicy(retry.PolicyLinear), retry.WithDelay(200*time.Millisecond)).SetDebug(true)
		var attempt int
		err := r.Do(func() error {
			attempt++
			if attempt == 2 {
				return retry.ErrExit
			}
			return retry.ErrRetry
		})
		require.ErrorIs(t, err, retry.ErrExit)
	})
}
