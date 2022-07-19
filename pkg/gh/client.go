package gh

import (
	"context"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
	"os"
	"time"
)

type Client struct {
	*github.Client
	rl *rate.Limiter
}

func NewClient(ctx context.Context) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return &Client{
		Client: client,
		rl:     rate.NewLimiter(rate.Every(time.Second*2), 1),
	}
}

func (c *Client) Wait(ctx context.Context) error {
	return c.rl.Wait(ctx)
}