package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"math/rand"
	"net"
	"sync"

	"github.com/blend/go-sdk/logger"
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
	log := logger.GetLogger(ctx)
	s.lock.Lock()
	if s.running {
		s.lock.Unlock()
		return fmt.Errorf("already running")
	}
	stop := make(chan struct{})
	s.stop = stop
	s.running = true
	s.lock.Unlock()

	listener, err := tcp.Listen(ctx, s.port)
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

			logger.MaybeErrorfContext(ctx, log, "Error listening for new tcp connections in server %s", err.Error())
			continue
		}
		clientHello, err := protocol.ParseClientHello(conn)
		if err != nil {
			conn.Close()
			logger.MaybeErrorfContext(ctx, log, "Error verifying connection for tcp server %s", err.Error())
			continue
		}

		if subtle.ConstantTimeCompare(clientHello.Secret, s.server.config.Secret) != 1 {
			// deny connection
			conn.Close()
			logger.MaybeErrorfContext(ctx, log, "Wrong Server secret")
			continue
		}

		if clientHello.Type == protocol.ClientHelloTypeData {

			err = s.server.ConnectTargetDataConn(ctx, clientHello, tcp.WrappedConn{Conn: conn})
			if err != nil {
				conn.Close()
				logger.MaybeErrorfContext(ctx, log, "Error adding data connection %s", err.Error())
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
				logger.MaybeErrorfContext(ctx, log, "Error verifying or creating backend %s", err.Error())
				continue
			}
			conn.Write(serverHello.Serialize())
		}

	}
}

func (s *ConnServer) listenForNew(ctx context.Context, stop chan struct{}, listener net.Listener) (net.Conn, error) {
	log := logger.GetLogger(ctx)
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
			logger.MaybeErrorfContext(ctx, log, "Error accepting connection %s", err.Error())
			continue
		}
		return conn, err
	}
}

func (s *ConnServer) RequestDataConnForTarget(ctx context.Context, target *Target) error {
	return target.cmdConn.Write(ctx, []byte{protocol.ClientHelloTypeData})
}
