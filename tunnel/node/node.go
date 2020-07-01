package server

import (
	"context"
	"strings"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/tunnel"
	"github.com/runner-mei/errors"
	"go.uber.org/fx"
)

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(lifecycle fx.Lifecycle, env *moo.Environment, logger log.Logger) (*tunnel.TunnelListener, error) {
			acceptURL := env.DaemonUrlPath
			if !strings.HasPrefix(acceptURL, "/") {
				acceptURL = "/" + acceptURL
			}
			if strings.HasSuffix(acceptURL, "/") {
				acceptURL = acceptURL + "tunnel"
			} else {
				acceptURL = acceptURL + "/tunnel"
			}
			tunnelListener, err := tunnel.Listen(logger,
				env.Config.IntWithDefault("tunnel.max_threads", 10),
				env.Config.StringWithDefault("tunnel.remote.network", "tcp"),
				env.Config.StringWithDefault("tunnel.remote.address", ""),
				env.Config.StringWithDefault("tunnel.remote.listen_at_url", acceptURL))
			if err != nil {
				return nil, errors.Wrap( err, "tunel listen")
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

	moo.On(func() moo.Option {
		return fx.Invoke(func(tunnelSrv *tunnel.TunnelListener) error {
			return nil
		})
	})
}
