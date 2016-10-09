package main

import (
	"io/ioutil"
	"log"
	"os"
)

func main() {
	// if you want to turn logging on; set env var ENABLE_LOG
	if os.Getenv("ENABLE_LOG") == "" {
		log.SetOutput(ioutil.Discard)
	}	
	RunPushApp()
}
