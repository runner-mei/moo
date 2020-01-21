package moo

import (
	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"go.uber.org/fx"
)

func init() {
	var _ *gobatis.SessionFactory = &gobatis.SessionFactory{}
	var _ log.Logger = nil
	var _ = loong.New()
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

	cfg, err := readConfigs(fs, "tpt.", args)
	if err != nil {
		return err
	}

	var opts = []fx.Option{
		fx.Provide(func() interface{} {
			return cfg
		}),
	}
	for _, cb := range initFuncs {
		opts = append(opts, cb())
	}
	app := fx.New(opts...)
	app.Run()
	return app.Err()
}
