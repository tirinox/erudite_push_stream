package main

import (
	"bufio"
	"log"
	"net"

	"github.com/Jeffail/gabs"
)

type Client struct {
	reader     *bufio.Reader
	writer     *bufio.Writer
	connection *net.Conn
}

func NewClient(connection net.Conn) *Client {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)

	client := &Client{
		connection: connection,
		reader:     reader,
		writer:     writer,
	}
	return client
}

func (client *Client) Write(s string) {
	client.writer.WriteString(s + "\n")
	client.writer.Flush()
}

func (client *Client) Listen() {
	for {
		line, err := client.reader.ReadBytes('\n')
		if err != nil {
			log.Println("Disconnecting. Reason: " + err.Error())
			break
		}

		jsonResult, err := gabs.ParseJSON(line)
		if err == nil {
			if jsonResult.Exists("command") {
				command := jsonResult.Path("command").Data().(string)
				client.Write("your command is " + command)
			} else {
				client.Write("no command!")
			}
		} else {
			client.Write("read json error!")
		}
	}
}
