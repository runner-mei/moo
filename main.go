package moo

import (
	"fmt"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/cfg"
	"go.uber.org/fx"
)

var (
	Version   string = "0.1"
	BuildTime string = "N/A"
	GitHash   string = "N/A"
	GoVersion string = "N/A"
)

// go build -ldflags "-X 'github.com/runner-mei/moo.GoVersion=$(go version)' -X 'github.com/runner-mei/moo.GitBranch=$(git show -s --format=%H)' -X 'github.com/runner-mei/moo.GitHash=$(git show -s --format=%H)' -X 'github.com/runner-mei/moo.BuildTime=$(git show -s --format=%cd)'"

var NS = "moo"

type Arguments struct {
	Defaults    []string
	Customs     []string
	CommandArgs []string
}

type Option = fx.Option
type Lifecycle = fx.Lifecycle
type Hook = fx.Hook
var Provide = fx.Provide
var Invoke = fx.Invoke

var initFuncs []func() Option

func On(cb func() Option) {
	initFuncs = append(initFuncs, cb)
}

// Reset 这个只是用于测试用的
func Reset(newFuncs []func() Option) []func() Option {
	oldFuncs := initFuncs
	initFuncs = newFuncs
	return oldFuncs
}

func Run(args *Arguments) error {
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
		return err
	}
	ns := params["env_prefix"]
	if ns == "" {
		ns = NS
	}

	fs, err := NewFileSystem(params)
	if err != nil {
		return err
	}

	config, err := ReadConfigs(fs, ns+".", args, params)
	if err != nil {
		return err
	}

	logger, undo, err := NewLogger(config)
	if err != nil {
		return err
	}
	if undo != nil {
		defer undo()
	}

	env := NewEnvironment(config, fs, logger)

	var opts = []fx.Option{
		fx.Logger(&LoggerPrinter{logger: logger.Named("fx").AddCallerSkip(3)}),
		fx.Provide(func() *cfg.Config {
			return env.Config
		}),
		fx.Provide(func() FileSystem {
			return env.Fs
		}),
		fx.Provide(func() log.Logger {
			return env.Logger
		}),
		fx.Provide(func() *Environment {
			return env
		}),
	}
	for _, cb := range initFuncs {
		opts = append(opts, cb())
	}
	app := fx.New(opts...)
	app.Run()
	return app.Err()
}
