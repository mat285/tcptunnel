package client

import (
	"context"

	"github.com/blend/go-sdk/configutil"
	"github.com/mat285/tcptunnel/pkg/config"
)

type Config struct {
	MaxConnections int

	ServerAddress string
	ForwardPort   uint16
	LocalPort     uint16
	RemotePort    uint16
	Secret        []byte
}

// Resolve populates configuration fields from a variety of input sources
func (c *Config) Resolve(ctx context.Context, files ...string) error {
	if err := config.ResolveFromFiles(&c, files...); err != nil {
		return err
	}
	return configutil.Resolve(ctx)
}

func (c Config) Context(ctx context.Context) (context.Context, error) {
	ctx = WithConfig(ctx, c)
	return ctx, nil
}
