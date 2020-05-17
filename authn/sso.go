package authn

import (
	"context"
	"net/http"
	"strings"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/authn/services"
)

func init() {
	moo.On(func() moo.Option {
		return moo.Invoke(func(env *moo.Environment, sessions *LoginManager, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {

			casUserPrefix := env.Config.StringWithDefault("users.cas.user_prefix", "")
			sessionPrefix := urlutil.Join(env.DaemonUrlPath, "/sessions")

			sessionuiMux := httpSrv.Engine().Group("/sessions")
			sessionStaticDir := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, "/sessions/static/")
				sessions.StaticDir(ctx, w, r)
			}
			sessionuiMux.GET("/static/*", loong.WrapContextHandler(sessionStaticDir))
			sessionuiMux.GET("", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(sessionPrefix, "login?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			sessionuiMux.GET("/login", loong.WrapContextHandler(sessions.LoginGet))
			sessionuiMux.POST("/login", loong.WrapContextHandler(sessions.LoginPost))
			// sessionuiMux.GET(urlutil.Join(sessionPrefix, "logout"), loong.WrapContextHandler(sessions.Logout))
			sessionuiMux.Any("/logout", loong.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 这里需要根据不同的用户跳到不同的退出界面上
				values, err := sessions.GetSession(r)
				if err != nil {
					sessions.Logout(r.Context(), w, r)
					return
				}

				if casUserPrefix != "" {
					if username := values.Get(authclient.SESSION_USER_KEY); strings.HasPrefix(username, casUserPrefix) {
						http.Redirect(w, r, urlutil.Join(env.DaemonUrlPath, "/cas/logout?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
						return
					}
				}
				sessions.Logout(r.Context(), w, r)
			}))

			sessionuiMux.GET("/captcha", loong.WrapHandler(http.HandlerFunc(services.GenerateCaptchaHandler(nil, sessions.cfg.Captcha))))
			return nil
		})
	})
}

func init() {
	moo.On(func() moo.Option {
		return moo.Invoke(func(env *moo.Environment, sessions *LoginManager, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {
			loginJWTFunc := loong.WrapContextHandler(sessions.LoginJWT)
			web := httpSrv.Engine().Group("api")
			web.GET("/login", loginJWTFunc)
			web.GET("/token", loginJWTFunc)
			web.POST("/login", loginJWTFunc)
			web.POST("/token", loginJWTFunc)

			sessionMux := web.Group("/sessions")
			sessionMux.POST("/", loginJWTFunc)
			sessionMux.POST("", loginJWTFunc)

			listHTTPFunc := loong.RawHTTPAuth(ReturnError, sessions.AuthValidates()...)(sessions.List)
			listFunc := loong.WrapContextHandler(listHTTPFunc)
			sessionMux.GET("/", listFunc)
			sessionMux.GET("", listFunc)

			getHTTPFunc := loong.RawHTTPAuth(ReturnError, sessions.AuthValidates()...)(sessions.Get)
			getFunc := loong.WrapContextHandler(getHTTPFunc)
			sessionMux.GET("/current", getFunc)
			sessionMux.GET("/current/", getFunc)

			getTokenFunc := loong.WrapContextHandler(sessions.GetCurrentToken)
			sessionMux.GET("/current_token", getTokenFunc)
			sessionMux.GET("/current_token/", getTokenFunc)

			logoutFunc := loong.WrapContextHandler(loong.ContextHandlerFunc(sessions.Logout))
			sessionMux.DELETE("/", logoutFunc)
			sessionMux.DELETE("", logoutFunc)

			return nil
		})
	})
}

func init() {
	moo.On(func() moo.Option {
		return moo.Invoke(func(env *moo.Environment, sessions *LoginManager, httpSrv *moo.HTTPServer, logger log.Logger) error {
			ssoEcho := httpSrv.Engine().Group("sso")
			mode := env.Config.StringWithDefault("users.login_url", "sessions")
			redirectPrefix := urlutil.Join(env.DaemonUrlPath, mode)
			ssoEcho.GET("", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(redirectPrefix, "login?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			ssoEcho.GET( "/login", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(redirectPrefix, "login?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			ssoEcho.POST( "/login", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(redirectPrefix, "login?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			ssoEcho.POST( "/logout", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(redirectPrefix, "logout?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			ssoEcho.GET( "/logout", loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, urlutil.Join(redirectPrefix, "logout?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
			}))
			return nil
		})
	})
}
