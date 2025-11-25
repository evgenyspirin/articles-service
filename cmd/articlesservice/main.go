package main

import (
	"context"
	"log"
	"os"

	"articles-service/internal"
)

func main() {
	ctx := context.Background()

	app, err := internal.NewApp()
	if err != nil {
		log.Fatalf("init articles service failed: %v", err)
	}
	defer app.Close()

	if err = app.Run(ctx); err != nil {
		app.Logger().Sugar().Errorf("articles service stopped with error: %v", err)
		os.Exit(1)
	}
}
