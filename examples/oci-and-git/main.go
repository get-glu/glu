package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("error running pipelines", "error", err)
		os.Exit(1)
	}
}
