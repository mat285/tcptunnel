package tcp

import (
	"context"
	"sync"
)

type ConnManager struct {
	lock  sync.Mutex
	conns map[Conn]Conn
}

func (cm *ConnManager) Register(conn Conn) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	owned := OwnConn(conn, cm.OnError, cm.OnClose)
	cm.conns[conn] = owned
	cm.conns[owned] = owned
}

func (cm *ConnManager) OnError(ctx context.Context, conn Conn, err error) error {

	return nil
}

func (cm *ConnManager) OnClose(ctx context.Context, conn Conn, err error) error {

	return nil
}
