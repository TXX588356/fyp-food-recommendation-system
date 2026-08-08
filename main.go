package main

import (
	"context"
	"fyp/food-rs/app/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(context.Background()); err != nil {
		log.Fatal(err)
	}
}
