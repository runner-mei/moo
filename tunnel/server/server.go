package server

import (
	"context"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/tunnel"
	"github.com/runner-mei/moo/api"
)

func init() {
	moo.On(func(env *moo.Environment) moo.Option {
		if env.Config.StringWithDefault(api.CfgTunnelRole, "") != "server" {
			return moo.None
		}

		return moo.Provide(func(lifecycle moo.Lifecycle, env *moo.Environment, logger log.Logger) (*tunnel.TunnelServer, error) {
			tunnelSrv, err := tunnel.NewTunnelServer(logger,
				uint32(env.Config.IntWithDefault("tunnel.max_threads", 10)),
				env.Config.DurationWithDefault("tunnel.dail_timeout", 10),
				nil)
			if err != nil {
				return nil, err
			}

			lifecycle.Append(moo.Hook{
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

	moo.On(func(env *moo.Environment) moo.Option {
		if env.Config.StringWithDefault(api.CfgTunnelRole, "") != "server" {
			return moo.None
		}

		return moo.Invoke(func(httpSrv *moo.HTTPServer, tunnelSrv *tunnel.TunnelServer) error {
			httpSrv.FastRoute(false, "tunnel", tunnelSrv)
			return nil
		})
	})
}
