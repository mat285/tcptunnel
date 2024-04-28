package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/blend/go-sdk/logger"
	"github.com/mat285/tcptunnel/pkg/tcp"
)

type Backend struct {
	*tcp.Server

	secret []byte

	lock    sync.Mutex
	running bool

	targets        map[uint64]*Target
	addedDataConns chan idConn
}

type idConn struct {
	id   uint64
	conn tcp.Conn
}

func NewBackend(port uint16, secret []byte) *Backend {
	backend := &Backend{
		targets:        make(map[uint64]*Target),
		secret:         secret,
		addedDataConns: make(chan idConn, 16),
	}
	backend.Server = tcp.NewServer(port, backend.NextConn)
	return backend
}

func (b *Backend) AddTarget(ctx context.Context, target *Target) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.targets[target.ID] = target
}

func (b *Backend) Run(ctx context.Context) error {
	if b.running {
		return fmt.Errorf("already running backend")
	}
	b.lock.Lock()
	b.running = true
	b.lock.Unlock()

	return b.runRestartServer(ctx)
}

func (b *Backend) runRestartServer(ctx context.Context) error {
	log := logger.GetLogger(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := b.Server.Listen(ctx)
		if err != nil {
			logger.MaybeErrorfContext(ctx, log, "Error running server %s", err.Error())
			continue
		}
	}
}

func (b *Backend) NextConn(ctx context.Context) (tcp.Conn, error) {
	log := logger.GetLogger(ctx)
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, target := range b.targets {
		logger.MaybeDebugfContext(ctx, log, "Requesting connection from %d", target.ID)
		conn, err := target.GetConn(ctx)
		if err != nil {
			logger.MaybeErrorfContext(ctx, log, "Error requesting con from target %d %s", target.ID, err.Error())
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("No available connections")
}

func (b *Backend) ValidSecret(secret []byte) bool {
	if len(secret) != len(b.secret) {
		return false
	}
	for i := 0; i < len(secret); i++ {
		if secret[i] != b.secret[i] {
			return false
		}
	}
	return true
}
