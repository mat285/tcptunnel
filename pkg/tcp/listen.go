package tcp

import (
	"context"
	"fmt"
	"net"

	"github.com/blend/go-sdk/logger"
)

func Listen(ctx context.Context, port uint16) (net.Listener, error) {
	logger.MaybeDebugfContext(ctx, logger.GetLogger(ctx), "Starting TCP listen on port %d", port)
	return net.Listen("tcp", fmt.Sprintf(":%d", port))
}

func DialConnProvider(addr string) ConnProvider {
	return func(ctx context.Context) (Conn, error) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		return &WrappedConn{Conn: conn}, nil
	}
}
