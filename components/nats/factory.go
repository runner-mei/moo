package compnats

import (
	"fmt"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
)

func NewFactory(logger log.Logger, queueURL, defaultConnName string) *Factory {
	return &Factory{
		logger:          logger,
		queueURL:        queueURL,
		defaultConnName: defaultConnName,
	}
}

type Factory struct {
	logger   log.Logger
	queueURL string

	defaultlock     sync.Mutex
	defaultConnName string
	defaultconn     *nats.Conn
}

func (f *Factory) Create(logger log.Logger, client string, customopts ...nats.Option) (*nats.Conn, error) {
	opts := make([]nats.Option, 0, 8)

	if logger == nil {
		logger = f.logger
	}
	opts = f.setupConnOptions(logger, opts)
	opts = append(opts, customopts...)

	if client != "" {
		opts = append(opts, nats.Name(client))
	}
	nc, err := nats.Connect(f.queueURL, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "连接 nats 服务器失败")
	}
	return nc, nil
}

func (f *Factory) Default() (*nats.Conn, error) {
	f.defaultlock.Lock()
	defer f.defaultlock.Unlock()

	if f.defaultconn != nil {
		return f.defaultconn, nil
	}
	conn, err := f.Create(nil, f.defaultConnName)
	if err == nil {
		f.defaultconn = conn
	}
	return conn, err
}

func (f *Factory) setupConnOptions(logger log.Logger, opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		if !nc.IsClosed() {
			logger.Warn("Disconnected and will attempt reconnects", log.Error(err), log.Stringer("interval", log.StringerFunc(func() string {
				return fmt.Sprintf("%.0fm", totalWait.Minutes())
			})))
		}
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		logger.Warn("Reconnected", log.Stringer("url", log.StringerFunc(nc.ConnectedUrl)))
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		if !nc.IsClosed() {
			logger.Info("Exiting: no servers available")
		} else {
			logger.Info("Exiting")
		}
	}))
	return opts
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(env *moo.Environment, logger log.Logger) (*Factory, error) {
			queueURL := env.Config.StringWithDefault(api.CfgNatsURL, "")
			if queueURL == "" {
				return nil, errors.New("Nats 服务器参数不正确： URL 为空")
			}

			return &Factory{
				logger:          logger.Named("nats"),
				queueURL:        queueURL,
				defaultConnName: env.Config.StringWithDefault(api.CfgNatsClientName, "moo.client"),
			}, nil
		})
	})
}
