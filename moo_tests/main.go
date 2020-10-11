package moo_tests

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/runner-mei/moo"
	_ "github.com/runner-mei/moo/authn/sessions/inmem"
	"go.uber.org/fx"
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
	App *moo.App
	// oldInitFuncs []func() moo.Option
	closers []io.Closer
	// shutdowner fx.Shutdowner

	Env  *moo.Environment
	Args moo.Arguments

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

func (a *TestApp) Start(t testing.TB) {
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
		fx.Provide(func() moo.HTTPLifecycle {
			return &httpLifecycle{
				TestApp: a,
			}
		}))

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
				"moo.operation_logger=2",
				"users.redirect_mode=code",
			},
		},
	}
}
