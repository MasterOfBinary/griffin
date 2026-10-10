package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MasterOfBinary/griffin/internal/app"
	"github.com/MasterOfBinary/griffin/internal/safety"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Execute(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, safety.Redact(err.Error()))
		os.Exit(1)
	}
}
