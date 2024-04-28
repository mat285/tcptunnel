package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mat285/tcptunnel/pkg/server"
	"github.com/spf13/cobra"
)

func run() error {
	ctx := context.Background()
	cfg := server.Config{}
	paths := []string{}
	cmd := &cobra.Command{
		Use:           "run-server",
		Short:         "Runs the tcp-tunnel server",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, _ []string) error {
			err := cfg.Resolve(ctx, paths...)
			if err != nil {
				return err
			}
			ctx, err = cfg.Context(ctx)
			if err != nil {
				return err
			}
			cfg.Port = 7890
			cfg.Secret = []byte("11111111111111111111111111111111")

			s := server.NewServer(cfg)
			return s.Start(ctx)
		},
	}

	cmd.PersistentFlags().StringSliceVar(
		&paths,
		"file",
		paths,
		"Path to a file where '.yml' configuration is stored; can be specified multiple times, last provided has highest precedence when merging",
	)

	return cmd.Execute()
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
