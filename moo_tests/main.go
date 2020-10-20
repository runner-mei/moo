package moo_tests

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
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
	// oldInitFuncs []func() moo.Option
	closers []io.Closer
	// shutdowner fx.Shutdowner

	Env         *moo.Environment
	Args        moo.Arguments
	HTTPServer  *moo.HTTPServer
	UserManager authn.UserManager

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

func (a *TestApp) CreateUser(t testing.TB, name, password string, attributes ...map[string]interface{}) interface{} {
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
		name, name, "", password, params, nil)
	if err != nil {
		t.Fatal(err)
	}
	return userid
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

	err := a.App.Stop(stopCtx)
	for _, closer := range a.closers {
		closer.Close()
	}
	// moo.Reset(a.oldInitFuncs)
	return err
}

func (a *TestApp) OnClosing(closer io.Closer) {
	a.closers = append(a.closers, closer)
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

	var err error
	a.App, err = moo.NewApp(&a.Args)
	if err != nil {
		t.Error(err)
		t.FailNow()
		return
	}
	a.Env = a.App.Environment

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
				"moo.runMode=dev",
				"users.version=2",
				api.CfgOperationLoggerVersion + "=2",
				"users.redirect_mode=code",
			},
		},
	}
}
