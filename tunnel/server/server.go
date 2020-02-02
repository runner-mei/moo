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
		return fx.Provide(func(lifecycle fx.Lifecycle, env *moo.Environment, logger log.Logger) (*tunnel.TunnelServer, error) {
			tunnelSrv, err := tunnel.NewTunnelServer(logger,
				uint32(env.Config.IntWithDefault("tunnel.max_threads", 10)),
				env.Config.DurationWithDefault("tunnel.dail_timeout", 10),
				nil)
			if  err != nil {
				return nil, err
			}

			lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) error {
					return nil
				},
				OnStop: func(context.Context) error {
					return tunnelSrv.Close()
				},
			})
			return tunnelSrv, nil
		})
	})
}
