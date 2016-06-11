package push

import (
	"bufio"
	"log"
	"net"

	"time"

	"github.com/Jeffail/gabs"
)

const pingPeriod = 20 * time.Second

const (
	ERROR_UNKNOWN_COMMAND = 4
	ERROR_JSON_PARSE = 1
	ERROR_NO_COMMAND = 2
	ERROR_COMMAND_MUST_BE_STRING = 3
)

type Client struct {
	reader      *bufio.Reader
	writer      *bufio.Writer
	connection  *net.Conn
	isPublisher bool
	id          string
	send        chan MessageContent
}

func NewClient(connection net.Conn) *Client {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)

	client := &Client{
		connection:  &connection,
		reader:      reader,
		writer:      writer,
		isPublisher: false,
		id:          "",
		send:        make(chan MessageContent),
	}
	return client
}

func (client *Client) Write(s string) {
	client.writer.WriteString(s + "\n")
	client.writer.Flush()
}

func (client *Client) WriteJSON(j *gabs.Container) {
	client.writer.WriteString(j.String())
	client.writer.Flush()
}

func errorJSON(code int, message string) *gabs.Container {
	j := gabs.New()
	j.Set("bad", "result")
	j.Set(message, "error")
	j.Set(code, "code")
	return j
}

func goodJSON(command string) *gabs.Container {
	j := gabs.New()
	j.Set("good", "result")
	j.Set(command, "command")
	return j
}

func (client *Client) parseCommand(command string, j *gabs.Container) {
	switch command {
	case "register":
		client.WriteJSON(goodJSON(command))
		break
	case "publish":
		client.WriteJSON(goodJSON(command))
		break
	default:
		client.WriteJSON(errorJSON(ERROR_UNKNOWN_COMMAND, "unknown command: "+command))
	}
}

func (client *Client) sendPing() {
	j := gabs.New()
	j.Set("ping", "type")
	client.WriteJSON(j)
}

func (client *Client) readPump() {
	for {
		line, err := client.reader.ReadBytes('\n')
		if err != nil {
			log.Println("Disconnecting. Reason: " + err.Error())
			break
		}

		jsonResult, err := gabs.ParseJSON(line)
		if err == nil {
			if jsonResult.Exists("command") {
				if command, ok := jsonResult.Path("command").Data().(string); ok {
					client.parseCommand(command, jsonResult)
				} else {
					client.WriteJSON(errorJSON(ERROR_COMMAND_MUST_BE_STRING, "command must be a string"))
				}
			} else {
				client.WriteJSON(errorJSON(ERROR_NO_COMMAND, "no command"))
			}
		} else {
			client.WriteJSON(errorJSON(ERROR_JSON_PARSE, "JSON parse error"))
		}
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case message := <-client.send:
			client.WriteJSON(message)

		case <-ticker.C:
			client.sendPing()
		}
	}
}

func (client *Client) Listen() {
	go client.writePump()
	client.readPump()
	defer func() {
		g_hub.unregister <- client
	}()
}
