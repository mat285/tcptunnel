package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mat285/tcptunnel/pkg/protocol"
	"github.com/mat285/tcptunnel/pkg/tcp"
)

type Client struct {
	config Config

	ID      uint64
	cmdConn net.Conn

	dataConns map[net.Conn]struct{}
}

func NewClient(cfg Config) *Client {
	return &Client{
		config:    cfg,
		dataConns: make(map[net.Conn]struct{}),
	}
}

func (c *Client) Start(ctx context.Context) error {
	cmdConn, err := c.connect(ctx, protocol.ClientHelloTypeCommand)
	if err != nil {
		return err
	}
	c.cmdConn = cmdConn
	return c.listenCommands(ctx)
}

func (c *Client) listenCommands(ctx context.Context) error {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := c.cmdConn.Read(buf)
		if err != nil {
			return err
		}
		fmt.Println("Got a message from the server", n)
		if n < 0 {
			continue
		}

		switch buf[0] {
		case protocol.ClientHelloTypeData:
			fmt.Println("Creating a new data connection")
			c.newDataConnection(ctx)
		case protocol.TypeServerHello:
			sh, err := protocol.ParseServerHelloBytes(buf[:n])
			if err != nil {
				fmt.Println("error parsing server hello", err)
				continue
			}
			fmt.Println("Got server hello id", sh.ID)
			c.ID = sh.ID
		default:
			fmt.Println("unknown type", buf[0])
		}

	}
	return nil
}

// func (c *Client) handle(ctx context.Context, conn tcp.Conn) error {
// 	buf := make([]byte, 4096)
// 	defer c.handleClose(ctx, conn)
// 	select {
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	default:
// 	}

// 	// wait for data on this conn before we open one to the local port
// 	n, err := conn.Read(ctx, buf)
// 	if err != nil {
// 		return err
// 	}
// 	local, err := c.forward(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	defer local.Close(ctx)

// 	err = local.Write(ctx, buf[:n])
// 	if err != nil {
// 		return err
// 	}

// 	return tcp.NewTunnel(conn, local).Run(ctx)
// }

// func (c *Client) handleClose(ctx context.Context, conn tcp.Conn) error {
// 	if conn == nil {
// 		return nil
// 	}
// 	conn.Close(ctx)
// 	return c.spawn(ctx)
// }

// func (c *Client) forward(ctx context.Context) (tcp.Conn, error) {
// 	return DialTCP(fmt.Sprintf("127.0.0.1:%d", c.config.ForwardPort))
// }

// func (c *Client) spawnConnections(ctx context.Context) error {
// 	for i := 0; i < c.config.MaxConnections; i++ {
// 		err := c.spawn(ctx)
// 		if err != nil {
// 			fmt.Println(err)
// 			continue
// 		}
// 	}
// 	return nil
// }

// func (c *Client) spawn(ctx context.Context) error {
// 	conn, err := c.connect(ctx, protocol.ClientHelloTypeData)
// 	if err != nil {
// 		return err
// 	}
// 	go c.handle(ctx, tcp.WrapConn(conn))
// 	return nil
// }

func (c *Client) newDataConnection(ctx context.Context) (net.Conn, error) {
	dataConn, err := c.connect(ctx, protocol.ClientHelloTypeData)
	if err != nil {
		return nil, err
	}

	fmt.Println("Dialing local port")
	sconn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", c.config.ForwardPort), 5*time.Second)
	if err != nil {
		fmt.Println("error dialing local", err)
		dataConn.Close()
		return nil, err
	}
	fmt.Println("estanlished local server connection")

	tunnel := tcp.NewTunnel(tcp.WrappedConn{Conn: dataConn}, tcp.WrappedConn{Conn: sconn})
	go tunnel.Run(ctx)
	return dataConn, nil
}

func (c *Client) connect(ctx context.Context, t byte) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", c.config.ServerAddress, 5*time.Second)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(c.generateClientHello(t))
	if err != nil {
		return nil, err
	}
	fmt.Println("Established connection with", c.config.ServerAddress)
	return conn, nil
}

func (c *Client) generateClientHello(t byte) []byte {
	hello := protocol.ClientHello{
		Type:   t,
		ID:     c.ID,
		Port:   c.config.RemotePort,
		Secret: c.config.Secret,
	}
	return hello.Serialize()
}

// func DialTCP(addr string) (tcp.Conn, error) {
// 	conn, err := net.Dial("tcp", addr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return tcp.WrapConn(conn), nil
// }
