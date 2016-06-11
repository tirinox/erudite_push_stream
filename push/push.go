package push

import (
	"log"
	"net"
)

var g_hub = NewHub()

type IncomingConnection struct {
	conn  net.Conn
	ident int
}

type MessageContent string

type PushMessage struct {
	receiverId string
	message    MessageContent
	sender     *Client
}

func RunPushApp() {
	readConfiguration()
	log.Println("Erudite Push Steam, socket version. Listening to " + g_bind)
	server, err := net.Listen("tcp", g_bind)
	if server == nil {
		panic("Couldn't start listening: " + err.Error())
	}
	g_hub.Run()
	conns := clientConns(server)
	for {
		go handleConn(<-conns)
	}
}

func clientConns(listener net.Listener) chan IncomingConnection {
	ch := make(chan IncomingConnection)
	i := 0
	go func() {
		for {
			client, err := listener.Accept()
			if client == nil {
				log.Println("Couldn't accept: " + err.Error())
				continue
			}
			i++
			log.Printf("Accepted #%d: %v <-> %v\n", i, client.LocalAddr(), client.RemoteAddr())
			ch <- IncomingConnection{
				conn:  client,
				ident: i,
			}
		}
	}()
	return ch
}

func handleConn(connection IncomingConnection) {
	client := NewClient(connection)
	client.Listen()
}
