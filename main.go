package moo

import (
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/cfg"
	"go.uber.org/fx"
)

var NS = "tpt"

type Arguments struct {
	Defaults    []string
	Customs     []string
	CommandArgs []string
}

type Option = fx.Option

var initFuncs []func() Option

func On(cb func() Option) {
	initFuncs = append(initFuncs, cb)
}

func Run(args *Arguments) error {
	params, err := readCommandLineArgs(args.CommandArgs)
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

	config, err := readConfigs(fs, ns+".", args, params)
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

	var opts = []fx.Option{
		fx.Logger(&LoggerPrinter{logger: logger.Named("fx").AddCallerSkip(1)}),
		fx.Provide(func() *cfg.Config {
			return config
		}),
		fx.Provide(func() FileSystem {
			return fs
		}),
		fx.Provide(func() log.Logger {
			return logger
		}),
	}
	for _, cb := range initFuncs {
		opts = append(opts, cb())
	}
	app := fx.New(opts...)
	app.Run()
	return app.Err()
}
