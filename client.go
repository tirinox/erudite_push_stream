package main

import (
	"bufio"
	"log"
	"net"

	"time"

	"strings"

	"github.com/Jeffail/gabs"
	"strconv"
)

const (
	PING_PERIOD      = 5 * time.Second
	TIME_TO_REGISTER = 20 * time.Second
)

const (
	IDENT_LIMIT   = 256
	MESSAGE_LIMIT = 1024 * 64
	APIKEY_LIMIT  = 256
)
const (
	ERROR_JSON_PARSE      = 1
	ERROR_NO_COMMAND      = 2
	ERROR_UNKNOWN_COMMAND = 3
	ERROR_NO_IDENT        = 4
	ERROR_NO_MESSAGE      = 5
	ERROR_NO_API_KEY      = 6
	ERROR_WRONG_API_KEY   = 7
)

type Client struct {
	reader     *bufio.Reader
	writer     *bufio.Writer
	connection net.Conn
	clientId   string
	send       chan *PushMessage
	connId     int
	lastActivity int64
}

func NewClient(connection IncomingConnection) *Client {
	writer := bufio.NewWriter(connection.conn)
	reader := bufio.NewReader(connection.conn)

	client := &Client{
		connection: connection.conn,
		reader:     reader,
		writer:     writer,
		clientId:   "",
		send:       make(chan *PushMessage),
		connId:     connection.ident,
		lastActivity: time.Now().Unix(),
	}
	return client
}

func (c *Client) writeMessage(m *PushMessage) {
	j := gabs.New()
	j.Set("message", "type")
	j.Set(m.message, "content")
	c.WriteJSON(j)
}

func (c *Client) WriteJSON(j *gabs.Container) {
	j.Set(time.Now().Unix(), "sv_time")
	log.Println("#", c.connId, " writing JSON. ", j.String())
	c.writer.WriteString(j.String() + "\n")
	c.writer.Flush()
}

func (c *Client) WritePublishResult(ok bool, ident string) {
	j := gabs.New()
	j.Set(strconv.FormatBool(ok), "result")
	j.Set(ident, "ident")
	c.WriteJSON(j)
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

func (c *Client) readString(j *gabs.Container, path string, lenLimit, errNo int) (string, bool) {
	if j.ExistsP(path) {
		if item, ok := j.Path(path).Data().(string); ok {
			itemLen := len(item)
			if itemLen > 1 && itemLen < lenLimit {
				return item, true
			} else {
				c.WriteJSON(errorJSON(errNo, path+" has inappropriate length"))
			}
		} else {
			c.WriteJSON(errorJSON(errNo, path+" must be a string"))
		}
	} else {
		c.WriteJSON(errorJSON(errNo, "there must be \""+path+"\" field"))
	}
	return "", false
}

func (c *Client) readIdent(j *gabs.Container) (string, bool) {
	return c.readString(j, "ident", IDENT_LIMIT, ERROR_NO_IDENT)
}

func (c *Client) registerCommand(j *gabs.Container) {
	if ident, ok := c.readIdent(j); ok {
		c.clientId = ident
		g_hub.register <- c
		c.WriteJSON(goodJSON("register"))
	}
}

func (c *Client) publishCommand(j *gabs.Container) {
	if ident, ok := c.readIdent(j); ok {
		if message, ok := c.readString(j, "message", MESSAGE_LIMIT, ERROR_NO_MESSAGE); ok {
			if apiKey, ok := c.readString(j, "api_key", APIKEY_LIMIT, ERROR_NO_API_KEY); ok {
				if apiKey != g_apiKey {
					c.WriteJSON(errorJSON(ERROR_WRONG_API_KEY, "\"api_key\" mismatch"))
				} else {
					g_hub.sendMessage(&PushMessage{
						receiverId: ident,
						message:    MessageContent(message),
						sender:     c,
					})
				}
			}
		}
	}
}

func (c *Client) parseCommand(command string, j *gabs.Container) {
	c.lastActivity = time.Now().Unix()
	switch command {
	case "register":
		c.registerCommand(j)
		break
	case "publish":
		c.publishCommand(j)
		break
	case "pong":
		// do nothing
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
	time.AfterFunc(TIME_TO_REGISTER, func() {
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
			log.Println("#", c.connId, " Disconnecting. Reason: "+err.Error())
			break
		}

		strLine := strings.TrimSpace(string(line))

		if len(strLine) > 0 {
			log.Println("#", c.connId, " received message ", strLine)

			jsonResult, err := gabs.ParseJSON([]byte(line))
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
}

func (c *Client) writePump() {
	log.Println("#", c.connId, " writePump() start")
	ticker := time.NewTicker(PING_PERIOD)
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
			c.writeMessage(message)
			n := len(c.send)
			for i := 0; i < n; i++ {
				c.writeMessage(message)
			}

		case <-ticker.C:

			secondsSinceActivity := time.Duration(time.Now().Unix() - c.lastActivity) * time.Second
			if secondsSinceActivity > PING_PERIOD {
				log.Println("#", c.connId, " closing inactive connection.")
				c.connection.Close()
			}

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
