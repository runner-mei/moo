package moo

import (
	"context"
	"net"
	"net/http"
	nhttputil "net/http/httputil"
	_ "net/http/pprof"
	"net/url"
	"strconv"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	bgo "github.com/digitalcrab/browscap_go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/loong/jaeger"
	"github.com/runner-mei/moo/api"
)

type InAuthFuncs struct {
	In

	Funcs []loong.AuthValidateFunc `group:"authValidate"`
}

type InHttpTracing struct {
	In

	Tracer opentracing.Tracer `optional:"true"`
}


type HTTPLifecycle interface {
	OnHTTP(addr string)
	OnHTTPs(addr string)
}

type InHTTPLifecycle struct {
	In

	Lifecycle HTTPLifecycle `optional:"true"`
}

type Middlewares struct {
	In

	Funcs []loong.MiddlewareFunc `group:"middlewares"`
}

type InAddress struct {
	In

	HttpFunc  func() (string, string, error) `name:"http-address" optional:"true"`
	HttpsFunc func() (string, string, error) `name:"https-address" optional:"true"`
}

type OutAddress struct {
	Out

	HttpFunc  func() (string, string, error) `name:"http-address"`
	HttpsFunc func() (string, string, error) `name:"https-address"`
}

func init() {
	loong.ContextWithUserHook = func(ctx context.Context, u interface{}) context.Context {
		return context.WithValue(ctx, api.UserKey, u)
	}
	loong.UserFromContextHook = func(ctx context.Context) interface{} {
		if ctx == nil {
			return nil
		}
		return ctx.Value(api.UserKey)
	}

	On(func(*Environment) Option {
		return Provide(func(env *Environment, logger log.Logger, authFuncs InAuthFuncs, tracer InHttpTracing) *HTTPServer {
			httpSrv := &HTTPServer{
				logger:     env.Logger.Named("http"),
				noPrefix:   env.DaemonUrlPath == "" || env.DaemonUrlPath == "/",
				homePrefix: strings.TrimSuffix(env.DaemonUrlPath, "/") + "/",
				trimPrefix: strings.TrimSuffix(env.DaemonUrlPath, "/"),
				engine:     loong.New(),
				fastRoutes: map[string]FastHandlerFunc{},
				homePage:   urlutil.JoinURLPath(env.DaemonUrlPath, "home/"),
				authFuncs:  authFuncs.Funcs,
			}
			httpSrv.engine.Logger = httpSrv.logger


			for _, file := range []string{
				env.Fs.FromData("resources", "favicon.ico"),
				env.Fs.FromInstallRoot("web", "resources", "favicon.ico"),
			} {
				if fileExists(file, nil) {
					httpSrv.faviconFile = file
					break
				}
			}

			if tracer.Tracer != nil {
				tracer := loong.Tracing(tracer.Tracer, "moo", false)
				httpSrv.engine.Use(tracer)

				httpSrv.logger.Named("opentracing").Info("opentracing is enabled")
			} else if env.Config.BoolWithDefault("opentracing", false) {
				jaeger.Init("wserver", httpSrv.logger.Named("jaegertracing").Unwrap())
				tracer := loong.Tracing(opentracing.GlobalTracer(), "moo", false)
				httpSrv.engine.Use(tracer)

				httpSrv.logger.Named("opentracing").Info("opentracing is enabled")
			} else {
				httpSrv.logger.Named("opentracing").Info("opentracing is disabled")
			}
			return httpSrv
		})
	})

	On(func(*Environment) Option {
		return Invoke(func(lifecycle Lifecycle, env *Environment, httpSrv *HTTPServer, inAddress InAddress, httpLifecycle InHTTPLifecycle) error {
			var noListen = true

			if inAddress.HttpFunc == nil {
				inAddress.HttpFunc = func() (string, string, error) {
					return env.Config.StringWithDefault(api.CfgHTTPNetwork, "tcp"),
						env.Config.StringWithDefault(api.CfgHTTPAddress, ""),
						nil
				}
			}
			if inAddress.HttpsFunc == nil {
				inAddress.HttpsFunc = func() (string, string, error) {
					return env.Config.StringWithDefault(api.CfgHTTPSNetwork, "tcp"),
						env.Config.StringWithDefault(api.CfgHTTPSAddress, ""),
						nil
				}
			}

			if env.Config.BoolWithDefault(api.CfgHTTPEnabled, true) {
				httpNetwork, httpListenAt, err := inAddress.HttpFunc()
				if err != nil {
					return err
				}
				if httpListenAt != "" {
					var hsrv *http.Server
					var listener net.Listener

					lifecycle.Append(Hook{
						OnStart: func(context.Context) error {
							httpSrv.logger.Info("http listen at: " + httpNetwork + "+" + httpListenAt)

							hsrv = &http.Server{Addr: httpListenAt, Handler: httpSrv}
							ln, err := netutil.Listen(httpNetwork, httpListenAt)
							if err != nil {
								return err
							}
							listener = ln

							go func() {
								tcpListener, ok := listener.(*net.TCPListener)
								if ok {
									listener = httputil.TcpKeepAliveListener{tcpListener}
								}
								err := hsrv.Serve(listener)
								if err != nil {
									if http.ErrServerClosed != err {
										httpSrv.logger.Error("start unsuccessful", log.Error(err))
									} else {
										httpSrv.logger.Info("stopped")
									}
								}
							}()

							if httpLifecycle.Lifecycle != nil {
								if httpListenAt == ":" || httpListenAt == ":0" || httpListenAt == "0.0.0.0:0" {
									httpLifecycle.Lifecycle.OnHTTP(listener.Addr().String())
								} else {
									httpLifecycle.Lifecycle.OnHTTP(httpListenAt)
								}
							}

							return nil
						},
						OnStop: func(context.Context) error {
							err := hsrv.Close()
							listener.Close()
							return err
						},
					})
					noListen = false
				}
			}

			if env.Config.BoolWithDefault(api.CfgHTTPSEnabled, true) {
				httpsNetwork, httpsListenAt, err := inAddress.HttpsFunc()
				if err != nil {
					return err
				}
				if httpsListenAt != "" {
					var hsrv *http.Server
					var listener net.Listener

					lifecycle.Append(Hook{
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

							httpSrv.logger.Info("https listen at: " + httpsNetwork + "+" + httpsListenAt)

							hsrv = &http.Server{Addr: httpsListenAt, Handler: httpSrv}
							ln, err := netutil.Listen(httpsNetwork, httpsListenAt)
							if err != nil {
								return err
							}
							listener = ln

							go func() {
								tcpListener, ok := listener.(*net.TCPListener)
								if ok {
									listener = httputil.TcpKeepAliveListener{tcpListener}
								}
								err := hsrv.ServeTLS(listener, certFile, keyFile)
								if err != nil {
									if http.ErrServerClosed != err {
										httpSrv.logger.Error("start unsuccessful", log.Error(err))
									} else {
										httpSrv.logger.Info("stopped")
									}
								}
							}()

							if httpLifecycle.Lifecycle != nil {
								if httpsListenAt == ":" || httpsListenAt == ":0" || httpsListenAt == "0.0.0.0:0" {
									httpLifecycle.Lifecycle.OnHTTPs(listener.Addr().String())
								} else {
									httpLifecycle.Lifecycle.OnHTTPs(httpsListenAt)
								}
							}

							return nil
						},
						OnStop: func(context.Context) error {
							err := hsrv.Close()
							listener.Close()
							return err
						},
					})

					noListen = false
				}
			}

			if noListen {
				env.Logger.Info("http not listen, http-address or https-address is missing")
			}
			return nil
		})
	})
}

type FastHandler interface {
	Serve(w http.ResponseWriter, r *http.Request, pa string)
}

type FastHandlerFunc func(w http.ResponseWriter, r *http.Request, pa string)

func (fn FastHandlerFunc) Serve(w http.ResponseWriter, r *http.Request, pa string) {
	fn(w, r, pa)
}

type HttpHandlerFunc func(w http.ResponseWriter, r *http.Request)

func (fn HttpHandlerFunc) Serve(w http.ResponseWriter, r *http.Request, pa string) {
	r.URL.Path = pa
	fn(w, r)
}

type HTTPServer struct {
	logger      log.Logger
	noPrefix    bool
	trimPrefix  string
	homePrefix  string
	homePage    string
	faviconFile string

	engine     *loong.Engine
	fastRoutes map[string]FastHandlerFunc

	authFuncs []loong.AuthValidateFunc
}

func (srv *HTTPServer) AuthMiddlewares() loong.MiddlewareFunc {
	return loong.HTTPAuth(srv.authFuncs...)
}

func (srv *HTTPServer) Engine() *loong.Engine {
	return srv.engine
}

func (srv *HTTPServer) AddFastHandler(name string, handler FastHandlerFunc) {
	srv.fastRoutes[name] = handler
}

func (srv *HTTPServer) FastRoute(stripPrefix bool, name string, handler http.Handler) {
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimPrefix(name, "/")
	if strings.ContainsRune(name, '/') {
		panic(errors.New("'" + name + "' is invalid fast urlpath, it must not contains '/'"))
	}

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
	var pa = r.URL.Path
	if !srv.noPrefix {
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

		pa = strings.TrimPrefix(r.URL.Path, srv.trimPrefix)
		if pa == "" || pa == "/" {
			http.Redirect(w, r, srv.homePage, http.StatusTemporaryRedirect)
			return
		}
	}
	name, urlPath := urlutil.SplitURLPath(pa)
	if h, exists := srv.fastRoutes[name]; exists {
		h(w, r, urlPath)
	} else {
		r.URL.Path = pa
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
