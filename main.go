// main
package main

import (
	"net"
	"bufio"
	"log"
)

const LISTEN_TO = ":3540"

func main() {
	log.Println("Erudite Push Steam, socket version. Listening to " + LISTEN_TO)
	server, err := net.Listen("tcp", LISTEN_TO)
	if server == nil {
		panic("couldn't start listening: " + err.Error())
	}
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

func handleConn(client net.Conn) {
	b := bufio.NewReader(client)
	for {
		line, err := b.ReadBytes('\n')
		if err != nil {
			log.Println("Disconnecting. Error: " + err.Error())
			break
		}
		log.Println("Message: " + string(line))
		client.Write(line)
	}
}