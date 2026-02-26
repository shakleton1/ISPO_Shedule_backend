package main

import (
	"log"

	"ispo-schedule/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
