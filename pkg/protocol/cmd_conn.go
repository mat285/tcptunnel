package protocol

import (
	"context"
	"net"
)

type CmdConn struct {
	net.Conn
}

func NewCmdConn(conn net.Conn) *CmdConn {
	return &CmdConn{Conn: conn}
}

func (c *CmdConn) Write(ctx context.Context, data []byte) error {
	_, err := c.Conn.Write(data)
	return err
}

func (c *CmdConn) Read(ctx context.Context, buf []byte) (int, error) {
	return c.Conn.Read(buf)
}

func (c *CmdConn) Close(ctx context.Context) error {
	return c.Conn.Close()
}
