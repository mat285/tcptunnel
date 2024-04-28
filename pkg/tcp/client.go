package tcp

import (
	"context"
	"fmt"
	"net"
)

type TCPClient struct {
	conn  net.Conn
	state ConnState
}

func (c *TCPClient) Write(ctx context.Context, data []byte) error {
	n, err := c.conn.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("failed to write all data")
	}
	return nil
}

func (c *TCPClient) Read(ctx context.Context, buf []byte) (int, error) {
	return c.conn.Read(buf)
}

func (c *TCPClient) Close(ctx context.Context) error {
	c.state = ConnStateClosed
	return c.conn.Close()
}

func (c *TCPClient) State() ConnState {
	return c.state
}
