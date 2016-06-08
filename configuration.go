package main

import "os"

var g_bind string
var g_apiKey string

func readConfiguration() {
	g_bind = os.Getenv("BIND")
	if g_bind == "" {
		g_bind = ":10026"
	}

	g_apiKey = os.Getenv("API_KEY")
}

func configBind() string {
	return g_bind
}

func configApiKey() string {
	return g_apiKey
}