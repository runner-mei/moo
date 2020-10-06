package usbkey

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/uuidlogin"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Renderer  *authn.Renderer
	Sessions  authn.Sessions
	Users     *usermodels.Users
	UuidLogin *uuidlogin.UuidLogin
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return fx.Invoke(func(env *moo.Environment, params Params, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {
			usbAddr := strings.TrimSpace(env.Config.StringWithDefault(api.CfgUserUsbKeyListenAddress, ":38091"))
			if usbAddr == "" {
				logger.Info("usbkey skipped")
				return nil
			}
			host, port, _ := net.SplitHostPort(usbAddr)
			if port == "" {
				logger.Info("usbkey skipped - port is empty")
				return nil
			}

			if host == "" {
				host = "127.0.0.1"
			}

			usbPrefix := urlutil.Join(env.DaemonUrlPath, "usbkey")
			usbkeyProxy := NewUSBKey(env, "http://"+net.JoinHostPort(host, port), params.UuidLogin, params.Renderer, params.Sessions, params.Users)
			ssoEcho := httpSrv.Engine().Group("usbkey", middlewares.Funcs...)
			ssoEcho.GET(usbPrefix, loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery == "" {
					http.Redirect(w, r, urlutil.Join(usbPrefix, "login"), http.StatusSeeOther)
					return
				}
				http.Redirect(w, r, urlutil.Join(usbPrefix, "login?"+r.URL.RawQuery), http.StatusSeeOther)
			}))
			ssoEcho.GET(urlutil.Join(usbPrefix, "login"), loong.WrapContextHandler(usbkeyProxy.Login))
			ssoEcho.POST(urlutil.Join(usbPrefix, "login"), loong.WrapContextHandler(usbkeyProxy.Login))
			logger.Info("usbkey started")
			return nil
		})
	})
}
