package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mat285/tcptunnel/pkg/protocol"
	"github.com/mat285/tcptunnel/pkg/tcp"
)

type Target struct {
	ID    uint64
	State uint64

	cmdConn tcp.Conn
	cmdBuf  []byte

	queuelock sync.Mutex
	queue     chan tcp.Conn
}

func NewTarget(id uint64, cmdConn tcp.Conn) *Target {
	target := &Target{
		ID:      id,
		cmdBuf:  make([]byte, 4096),
		cmdConn: cmdConn,
		queue:   make(chan tcp.Conn, 1024),
	}

	return target
}

func (t *Target) RequestCommand(ctx context.Context, req []byte) ([]byte, error) {
	err := t.cmdConn.Write(ctx, req)
	if err != nil {
		// terminate the conn
		return nil, err
	}
	n, err := t.cmdConn.Read(ctx, t.cmdBuf)
	if err != nil {
		return nil, err
	}
	return t.cmdBuf[:n], nil
}

func (t *Target) AddDataConn(ctx context.Context, conn tcp.Conn) error {
	t.queuelock.Lock()
	defer t.queuelock.Unlock()
	if len(t.queue) == cap(t.queue) {
		conn.Close(ctx)
		return fmt.Errorf("max connections reached") // drop excess connections
	}
	t.queue <- conn
	return nil
}

func (t *Target) GetConn(ctx context.Context) (tcp.Conn, error) {
	fmt.Println("Requesting connection from target", t.ID)
	err := t.cmdConn.Write(ctx, []byte{protocol.ClientHelloTypeData})
	if err != nil {
		return nil, err
	}
	return t.WaitForConn(ctx)
}

func (t *Target) WaitForConn(ctx context.Context) (tcp.Conn, error) {
	timeout := time.Second * 5
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("Timeout trying to connect")
	case conn := <-t.queue:
		return conn, nil

	}
}
