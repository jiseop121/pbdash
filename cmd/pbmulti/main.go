package main

import (
	"context"
	"os"

	"multi-pocketbase-ui/internal/app"
)

func main() {
	code := app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
