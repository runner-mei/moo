package server

import (
	"context"
	"strings"
	"net/http"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/tunnel"
	"github.com/runner-mei/errors"
)

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(lifecycle moo.Lifecycle, env *moo.Environment, httpSrv *moo.HTTPServer, logger log.Logger) (*tunnel.TunnelListener, error) {
			acceptURL := env.DaemonUrlPath
			if !strings.HasPrefix(acceptURL, "/") {
				acceptURL = "/" + acceptURL
			}
			if strings.HasSuffix(acceptURL, "/") {
				acceptURL = acceptURL + "tunnel"
			} else {
				acceptURL = acceptURL + "/tunnel"
			}

			engineName := env.Config.StringWithDefault(api.CfgTunnelEngineName, "")
			acceptURL = acceptURL + "/"+ engineName +"?engine_name="+engineName


			tunnelListener, err := tunnel.Listen(logger,
				env.Config.IntWithDefault(api.CfgTunnelMaxThreads, 10),
				env.Config.StringWithDefault(api.CfgTunnelRemoteNetwork, "tcp"),
				env.Config.StringWithDefault(api.CfgTunnelRemoteAddress, ""),
				env.Config.StringWithDefault(api.CfgTunnelRemoteListenAtURL, acceptURL))
			if err != nil {
				return nil, errors.Wrap( err, "tunel listen")
			}

			lifecycle.Append(moo.Hook{
				OnStart: func(context.Context) error {

					go http.Serve(tunnelListener, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						httpSrv.ServeHTTP(w, req)
					}))

					return nil
				},
				OnStop: func(context.Context) error {
					return tunnelListener.Close()
				},
			})
			return tunnelListener, nil
		})
	})

	moo.On(func(*moo.Environment) moo.Option {
		// 增加这个是为了确保会实例化 TunnelListener
		// 不要删它
		return moo.Invoke(func(tunnelSrv *tunnel.TunnelListener) error {
			return nil
		})
	})
}
