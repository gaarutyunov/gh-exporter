package gh

import (
	"context"
	"github.com/google/go-github/v45/github"
	"golang.org/x/time/rate"
	"time"
)

type Limiter struct {
	rl     *rate.Limiter
	limits *github.RateLimits
}

type Option func(*Limiter)

type GitHubLimit int

const (
	CoreLimit GitHubLimit = iota
	SearchLimit
	GraphQLLimit
)

func WithLimit(limit GitHubLimit) Option {
	return func(c *Limiter) {
		if c.limits == nil {
			return
		}

		switch limit {
		case CoreLimit:
			c.rl.SetLimit(rate.Limit(
				float64(c.limits.Core.Remaining) / c.limits.Core.Reset.Sub(time.Now()).Seconds(),
			))
		case SearchLimit:
			c.rl.SetLimit(rate.Limit(
				float64(c.limits.Search.Remaining) / c.limits.Core.Reset.Sub(time.Now()).Seconds(),
			))
		case GraphQLLimit:
			c.rl.SetLimit(rate.Limit(
				float64(c.limits.GraphQL.Remaining) / c.limits.GraphQL.Reset.Sub(time.Now()).Seconds(),
			))
		}
	}
}

func WithBurst(n int) Option {
	return func(c *Limiter) {
		c.rl.SetBurst(n)
	}
}

func NewLimiter(client *Client, options ...Option) *Limiter {
	l := &Limiter{
		rl: rate.NewLimiter(rate.Every(time.Second*2), 1),
	}

	limits, _, _ := client.RateLimits(context.Background())

	l.limits = limits

	for _, option := range options {
		option(l)
	}

	return l
}

func (c *Limiter) Wait(ctx context.Context) error {
	return c.rl.Wait(ctx)
}
