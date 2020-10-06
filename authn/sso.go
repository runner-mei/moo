package authn

import (
	"context"
	"net/http"
	"strings"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/authn/services"
)

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(env *moo.Environment, sessions *LoginManager, httpSrv *moo.HTTPServer, middlewares moo.Middlewares, logger log.Logger) error {
			casUserPrefix := env.Config.StringWithDefault(api.CfgUserCasUserPrefix, "")
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

	moo.On(func(*moo.Environment) moo.Option {
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

			signature := loong.WrapContextHandler(loong.ContextHandlerFunc(sessions.Signature))
			sessionMux.GET("/signature", signature)

			return nil
		})
	})

	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(env *moo.Environment, sessions *LoginManager, httpSrv *moo.HTTPServer, logger log.Logger) error {
			ssoEcho := httpSrv.Engine().Group("sso")
			mode := env.Config.StringWithDefault(api.CfgUserLoginURL, "sessions")

			if mode == "ca" {
				redirectPrefix := urlutil.Join(env.DaemonUrlPath, "sessions")

				redirectURL := env.Config.StringWithDefault(api.CfgUserRedirectTo,
					urlutil.Join(redirectPrefix, "login"))
				if redirectURL != "" {
					redirectURL = strings.Replace(redirectURL, "\\$\\{appRoot}", env.DaemonUrlPath, -1)
					redirectURL = strings.Replace(redirectURL, "${appRoot}", env.DaemonUrlPath, -1)
				}

				cb := loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery != "" {
						http.Redirect(w, r, redirectURL+"?"+r.URL.RawQuery, http.StatusTemporaryRedirect)
						return
					}
					http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				})
				ssoEcho.GET("", cb)
				ssoEcho.GET("/login", cb)
				ssoEcho.POST("/login", cb)
				ssoEcho.POST("/logout", cb)
				ssoEcho.GET("/logout", cb)
			} else {
				redirectPrefix := urlutil.Join(env.DaemonUrlPath, mode)

				cb := loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery == "" {
						http.Redirect(w, r, urlutil.Join(redirectPrefix, "login"), http.StatusTemporaryRedirect)
						return
					}

					http.Redirect(w, r, urlutil.Join(redirectPrefix, "login?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
				})
				ssoEcho.GET("", cb)
				ssoEcho.GET("/login", cb)
				ssoEcho.POST("/login", cb)

				logoutCb := loong.WrapContextHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery == "" {
						http.Redirect(w, r, urlutil.Join(redirectPrefix, "logout"), http.StatusTemporaryRedirect)
						return
					}

					http.Redirect(w, r, urlutil.Join(redirectPrefix, "logout?"+r.URL.RawQuery), http.StatusTemporaryRedirect)
				})
				ssoEcho.POST("/logout", logoutCb)
				ssoEcho.GET("/logout", logoutCb)
			}
			return nil
		})
	})
}
