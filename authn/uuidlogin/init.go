package uuidlogin

import (
	"net/http"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Renderer *authn.Renderer
	Sessions authn.Sessions
	Users    *usermodels.Users
}

func init() {
	moo.On(func() moo.Option {
		return fx.Invoke(func(env *moo.Environment, params Params, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {
			uuidProxy := NewUuidLogin(env, params.Renderer, params.Sessions, params.Users)
			httpSrv.FastRoute(false, "uuid", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				uuidProxy.Login(r.Context(), w, r)
			}))
			logger.Info("uuid login started")
			return nil
		})
	})
}
