package server

import (
	"context"
	"time"

	"github.com/blend/go-sdk/configutil"
	"github.com/blend/go-sdk/logger"
	"github.com/mat285/tcptunnel/pkg/config"
)

type Config struct {
	Port uint16

	ClientConnectTimeout time.Duration

	Secret []byte
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
	ctx = logger.WithLogger(ctx, logger.All())
	return ctx, nil
}
