package main

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logrus.Fatalln(err)
	}
}
