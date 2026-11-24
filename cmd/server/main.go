package main

import (
	"log"

	"water-demand/backend"
)

func main() {
	if err := backend.Run(); err != nil {
		log.Fatal(err)
	}
}
