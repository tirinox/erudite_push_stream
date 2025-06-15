package main

import (
	"io"
	"log"
	"os"
)

func main() {
	// if you want to turn logging on; set env var ENABLE_LOG
	if os.Getenv("DISABLE_LOG") == "1" {
		log.SetOutput(io.Discard)
	}
	RunPushApp()
}
