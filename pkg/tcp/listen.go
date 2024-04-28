package tcp

import (
	"context"
	"fmt"
	"net"
)

func Listen(port uint16) (net.Listener, error) {
	fmt.Println("Starting TCP listen on port", port)
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
