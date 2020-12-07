package moo

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"go.uber.org/fx"
)

var (
	Version   string = "0.1"
	BuildTime string = "N/A"
	GitHash   string = "N/A"
	GoVersion string = "N/A"

	NS                 = "moo"
	DefaultProductName = "moo"
	DefaultURLPath     = "moo"
)

// go build -ldflags "-X 'github.com/runner-mei/moo.GoVersion=$(go version)' -X 'github.com/runner-mei/moo.GitBranch=$(git show -s --format=%H)' -X 'github.com/runner-mei/moo.GitHash=$(git show -s --format=%H)' -X 'github.com/runner-mei/moo.BuildTime=$(git show -s --format=%cd)'"

type Arguments struct {
	Defaults    []string
	Customs     []string
	CommandArgs []string

	PreRun  func(*Environment) error
	Options []Option
}

type Option = fx.Option
type Lifecycle = fx.Lifecycle
type Hook = fx.Hook
type In = fx.In
type Out = fx.Out
type Shutdowner = fx.Shutdowner
type Annotated = fx.Annotated

var Provide = fx.Provide
var Invoke = fx.Invoke
var Supply = fx.Supply
var Populate = fx.Populate

type Closes struct {
	mu      sync.Mutex
	closers []io.Closer
}

func (self *Closes) OnClosing(closers ...io.Closer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.closers = append(self.closers, closers...)
}

func (self *Closes) Close() error {
	self.mu.Lock()
	defer self.mu.Unlock()
	var errList []error
	for _, closer := range self.closers {
		if e := closer.Close(); e != nil {
			errList = append(errList, e)
		}
	}
	if len(errList) == 0 {
		return nil
	}
	if len(errList) == 1 {
		return errList[0]
	}
	return errors.ErrArray(errList)
}

type emptyOut struct {
	Out

	Empty empty `group:"empty"`
}

type empty struct{}

var None = Provide(func() emptyOut {
	return emptyOut{}
})

var initFuncs []func(*Environment) Option

func On(cb func(*Environment) Option) {
	initFuncs = append(initFuncs, cb)
}

type App struct {
	*fx.App
	Environment *Environment
	Closes

	undo func()
}

func (app *App) Start(ctx context.Context) error {
	return app.App.Start(ctx)
}

func (app *App) Stop(ctx context.Context) error {
	defer func() {
		if app.undo != nil {
			app.undo()
			app.undo = nil
		}
	}()

	return app.App.Stop(ctx)
}

func (app *App) Run() error {
	defer func() {
		if app.undo != nil {
			app.undo()
			app.undo = nil
		}
	}()

	app.App.Run()

	return app.App.Err()
}

func NewApp(args *Arguments) (*App, error) {
	for idx, a := range args.CommandArgs {
		if a == "version" {
			fmt.Println("Version=" + Version)
			fmt.Println("BuildTime=" + BuildTime)
			fmt.Println("GitHash=" + GitHash)
			fmt.Println("GoVersion=" + GoVersion)

			copy(args.CommandArgs[:idx], args.CommandArgs[idx+1:])
			args.CommandArgs = args.CommandArgs[:len(args.CommandArgs)-1]
			break
		}
	}

	params, err := ReadCommandLineArgs(args.CommandArgs)
	if err != nil {
		return nil, err
	}

	var namespace = os.Getenv("moo.namespace")
	if params != nil {
		if s := params["moo.namespace"]; s != "" {
			namespace = s
		}
	}
	if namespace == "" {
		namespace = NS
	}

	fs, err := NewFileSystem(namespace, params)
	if err != nil {
		return nil, err
	}

	existnames, nonexistnames, config, err := ReadConfigs(fs, namespace+".", args, params)
	if err != nil {
		return nil, err
	}

	logger, undo, err := NewLogger(config)
	if err != nil {
		return nil, err
	}

	logger.Debug("load config successful",
		log.StringArray("existnames", existnames),
		log.StringArray("nonexistnames", nonexistnames))

	env := NewEnvironment(namespace, config, fs, logger)

	if args.PreRun != nil {
		err := args.PreRun(env)
		if err != nil {
			return nil, err
		}
	}

	app := &App{
		Environment: env,
		undo:        undo,
	}

	var opts = []fx.Option{
		fx.Logger(&LoggerPrinter{logger: logger.Named("fx").AddCallerSkip(3)}),
		fx.Supply(env.Config),
		fx.Supply(env.Fs),
		fx.Provide(func() log.Logger{
			return env.Logger
		}),
		fx.Supply(env),
		fx.Supply(&app.Closes),
		fx.Provide(NewBus),
	}
	if len(args.Options) > 0 {
		opts = append(opts, args.Options...)
	}

	for _, cb := range initFuncs {
		opts = append(opts, cb(env))
	}
	app.App = fx.New(opts...)
	return app, nil
}

func Run(args *Arguments) (rerr error) {
	app, err := NewApp(args)
	if err != nil {
		return err
	}
	defer func() {
		errList := app.Close()
		rerr = errors.Join(rerr, errList)
	}()
	return app.Run()
}
