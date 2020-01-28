package moo

import (
	"context"
	"net"
	"net/http"
	nhttputil "net/http/httputil"
	"net/url"	
	"strconv"
	"strings"

	bgo "github.com/digitalcrab/browscap_go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/loong/jaeger"
	"go.uber.org/fx"
)

func init() {
	On(func() Option {
		return fx.Provide(func(env *Environment, logger log.Logger) *HTTPServer {
			httpSrv := &HTTPServer{
				logger:     env.Logger.Named("http"),
				homePrefix: env.DaemonUrlPath,
				engine:     loong.New(),
				fastRoutes: map[string]HandlerFunc{},
				homePage:   urlutil.JoinURLPath(env.DaemonUrlPath, "home/"),
			}
			for _, file := range []string{
				env.Fs.FromData("resources", "favicon.ico"),
				env.Fs.FromInstallRoot("web", "resources", "favicon.ico"),
			} {
				if fileExists(file, nil) {
					httpSrv.faviconFile = file
					break
				}
			}

			if env.Config.BoolWithDefault("opentracing", false) {
				jaeger.Init("wserver", httpSrv.logger.Named("jaegertracing").Unwrap())
				httpSrv.logger.Named("jaegertracing").Info("opentracing is enabled")
			} else {
				httpSrv.logger.Named("jaegertracing").Info("opentracing is disabled")
			}
			return httpSrv
		})
	})

	On(func() Option {
		return fx.Invoke(func(lifecycle fx.Lifecycle, env *Environment, httpSrv *HTTPServer) error {
			if listenAt := env.Config.StringWithDefault("http-address", ""); listenAt != "" {
				var hsrv *http.Server
				var listener net.Listener

				lifecycle.Append(fx.Hook{
					OnStart: func(context.Context) error {
						network := env.Config.StringWithDefault("http-network", "tcp")
						httpSrv.logger.Info("http listen at: " + network + "+"+ listenAt)

						hsrv = &http.Server{Addr: listenAt, Handler: httpSrv}
						ln, err := netutil.Listen(network, listenAt)
						if err != nil {
							return err
						}
						listener = ln

						go func() {
							tcpListener, ok := listener.(*net.TCPListener)
							if ok {
								listener = httputil.TcpKeepAliveListener{tcpListener}
							}
							err :=  hsrv.Serve(listener)
							if err != nil {
								if http.ErrServerClosed != err {
									httpSrv.logger.Error("start unsuccessful", log.Error(err))
								} else {
									httpSrv.logger.Info("stopped")
								}
							}
						}()
						return nil
					},
					OnStop: func(context.Context) error {
						err := hsrv.Close()
						listener.Close()
						return err
					},
				})
			}

			if listenAt := env.Config.StringWithDefault("https-address", ""); listenAt != "" {
				var hsrv *http.Server
				var listener net.Listener

				lifecycle.Append(fx.Hook{
					OnStart: func(context.Context) error {
						var certFile, keyFile string
						for _, file := range []string{
							env.Fs.FromConfig("cert.pem"),
							env.Fs.FromDataConfig("cert.pem"),
						} {
							if fileExists(file, nil) {
								certFile = file
								break
							}
						}
						for _, file := range []string{
							env.Fs.FromConfig("key.pem"),
							env.Fs.FromDataConfig("key.pem"),
						} {
							if fileExists(file, nil) {
								keyFile = file
								break
							}
						}
						if keyFile == "" || certFile == "" {
							return errors.New("keyFile or certFile isn't found")
						}

						network := env.Config.StringWithDefault("https-network", "tcp")
						httpSrv.logger.Info("https listen at: " + network + "+"+ listenAt)

						hsrv = &http.Server{Addr: listenAt, Handler: httpSrv}
						ln, err := netutil.Listen(network, listenAt)
						if err != nil {
							return err
						}
						listener = ln

						go func() {
							tcpListener, ok := listener.(*net.TCPListener)
							if ok {
								listener = httputil.TcpKeepAliveListener{tcpListener}
							}
							err :=  hsrv.ServeTLS(listener, certFile, keyFile)
							if err != nil {
								if http.ErrServerClosed != err {
									httpSrv.logger.Error("start unsuccessful", log.Error(err))
								} else {
									httpSrv.logger.Info("stopped")
								}
							}
						}()
						return nil
					},
					OnStop: func(context.Context) error {
						err := hsrv.Close()
						listener.Close()
						return err
					},
				})
			}
			return nil
		})
	})
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, pa string)

type HTTPServer struct {
	logger      log.Logger
	homePrefix  string
	homePage    string
	faviconFile string

	engine     *loong.Engine
	fastRoutes map[string]HandlerFunc
}

func (srv *HTTPServer) Engine() *loong.Engine {
	return srv.engine
}

func (srv *HTTPServer) FastRoute(stripPrefix bool, name string, handler http.Handler) {
	if stripPrefix {
		srv.fastRoutes[name] = func(w http.ResponseWriter, r *http.Request, pa string) {
			r.URL.Path = pa
			handler.ServeHTTP(w, r)
		}
	} else {
		srv.fastRoutes[name] = func(w http.ResponseWriter, r *http.Request, pa string) {
			handler.ServeHTTP(w, r)
		}
	}
}

func (srv *HTTPServer) RouteProxy(stripPrefix bool, name, urlstr string) {
	u, err := url.Parse(urlstr)
	if err != nil {
		srv.logger.Fatal("add proxy fail", log.String("name", name), log.String("url", urlstr), log.Error(err))
	}
	handler := nhttputil.NewSingleHostReverseProxy(u)
	srv.FastRoute(stripPrefix, name, handler)
}

func (srv *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, srv.homePrefix) {
		if r.URL.Path == "/favicon.ico" {
			if srv.faviconFile != "" {
				http.ServeFile(w, r, srv.faviconFile)
				return
			}
			http.NotFound(w, r)
			return
		}

		BrowserCheckFunc(srv.logger, srv.homePrefix, w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "" ||
				r.URL.Path == "/" ||
				srv.homePrefix == r.URL.Path ||
				srv.homePrefix == r.URL.Path+"/" ||
				srv.homePage == r.URL.Path+"/" {
				http.Redirect(w, r, srv.homePage, http.StatusTemporaryRedirect)
				return
			}
			http.DefaultServeMux.ServeHTTP(w, r)
		}))
		return
	}

	pa := strings.TrimPrefix(r.URL.Path, srv.homePrefix)
	name, urlPath := urlutil.SplitURLPath(pa)
	if h, exists := srv.fastRoutes[name]; exists {
		h(w, r, urlPath)
	} else {
		r.URL.Path = urlPath
		srv.engine.ServeHTTP(w, r)
	}
}

func BrowserCheck(logger log.Logger, appRoot string, next loong.ContextHandlerFunc) loong.ContextHandlerFunc {
	return loong.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		browser, _ := bgo.GetBrowser(r.UserAgent())
		if browser != nil && browser.Browser == "IE" {
			versions := strings.Split(browser.BrowserVersion, ".")
			version, err := strconv.ParseInt(versions[0], 10, 64)
			if err != nil {
				logger.Warn("read browser version fail", log.Error(err))
			} else if version < 11 {
				urlStr := urlutil.JoinURLPath(appRoot, "/internal/misc/browser_compatibility.html")
				http.Redirect(w, r, urlStr, http.StatusTemporaryRedirect)
				return
			}
		}

		next(ctx, w, r)
	})
}

func BrowserCheckFunc(logger log.Logger, appRoot string, w http.ResponseWriter, r *http.Request, cb http.HandlerFunc) {
	browser, _ := bgo.GetBrowser(r.UserAgent())
	if browser != nil && browser.Browser == "IE" {
		versions := strings.Split(browser.BrowserVersion, ".")
		version, err := strconv.ParseInt(versions[0], 10, 64)
		if err != nil {
			logger.Warn("read browser version fail", log.Error(err))
		} else if version < 11 {
			urlStr := urlutil.JoinURLPath(appRoot, "/internal/misc/browser_compatibility.html")
			http.Redirect(w, r, urlStr, http.StatusTemporaryRedirect)
			return
		}
	}

	cb(w, r)
}
