package retry

import (
	"context"
	"time"
)

type Option func(*Retry)

// WithMaxAttempts sets the maximum number of attempts
func WithMaxAttempts(maxAttempts int) Option {
	return func(r *Retry) {
		r.maxAttempts = maxAttempts
	}
}

// WithPolicy sets the retry policy
func WithPolicy(policy Policy) Option {
	return func(r *Retry) {
		r.policy = policy
	}
}

// WithDelay sets the delay between retries
func WithDelay(delay time.Duration) Option {
	return func(r *Retry) {
		r.delay = delay
	}
}

// WithDebug sets the debug mode
func WithDebug(debug bool) Option {
	return func(r *Retry) {
		r.debug = debug
	}
}

// WithContext sets the ctx for Infinite policy retry
func WithContext(ctx context.Context) Option {
	return func(r *Retry) {
		r.ctx = ctx
	}
}

// SetMaxAttempts sets the maximum number of attempts
func (r *Retry) SetMaxAttempts(maxAttempts int) *Retry {
	r.maxAttempts = maxAttempts
	return r
}

// SetPolicy sets the retry policy
func (r *Retry) SetPolicy(policy Policy) *Retry {
	r.policy = policy
	return r
}

// SetDelay sets the delay between retries
func (r *Retry) SetDelay(delay time.Duration) *Retry {
	r.delay = delay
	return r
}

// SetDebug sets the debug mode
func (r *Retry) SetDebug(debug bool) *Retry {
	r.debug = debug
	return r
}
