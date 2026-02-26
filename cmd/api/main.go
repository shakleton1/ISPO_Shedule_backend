package main

import (
	"ispo-schedule/internal/app"

	"github.com/rs/zerolog/log"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal().Err(err).Msg("app exited")
	}
}
