package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/traherom/gomponentsconverter"
)

func main() {
	ctx := context.Background()
	if err := gomponentsconverter.Run(ctx, os.Stdout); err != nil {
		slog.Error("Execution failed", "error", err)
		os.Exit(1)
	}
}
