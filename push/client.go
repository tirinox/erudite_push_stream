package push

import (
	"bufio"
	"log"
	"net"

	"time"

	"github.com/Jeffail/gabs"
)

const (
	pingPeriod     = 30 * time.Second
	timeToRegister = 20 * time.Second
)

const (
	ERROR_JSON_PARSE      = 1
	ERROR_NO_COMMAND      = 2
	ERROR_UNKNOWN_COMMAND = 3
	ERROR_NO_IDENT        = 4
)

type Client struct {
	reader      *bufio.Reader
	writer      *bufio.Writer
	connection  net.Conn
	isPublisher bool
	clientId    string
	send        chan MessageContent
	connId      int
}

func NewClient(connection IncomingConnection) *Client {
	writer := bufio.NewWriter(connection.conn)
	reader := bufio.NewReader(connection.conn)

	client := &Client{
		connection:  connection.conn,
		reader:      reader,
		writer:      writer,
		isPublisher: false,
		clientId:    "",
		send:        make(chan MessageContent),
		connId:      connection.ident,
	}
	return client
}

func (c *Client) WriteJSON(j *gabs.Container) {
	log.Println("#", c.connId, " writing JSON. ", j.String())
	c.writer.WriteString(j.String() + "\n")
	c.writer.Flush()
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

func (c *Client) close() {
	log.Println("#", c.connId, " close()")
	close(c.send)
}

func (c *Client) parseCommand(command string, j *gabs.Container) {
	switch command {
	case "register":
		if j.Exists("ident") {
			if ident, ok := j.Path("ident").Data().(string); ok {
				identLen := len(ident)
				if identLen > 1 && identLen < 256 {
					c.clientId = ident
					g_hub.register <- c
				} else {
					c.WriteJSON(errorJSON(ERROR_NO_IDENT, "ident has inappropriate length"))
				}
			} else {
				c.WriteJSON(errorJSON(ERROR_NO_IDENT, "ident must be a string"))
			}
		} else {
			c.WriteJSON(errorJSON(ERROR_NO_IDENT, "there must be \"ident\" field"))
		}

		break
	case "publish":
		c.WriteJSON(goodJSON(command))
		break
	default:
		c.WriteJSON(errorJSON(ERROR_UNKNOWN_COMMAND, "unknown command: "+command))
	}
}

func (c *Client) sendPing() {
	log.Println("#", c.connId, " sending ping")
	j := gabs.New()
	j.Set("ping", "type")
	c.WriteJSON(j)
}

func (c *Client) registerTimeout() {
	time.AfterFunc(timeToRegister, func() {
		if c.clientId == "" {
			log.Println("#", c.connId, " I am closing the connection. You didn't manage to register in time. Too late!")
			c.connection.Close()
		} else {
			log.Println("#", c.connId, " Register timeout does nothing. You are registered.")
		}
	})
}

func (c *Client) readPump() {

	log.Println("#", c.connId, " readPump() start")
	defer func() { log.Println("#", c.connId, " readPump() end") }()

	c.registerTimeout()

	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			log.Println("#", c.connId, " Disconnecting. Reason: " + err.Error())
			break
		}

		log.Println("#", c.connId, " received message ", line)

		jsonResult, err := gabs.ParseJSON(line)
		if err == nil {
			if jsonResult.Exists("command") {
				if command, ok := jsonResult.Path("command").Data().(string); ok {
					c.parseCommand(command, jsonResult)
				} else {
					c.WriteJSON(errorJSON(ERROR_NO_COMMAND, "command must be a string"))
				}
			} else {
				c.WriteJSON(errorJSON(ERROR_NO_COMMAND, "there must be \"command\" field"))
			}
		} else {
			c.WriteJSON(errorJSON(ERROR_JSON_PARSE, "JSON parse error"))
		}
	}
}

func (c *Client) writePump() {
	log.Println("#", c.connId, " writePump() start")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Println("#", c.connId, " ping stop")
		ticker.Stop()

		log.Println("#", c.connId, " writePump() end")
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				log.Println("#", c.connId, " unable to read a message; the send channel was closed")
				return
			}
			c.WriteJSON(message)
			n := len(c.send)
			for i := 0; i < n; i++ {
				c.WriteJSON(message)
			}

		case <-ticker.C:
			c.sendPing()
		}
	}
}

func (c *Client) Listen() {

	defer func() {
		log.Println("#", c.connId, " Listen() closing")
		c.close()
		g_hub.unregister <- c
	}()

	go c.writePump()
	c.readPump()
}
