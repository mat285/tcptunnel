package tcp

import (
	"context"
	"net"
)

type ConnState int

const (
	ConnStateUnknown ConnState = 0
	ConnStateOpen    ConnState = 1
	ConnStateClosed  ConnState = 2
)

type Conn interface {
	Write(context.Context, []byte) error
	Read(context.Context, []byte) (int, error)
	Close(context.Context) error
	State() ConnState
}

type ConnProvider func(context.Context) (Conn, error)

type OwnedConn struct {
	Conn
	OnError func(context.Context, Conn, error) error
	Closer  func(context.Context, Conn, error) error
}

func OwnConn(conn Conn, onE, onClose func(context.Context, Conn, error) error) Conn {
	return &OwnedConn{
		Conn:    conn,
		OnError: onE,
		Closer:  onClose,
	}
}

func (oc *OwnedConn) Write(ctx context.Context, data []byte) error {
	err := oc.Conn.Write(ctx, data)
	if oc.OnError != nil {
		err = oc.OnError(ctx, oc.Conn, err)
	}
	return err
}

func (oc *OwnedConn) Read(ctx context.Context, buf []byte) (int, error) {
	n, err := oc.Conn.Read(ctx, buf)
	if oc.OnError != nil {
		err = oc.OnError(ctx, oc.Conn, err)
	}
	return n, err
}

func (oc *OwnedConn) Close(ctx context.Context) error {
	err := oc.Conn.Close(ctx)
	if oc.Closer == nil {
		return err
	}
	return oc.Closer(ctx, oc.Conn, err)
}

type WrappedConn struct {
	net.Conn
	state ConnState
}

func WrapConn(conn net.Conn) Conn {
	if conn == nil {
		return nil
	}
	return WrappedConn{
		Conn:  conn,
		state: ConnStateOpen,
	}
}

func (w WrappedConn) Write(ctx context.Context, data []byte) error {
	_, err := w.Conn.Write(data)
	return err
}

func (w WrappedConn) Read(ctx context.Context, buf []byte) (int, error) {
	return w.Conn.Read(buf)
}

func (w WrappedConn) Close(ctx context.Context) error {
	w.state = ConnStateClosed
	return w.Conn.Close()
}

func (w WrappedConn) State() ConnState {
	return w.state
}
