package tcp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Server struct {
	lock    sync.Mutex
	running bool
	stop    chan struct{}
	port    uint16

	newConn ConnProvider

	tunnels map[*Tunnel]struct{}
}

func NewServer(port uint16, provider ConnProvider) *Server {
	return &Server{
		port:    port,
		newConn: provider,
		tunnels: make(map[*Tunnel]struct{}),
		running: false,
	}
}

func (s *Server) Port() uint16 {
	return s.port
}

func (s *Server) Listen(ctx context.Context) error {
	s.lock.Lock()
	if s.running {
		s.lock.Unlock()
		return fmt.Errorf("already running")
	}
	stop := make(chan struct{})
	s.stop = stop
	s.running = true
	s.lock.Unlock()

	listener, err := Listen(s.port)
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
	defer s.closeTunnels(ctx)

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
		fmt.Println("Got new connection on", s.port, conn.RemoteAddr())
		err = s.addConn(ctx, conn)
		if err != nil {
			fmt.Println("Error adding connection for tcp server", err)
			continue
		}
	}
}

func (s *Server) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.running {
		return
	}
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
}

func (s *Server) closeTunnels(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	errs := make([]string, 0, len(s.tunnels))
	for tunnel := range s.tunnels {
		err := tunnel.Stop(ctx)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	s.tunnels = make(map[*Tunnel]struct{})
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (s *Server) addConn(ctx context.Context, conn net.Conn) error {
	// s.lock.Lock()
	// defer s.lock.Unlock()
	fmt.Println("Requesting connection to forward")
	upstream, err := s.newConn(ctx)
	if err != nil {
		conn.Close()
		return err
	}
	client := &TCPClient{
		conn:  conn,
		state: ConnStateOpen,
	}
	tunnel := NewTunnel(client, upstream)
	s.lock.Lock()
	s.tunnels[tunnel] = struct{}{}
	s.lock.Unlock()
	go func() {
		tunnel.Run(ctx)
		s.lock.Lock()
		delete(s.tunnels, tunnel)
		s.lock.Unlock()
	}()
	return nil
}

func (s *Server) listenForNew(ctx context.Context, stop chan struct{}, listener net.Listener) (net.Conn, error) {
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
