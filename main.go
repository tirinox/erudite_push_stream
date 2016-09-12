package main

import (
	"io/ioutil"
	"log"
)

func main() {
	// if you want to turn logging off
	log.SetOutput(ioutil.Discard)
	RunPushApp()
}
