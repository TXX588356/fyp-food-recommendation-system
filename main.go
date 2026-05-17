package main

import (
	"context"
	"fyp/food-rs/app/cmd/server"
	"log"
)

func main() {
	if err := server.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
