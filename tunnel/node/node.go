package server

import (
	"context"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/tunnel"
	"github.com/runner-mei/moo"
	"go.uber.org/fx"
)

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(lifecycle fx.Lifecycle, env *moo.Environment, logger log.Logger) (*tunnel.TunnelListener, error) {
			tunnelListener, err := tunnel.Listen(logger,
				 env.Config.IntWithDefault("tunnel.max_threads", 10),
				 env.Config.StringWithDefault("tunnel.remote.network", "tcp"), 
				 env.Config.StringWithDefault("tunnel.remote.address", ""), 
				 env.Config.StringWithDefault("tunnel.remote.listen_at_url", ""),)
			if  err != nil {
				return nil, err
			}

			lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) error {
					return nil
				},
				OnStop: func(context.Context) error {
					return tunnelListener.Close()
				},
			})
			return tunnelListener, nil
		})
	})
}
