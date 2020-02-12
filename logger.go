package moo

import (
	"github.com/runner-mei/log"
	"github.com/runner-mei/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/runner-mei/goutils/cfg"
)

func NewLogger(cfg *cfg.Config) (log.Logger, func(), error) {
	logConfig := zap.NewProductionConfig()

	var levelErr error
	if levelStr := cfg.StringWithDefault("moo.log.level", ""); levelStr != "" {
		level := logConfig.Level.Level()
		levelErr = level.Set(levelStr)
		if levelErr == nil {
			logConfig.Level.SetLevel(level)
		}
	}

	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := logConfig.Build()
	if err != nil {
		return nil, nil, errors.Wrap(err, "init zap logger fail")
	}
	if name := cfg.StringWithDefault("log.name", ""); name != "" {
		logger = logger.Named(name)
	}

	if levelErr != nil {
		logger.Warn("set level fail", log.Error(levelErr))
	}

	zap.ReplaceGlobals(logger)

	var undoRedirectStdLog func()
	if enabled := cfg.BoolWithDefault("log.redirect_std_log", true); enabled {
		undoRedirectStdLog = zap.RedirectStdLog(logger)
	}
	return log.NewLogger(logger), undoRedirectStdLog, nil
}


type LoggerPrinter struct {
	logger log.Logger
}


func (p *LoggerPrinter) Printf(format string, args ...interface{}) {
	p.logger.Infof(format, args...)
}
