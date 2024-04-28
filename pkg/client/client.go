package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/blend/go-sdk/logger"
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
	log := logger.GetLogger(ctx)
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

		logger.MaybeDebugfContext(ctx, log, "new message from server")
		if n < 0 {
			continue
		}

		switch buf[0] {
		case protocol.ClientHelloTypeData:
			logger.MaybeDebugfContext(ctx, log, "Creating new data connection")
			c.newDataConnection(ctx)
		case protocol.TypeServerHello:
			sh, err := protocol.ParseServerHelloBytes(buf[:n])
			if err != nil {
				logger.MaybeErrorfContext(ctx, log, "error parsing server hello %s", err.Error())
				continue
			}
			logger.MaybeDebugfContext(ctx, log, "Got ID from server %s", sh.ID)
			c.ID = sh.ID
		default:
			logger.MaybeErrorfContext(ctx, log, "unknown message type %s", buf[0])
		}

	}
	return nil
}

func (c *Client) newDataConnection(ctx context.Context) (net.Conn, error) {
	log := logger.GetLogger(ctx)
	dataConn, err := c.connect(ctx, protocol.ClientHelloTypeData)
	if err != nil {
		return nil, err
	}

	logger.MaybeDebugfContext(ctx, log, "Dialing local port")
	sconn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", c.config.ForwardPort), 5*time.Second)
	if err != nil {
		logger.MaybeErrorfContext(ctx, log, "error dialing local port %s", err.Error())
		dataConn.Close()
		return nil, err
	}
	tunnel := tcp.NewTunnel(tcp.WrappedConn{Conn: dataConn}, tcp.WrappedConn{Conn: sconn})
	go tunnel.Run(ctx)
	return dataConn, nil
}

func (c *Client) connect(ctx context.Context, t byte) (net.Conn, error) {
	log := logger.GetLogger(ctx)
	logger.MaybeDebugfContext(ctx, log, "Dialing Server Address %s", c.config.ServerAddress)
	conn, err := net.DialTimeout("tcp", c.config.ServerAddress, 5*time.Second)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(c.generateClientHello(t))
	if err != nil {
		return nil, err
	}
	logger.MaybeDebugfContext(ctx, log, "Established Connection with Server %s", c.config.ServerAddress)
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
