package moo_tests

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/resty"
)

type httpLifecycle struct {
	*TestApp
}

func (l *httpLifecycle) OnHTTP(addr string) {
	l.TestApp.ListenAt = addr
	select {
	case l.httpOK <- nil:
	}
}
func (l *httpLifecycle) OnHTTPs(addr string) {
	l.TestApp.SListenAt = addr
	select {
	case l.httpOK <- nil:
	}
}

type TestApp struct {
	started bool
	App     *moo.App
	closers []io.Closer

	Env         *moo.Environment
	Args        moo.Arguments
	HTTPServer  *moo.HTTPServer
	UserManager authn.UserManager
	cfg         *authn.Config

	HttpPort  string
	HttpsPort string
	ListenAt  string
	SListenAt string
	URL       string

	httpOK chan error
}

func (a *TestApp) Read(value interface{}) {
	a.Args.Options = append(a.Args.Options, moo.Populate(value))
}

func (a *TestApp) CreateUser(t testing.TB, name, password string, attributes ...map[string]interface{}) int64 {
	ctx := context.Background()

	user, err := a.UserManager.UserByName(ctx, api.UserBgOperator)
	if err != nil {
		t.Fatal("load"+api.UserBgOperator+" fail", err)
	}
	ctx = api.ContextWithUser(ctx, user)

	var params = map[string]interface{}{}
	if len(attributes) > 0 {
		params = attributes[0]
	}
	userid, err := a.UserManager.Create(ctx,
		name, name, "", password, params, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return userid.(int64)
}

func (a *TestApp) CreateCookieWithUsername(t testing.TB, username string) *http.Cookie {
	queryParams := url.Values{}
	queryParams.Set(authclient.SESSION_EXPIRE_KEY, "session")
	queryParams.Set(authclient.SESSION_VALID_KEY, "true")
	queryParams.Set(authclient.SESSION_USER_KEY, username)

	value := authclient.Encode(queryParams, a.cfg.GetSessionHashFunc(), a.cfg.SessionSecretKey)
	return &http.Cookie{
		Name:  a.cfg.SessionKey,
		Value: value,
		Path:  a.cfg.SessionPath,
	}
}

func (app *TestApp) GetAuthFuncForCookie(t testing.TB, username string) resty.AuthFunc {
	value := app.CreateCookieWithUsername(t, username)
	return func(ctx context.Context, r *resty.Request, force bool) (*resty.Request, error) {
		return r.AddCookie(value), nil
	}
}

func (a *TestApp) Close() error {
	// if a.shutdowner != nil {
	// 	err := a.shutdowner.Shutdown()
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// }

	stopCtx, cancel := context.WithTimeout(context.Background(), a.App.StopTimeout())
	defer cancel()

	err1 := a.App.Stop(stopCtx)
	err2 := a.App.Close()
	return errors.Join(err1, err2)
}

func (a *TestApp) OnClosing(closer io.Closer) {
	if a.App == nil {
		a.closers = append(a.closers, closer)
		return
	}
	a.App.OnClosing(closer)
}

func (a *TestApp) IsStarted() bool {
	return a.started
}

func (a *TestApp) Start(t testing.TB) {
	if !a.started {
		a.started = true
	}
	a.init()

	found := false
	for _, s := range a.Args.CommandArgs {
		if strings.HasPrefix(s, "http-address=") {
			found = true
			break
		}
	}
	if !found {
		a.Args.CommandArgs = append(a.Args.CommandArgs, "http-address=:")
	}
	a.Args.CommandArgs = append(a.Args.CommandArgs, "operation_logger=2")

	if a.httpOK == nil {
		a.httpOK = make(chan error, 3)
	}

	//a.Args.Options = append(a.Args.Options, fx.Populate(&a.shutdowner))
	//a.Args.Options = append(a.Args.Options, fx.Populate(&a.Env))
	a.Args.Options = append(a.Args.Options,
		moo.Provide(func() moo.HTTPLifecycle {
			return &httpLifecycle{
				TestApp: a,
			}
		}))
	a.Read(&a.HTTPServer)
	a.Read(&a.UserManager)
	a.Read(&a.cfg)

	var err error
	a.App, err = moo.NewApp(&a.Args)
	if err != nil {
		t.Error(err)
		t.FailNow()
		return
	}
	a.Env = a.App.Environment
	a.Env.RunMode = moo.TestRunMode

	for _, closer := range a.closers {
		a.App.OnClosing(closer)
	}
	a.closers = nil

	startCtx, cancel := context.WithTimeout(context.Background(), a.App.StartTimeout())
	defer cancel()
	if err := a.App.Start(startCtx); err != nil {
		t.Errorf("ERROR\t\tFailed to start: %v", err)
		t.FailNow()
		return
	}
	select {
	case <-a.httpOK:
	case <-startCtx.Done():
		t.Error("ERROR\t\tFailed to start: timeout")
		t.FailNow()
		return
	}

	if a.ListenAt != "" {
		_, port, err := net.SplitHostPort(a.ListenAt)
		if err != nil {
			t.Error(err)
			t.FailNow()
			return
		}
		a.HttpPort = port
		a.URL = "http://127.0.0.1:" + port
	} else if a.SListenAt != "" {
		_, port, err := net.SplitHostPort(a.SListenAt)
		if err != nil {
			t.Error(err)
			t.FailNow()
			return
		}

		a.HttpsPort = port
		a.URL = "https://127.0.0.1:" + port
	} else {

		t.Error("im")
		t.FailNow()
	}
}

func NewTestApp(t testing.TB) *TestApp {
	// oldInitFuncs := moo.Reset(nil)
	// moo.Reset(oldInitFuncs)
	// defer moo.Reset(oldInitFuncs)

	return &TestApp{
		// oldInitFuncs: oldInitFuncs,
		// HttpOK: make(chan error, 3),
		Args: moo.Arguments{
			CommandArgs: []string{
				moo.NS + ".runMode=" + moo.TestRunMode,
				"users.version=2",
				api.CfgOperationLoggerVersion + "=2",
				"users.redirect_mode=code",
			},
		},
	}
}

func NewEnvironment(t testing.TB, args *moo.Arguments) *moo.Environment{
	if args == nil {
		args = &moo.Arguments{
			CommandArgs: []string{
				moo.NS + ".runMode=" + moo.TestRunMode,
				"users.version=2",
				api.CfgOperationLoggerVersion + "=2",
				"users.redirect_mode=code",
			},
		}
	}

	app, err := moo.NewApp(args)
	if err != nil {
		t.Error(err)
		t.FailNow()
		return nil
	}
	return app.Environment
}