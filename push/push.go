package push

import (
	"log"
	"net"
)

var g_hub = NewHub()

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

func clientConns(listener net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
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
			ch <- client
		}
	}()
	return ch
}

func handleConn(connection net.Conn) {
	client := NewClient(connection)
	client.Listen()
}
