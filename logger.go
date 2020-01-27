package moo

import (
	"github.com/runner-mei/log"
	"github.com/runner-mei/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/fx"
	"github.com/runner-mei/moo/cfg"
)


func init() {
	On(func() Option {
		var undoRedirectStdLog func()
		return fx.Provide(func(cfg *cfg.Config) (log.Logger, error) {
			logConfig := zap.NewProductionConfig()
			logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			logger, err := logConfig.Build()
			if err != nil {
				return nil, errors.Wrap(err, "init zap logger fail")
			}
			if name := cfg.StringWithDefault("log.name", ""); name != "" {
				logger = logger.Named(name)
			}

			zap.ReplaceGlobals(logger)
		

			if enabled := cfg.BoolWithDefault("log.redirect_std_log", true); enabled {
				if undoRedirectStdLog != nil {
					undoRedirectStdLog()
				}
				undoRedirectStdLog = zap.RedirectStdLog(logger)
			}
			return log.NewLogger(logger), nil
		})
	})
}

