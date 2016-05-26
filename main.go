// main
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 8) / 10
	maxMessageSize     = 16000
	sendBufferSize     = maxMessageSize * 2
	upgraderBufferSize = sendBufferSize
)

type channel struct {
	connections map[*connection] bool
	name string
}

type hub struct {
	connections map[string]*connection
	register    chan *connection
	unregister  chan *connection
}

func makeHub() *hub {
	return &hub{
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[string]*connection),
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
		}
	}
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
	chanName string
}

func (c *connection) readPump(h *hub) {
	defer func() {
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveSubscriber(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, h *hub) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	c := &connection{send: make(chan []byte, sendBufferSize), ws: ws}
	h.register <- c
	go c.writePump()
	c.readPump(h)
}

func servePublisher(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, h *hub) {
	// 1. get the channel's name
	vars := mux.Vars(r)
	if chanName, ok := vars["id"]; ok {
		log.Println("subscribe chanName " + chanName)
	}

	// 2. найти этот онекшн
	// 3. влепить ему в send сообщение
}

func main() {

	addr := flag.String("addr", ":10026", "http service address")
	log.Println("Starting erudite_push_stream at " + *addr)

	h := makeHub()
	go h.run()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  upgraderBufferSize,
		WriteBufferSize: upgraderBufferSize,
	}

	router := mux.NewRouter()
	router.HandleFunc("/sub/{id}", func(w http.ResponseWriter, r *http.Request) {
		serveSubscriber(w, r, &upgrader, h)
	})
	router.HandleFunc("/pub/{id}", func(w http.ResponseWriter, r *http.Request) {
		servePublisher(w, r, &upgrader, h)
	})
	http.Handle("/", router)

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
