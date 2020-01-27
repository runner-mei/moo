package moo

import (
	gobatis "github.com/runner-mei/GoBatis"
	"go.uber.org/fx"

	"github.com/runner-mei/moo/cfg"
)

var NS = "tpt"

func init() {
	var _ *gobatis.SessionFactory = &gobatis.SessionFactory{}
}

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

	fs, err := NewFileSystem(params)
	if err != nil {
		return err
	}

	config, err := readConfigs(fs, NS+".", args)
	if err != nil {
		return err
	}

	var opts = []fx.Option{
		fx.Provide(func() *cfg.Config {
			return config
		}),
	}
	for _, cb := range initFuncs {
		opts = append(opts, cb())
	}
	app := fx.New(opts...)
	app.Run()
	return app.Err()
}
