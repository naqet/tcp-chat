package network

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

type Message struct {
	from    net.Addr
	payload []byte
}

type ChatServer struct {
	listenAddr string
	listener   net.Listener
	quitCh     chan struct{}
	msgCh      chan Message
	mutex      sync.Mutex
	clients    map[net.Addr]net.Conn
}

func NewServer(listenAddr string) *ChatServer {
	return &ChatServer{
		listenAddr: listenAddr,
		quitCh:     make(chan struct{}),
		msgCh:      make(chan Message, 10),
		clients:    make(map[net.Addr]net.Conn),
	}
}

func (cs *ChatServer) Start() error {
	listener, err := net.Listen("tcp", cs.listenAddr)

	if err != nil {
		return err
	}

	defer listener.Close()
	defer close(cs.msgCh)

	cs.listener = listener

	go cs.acceptLoop()

	go func() {
		for msg := range cs.msgCh {
			cs.broadcast(&msg)
		}
	}()

	<-cs.quitCh

	return nil
}

func (cs *ChatServer) addClient(conn net.Conn) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.clients[conn.RemoteAddr()] = conn
}

func (cs *ChatServer) removeClient(addr net.Addr) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	delete(cs.clients, addr)
}

func (cs *ChatServer) acceptLoop() {
	for {
		conn, err := cs.listener.Accept()

		if err != nil {
			fmt.Println("error while accepting:", err)
			continue
		}

		cs.addClient(conn)

		go cs.readLoop(conn)
	}
}

func (cs *ChatServer) readLoop(conn net.Conn) {
	defer func() {
		conn.Close()
		cs.removeClient(conn.RemoteAddr())
	}()

	for {
		buf := make([]byte, 10)
		n, err := conn.Read(buf)

		if errors.Is(err, io.EOF) {
			fmt.Println("Client closed the connections:", conn.RemoteAddr())
			break
		} else if err != nil {
			fmt.Println("error while reading:", err)
			continue
		}

        cs.msgCh <- Message{conn.RemoteAddr(), buf[:n]}
	}
}

func (cs *ChatServer) broadcast(msg *Message) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for _, client := range cs.clients {
		if msg.from != client.RemoteAddr() {
			_, err := client.Write([]byte(msg.from.String() + ": " + string(msg.payload)))

			if err != nil {
				fmt.Println("Error while broadcasting the message")
				continue
			}
		}
	}
}
