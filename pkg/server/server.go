package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/mat285/tcptunnel/pkg/protocol"
	"github.com/mat285/tcptunnel/pkg/tcp"
)

type Server struct {
	lock    sync.Mutex
	running bool
	stop    chan struct{}

	config Config

	connServer *ConnServer

	targets map[uint64]*Target

	backends map[uint16]*Backend // inbound traffic for port maps to serving backends
}

func NewServer(cfg Config) *Server {
	server := &Server{
		config:   cfg,
		backends: make(map[uint16]*Backend),
		targets:  make(map[uint64]*Target),
	}
	server.connServer = NewConnServer(server, cfg.Port)
	return server
}

func (s *Server) Start(ctx context.Context) error {
	s.lock.Lock()
	if s.running {
		s.lock.Unlock()
		return fmt.Errorf("server already started")
	}
	s.running = true
	s.lock.Unlock()

	fmt.Println("Starting conn server listen")
	err := s.connServer.Listen(ctx)
	s.lock.Lock()
	s.running = false
	s.lock.Unlock()
	return err
}

func (s *Server) Stop() error {
	s.lock.Lock()
	if !s.running {
		s.lock.Unlock()
		return nil
	}
	s.connServer.Stop()
	s.stopBackendsUnsafe()
	s.lock.Unlock()
	return nil
}

func (s *Server) stopBackendsUnsafe() {
	for _, backend := range s.backends {
		backend.Stop()
	}
	s.backends = make(map[uint16]*Backend)
}

func (s *Server) CreateOrVerifyBackend(ctx context.Context, hello *protocol.ClientHello, target *Target) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if existing, has := s.backends[hello.Port]; has && existing != nil {
		if existing.ValidSecret(hello.Secret) {
			fmt.Println("Added target to existing backend")
			existing.AddTarget(ctx, target)
			return false, nil
		}
		return false, fmt.Errorf("Invalid secret for existing backend")
	}
	fmt.Println("Adding target to new backend", hello.Port)
	backend := NewBackend(hello.Port, hello.Secret)
	s.backends[hello.Port] = backend
	backend.AddTarget(ctx, target)
	s.targets[hello.ID] = target
	go s.listenAndCleanup(ctx, backend)
	return true, nil
}

func (s *Server) ConnectTargetDataConn(ctx context.Context, hello *protocol.ClientHello, conn tcp.Conn) error {
	fmt.Println("Adding conn to target for backend", hello.Port)
	s.lock.Lock()
	defer s.lock.Unlock()
	backend := s.backends[hello.Port]
	if backend == nil {
		fmt.Println("backend nil")
		return fmt.Errorf("No existing backend")
	}
	if !backend.ValidSecret(hello.Secret) {
		fmt.Println("invalid secret")
		return fmt.Errorf("Invalid secret")
	}
	target := s.targets[hello.ID]
	if target == nil {
		return fmt.Errorf("No existing target")
	}
	fmt.Println("Backend adding target conn")
	target.AddDataConn(ctx, conn)
	// backend.AddTargetConn(ctx, hello.ID, conn)
	return nil
}

func (s *Server) listenAndCleanup(ctx context.Context, backend *Backend) error {
	err := backend.Run(ctx)
	s.lock.Lock()
	if s.backends[backend.Port()] == backend {
		delete(s.backends, backend.Port())
	}
	s.lock.Unlock()
	return err
}
