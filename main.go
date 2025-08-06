package main

import (
	"context"
	"log"
	"os"

	"charon-key/internal/cli"
)

func main() {
	app := cli.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}