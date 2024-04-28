package server

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"

	"github.com/mat285/tcptunnel/pkg/protocol"
	"github.com/mat285/tcptunnel/pkg/tcp"
)

type ConnServer struct {
	lock    sync.Mutex
	running bool
	stop    chan struct{}

	port   uint16
	server *Server
}

func NewConnServer(server *Server, port uint16) *ConnServer {
	return &ConnServer{
		running: false,
		port:    port,
		server:  server,
	}
}

func (s *ConnServer) Stop() error {
	s.lock.Lock()
	if !s.running {
		s.lock.Unlock()
		return nil
	}
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
	s.lock.Unlock()
	return nil
}

func (s *ConnServer) Listen(ctx context.Context) error {
	s.lock.Lock()
	if s.running {
		s.lock.Unlock()
		return fmt.Errorf("already running")
	}
	stop := make(chan struct{})
	s.stop = stop
	s.running = true
	s.lock.Unlock()

	listener, err := tcp.Listen(s.port)
	if err != nil {
		return err
	}

	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		s.stop = nil
		s.running = false
	}()
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stop:
			return nil
		default:
		}

		conn, err := s.listenForNew(ctx, stop, listener)
		if err != nil {
			fmt.Println("Error listening for new tcp connections in server", err)
			continue
		}
		clientHello, err := protocol.ParseClientHello(conn)
		if err != nil {
			conn.Close()
			fmt.Println("Error verifying connection for tcp server", err)
			continue
		}

		if clientHello.Type == protocol.ClientHelloTypeData {
			// err = s.handleNewDataConn(ctx, clientHello, tcp.WrappedConn{Conn: conn})

			err = s.server.ConnectTargetDataConn(ctx, clientHello, tcp.WrappedConn{Conn: conn})
			if err != nil {
				conn.Close()
				fmt.Println("Error adding data con", err)
				continue
			}
		} else {
			serverHello := protocol.ServerHello{
				Type:   protocol.TypeServerHello,
				ID:     rand.Uint64(),
				Port:   clientHello.Port,
				Secret: clientHello.Secret,
			}
			clientHello.ID = serverHello.ID

			target := NewTarget(serverHello.ID, tcp.WrapConn(conn))

			_, err = s.server.CreateOrVerifyBackend(ctx, clientHello, target)
			if err != nil {
				conn.Close()
				fmt.Println("Error verifying or creating backend", err)
				continue
			}
			fmt.Println("Sending server hello")

			conn.Write(serverHello.Serialize())
		}

	}
}

func (s *ConnServer) listenForNew(ctx context.Context, stop chan struct{}, listener net.Listener) (net.Conn, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-stop:
			return nil, nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("server accept error", err)
			continue
		}
		return conn, err
	}
}

func (s *ConnServer) RequestDataConnForTarget(ctx context.Context, target *Target) error {
	return target.cmdConn.Write(ctx, []byte{protocol.ClientHelloTypeData})
}

// func (s *ConnServer) GetConnectionForTarget(ctx context.Context, target *Target) (tcp.Conn, error) {
// 	fmt.Println("Requesting connection from target", target.ID)
// 	err := target.cmdConn.Write(ctx, []byte{protocol.ClientHelloTypeData})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return target.WaitForConn(ctx)
// }
