package moo_tests

import (
	"io"
	"log"
	"net"
	"strings"
	"testing"

	"github.com/runner-mei/moo"
	_ "github.com/runner-mei/moo/auth/sessions/inmem"
	"go.uber.org/fx"
)

type httpLifecycle struct {
	*AppTest
}

func (l *httpLifecycle) OnHTTP(addr string) {
	l.AppTest.ListenAt = addr
	l.HttpOK <- nil
}
func (l *httpLifecycle) OnHTTPs(addr string) {
	l.AppTest.SListenAt = addr
}

type AppTest struct {
	oldInitFuncs []func() moo.Option
	closers      []io.Closer
	shutdowner   fx.Shutdowner

	Env  *moo.Environment
	Args moo.Arguments

	ListenAt  string
	SListenAt string
	HttpOK    chan error

	URL string
}

func (a *AppTest) Close() error {
	if a.shutdowner != nil {
		err := a.shutdowner.Shutdown()
		if err != nil {
			log.Println(err)
		}
	}

	for _, closer := range a.closers {
		closer.Close()
	}

	close(a.HttpOK)
	a.HttpOK = nil

	moo.Reset(a.oldInitFuncs)
	return nil
}

func (a *AppTest) OnClosing(closer io.Closer) {
	a.closers = append(a.closers, closer)
}

func (a *AppTest) Start(t testing.TB) {
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

	if a.HttpOK == nil {
		a.HttpOK = make(chan error, 3)
	}

	moo.On(func() moo.Option {
		return fx.Populate(&a.shutdowner)
	})
	moo.On(func() moo.Option {
		return fx.Populate(&a.Env)
	})
	moo.On(func() moo.Option {
		return fx.Provide(func() moo.HTTPLifecycle {
			return &httpLifecycle{
				AppTest: a,
			}
		})
	})

	go func() {
		err := moo.Run(&a.Args)
		if err != nil {
			t.Error(err)
		}
		select {
		case a.HttpOK <- err:
		default:
		}
	}()

	err := <-a.HttpOK
	if err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	_, port, err := net.SplitHostPort(a.ListenAt)
	if err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	a.URL = "http://127.0.0.1:" + port
}

func NewAppTest(t testing.TB) *AppTest {
	oldInitFuncs := moo.Reset(nil)
	moo.Reset(oldInitFuncs)
	defer moo.Reset(oldInitFuncs)

	return &AppTest{
		oldInitFuncs: oldInitFuncs,
		HttpOK:       make(chan error, 3),
		Args: moo.Arguments{
			CommandArgs: []string{
				"users.version=2",
				"moo.operation_logger=2",
				"users.redirect_mode=code",
			},
		},
	}
}
