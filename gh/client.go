package gh

import (
	"context"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"os"
)

type Client struct {
	*github.Client
}

func NewClient(ctx context.Context) *Client {
	return &Client{github.NewClient(oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
		),
	))}
}
