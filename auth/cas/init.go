package cas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/auth"
	"github.com/runner-mei/moo/db"
	"github.com/runner-mei/moo/operation_logs"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	LoginManager *auth.LoginManager
	Renderer     *auth.Renderer
	Sessions     auth.Sessions
	Users        *usermodels.Users
	UserSyncer   UserSyncer `optional:"true"`
}

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(env *moo.Environment, db *db.ArgModelDb, ologger operation_logs.OperationLogger) (UserSyncer, error) {
			return CreateUserSyncer(env, db.DB)
		})
	})

	moo.On(func() moo.Option {
		return fx.Invoke(func(env *moo.Environment, params Params, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {
			casURL := strings.TrimSpace(env.Config.StringWithDefault("users.cas.server", ""))
			if casURL == "" {
				logger.Info("cas skipped")
				return nil
			}
			u, err := url.Parse(casURL)
			if err != nil {
				return errors.Wrap(err, "users.cas.server is invalid url - '"+casURL+"'")
			}

			casPrefix := urlutil.Join(env.DaemonUrlPath, "cas")

			fields := map[string]string{}
			env.Config.ForEachWithPrefix("users.cas.fields.", func(key string, value interface{}) {
				key = strings.TrimPrefix(key, "users.cas.fields.")
				fields[key] = fmt.Sprint(value)
			})
			roles := env.Config.StringsWithDefault("users.cas.roles", nil)
			authOpts := &CASOptions{
				Env:           env,
				Logger:        logger.Named("cas"),
				UserPrefix:    env.Config.StringWithDefault("users.cas.user_prefix", ""),
				URL:           u,
				Client:        httputil.InsecureHttpClent,
				Sessions:      params.Sessions,
				Renderer:      params.Renderer,
				SendService:   true,
				LoginCallback: urlutil.Join(casPrefix, "login_callback"),
				// LogoutCallback: urlutil.Join(casPrefix, "logout_callback"),
				IgnoreList: []string{
					urlutil.Join(casPrefix, "login"),
					urlutil.Join(casPrefix, "logout"),
				},
				Roles:      roles,
				Fields:     fields,
				Users:      params.Users,
				UserSyncer: params.UserSyncer,
			}
			casClient := NewCASClient(authOpts)

			ssoEcho := httpSrv.Engine().Group("cas", middlewares.Funcs...)
			ssoEcho.GET("", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(casPrefix, "login?"+r.URL.RawQuery), http.StatusSeeOther)
			}))
			ssoEcho.GET("/login", loong.WrapHandlerFunc(casClient.RedirectToLogin))
			ssoEcho.POST("/login", loong.WrapHandlerFunc(casClient.RedirectToLogin))
			ssoEcho.Any("/logout", loong.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 这里需要根据不同的用户跳到不同的退出界面上
				values, err := params.LoginManager.GetSession(r)
				if err != nil {
					casClient.RedirectToLogout(w, r)
					return
				}

				if authOpts.UserPrefix != "" {
					if username := values.Get(authclient.SESSION_USER_KEY); !strings.HasPrefix(username, authOpts.UserPrefix) {
						http.Redirect(w, r, urlutil.Join(env.DaemonUrlPath, "/sessions/logout?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
						return
					}
				}
				casClient.RedirectToLogout(w, r)
			}))

			ssoEcho.GET("/login_callback", loong.WrapHandlerFunc(casClient.LoginCallback))
			// ssoEcho.GET(urlutil.Join(casPrefix, "logout_callback"), loong.WrapHandlerFunc(casClient.LogoutCallback))
			logger.Info("cas started")
			return nil
		})
	})
}
