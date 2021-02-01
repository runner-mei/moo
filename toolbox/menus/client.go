// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package menus

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/runner-mei/goutils/syncx"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/resty"
)

// Client 菜单服务
type Client interface {
	io.Closer

	WhenChanged(cb func())

	Read(context.Context) ([]Menu, error)

	Flush() error
}

// Callback 菜单的读取函数
type Callback func(context.Context) ([]Menu, error)
type ConvertFunc func(context.Context, map[string][]Menu) []Menu

// appName = env.GetServiceConfig(appID)

// Connect 连接到 weaver 服务
func Connect(logger log.Logger, env *moo.Environment, prx *resty.Proxy, appName, mode, urlPath string, cb Callback, convert ConvertFunc) Client {
	switch mode {
	case "apart":
		apart := &apartClient{
			logger:    logger,
			env:       env,
			prx:       prx,
			appName:   appName,
			urlPath:   urlPath,
			convert:   convert,
			readLocal: cb,
			c:         make(chan struct{}),
		}
		go apart.run()
		// go apart.runSub()
		return apart
	default:
		return &standaloneClient{env: env, readLocal: cb}
	}
}

type standaloneClient struct {
	env       *moo.Environment
	readLocal Callback
}

func (srv *standaloneClient) Close() error {
	return nil
}

func (srv *standaloneClient) Flush() error {
	return nil
}

func (srv *standaloneClient) WhenChanged(cb func()) {
}

func (srv *standaloneClient) Read(ctx context.Context) ([]Menu, error) {
	return srv.readLocal(ctx)
}

type apartResult struct {
	value map[string][]Menu
	err   error
}

type apartClient struct {
	logger log.Logger
	env    *moo.Environment
	prx    *resty.Proxy
	// wsrv           *environment.ServiceConfig
	// appSrv         *environment.ServiceConfig
	appName string
	urlPath string
	// queueName      string
	readLocal Callback
	convert   ConvertFunc

	closed int32
	cw     syncx.CloseWrapper
	pad    int32
	c      chan struct{}
	cached atomic.Value
	mu     sync.Mutex
	cbList []func()
}

func (srv *apartClient) Close() error {
	if atomic.CompareAndSwapInt32(&srv.closed, 0, 1) {
		close(srv.c)
		return srv.cw.Close()
	}
	return nil
}

func (srv *apartClient) save(value map[string][]Menu, err error) {
	srv.cached.Store(&apartResult{
		value: value,
		err:   err,
	})
	srv.mu.Lock()
	defer srv.mu.Unlock()

	for _, cb := range srv.cbList {
		go cb()
	}
}

func (srv *apartClient) Read(ctx context.Context) ([]Menu, error) {
	o := srv.cached.Load()
	if o != nil {
		if result, ok := o.(*apartResult); ok {
			return srv.convert(ctx, result.value), result.err
		}
	}

	value, err := srv.read(ctx)
	srv.save(value, err)
	return srv.convert(ctx, value), err
}

func (srv *apartClient) read(ctx context.Context) (map[string][]Menu, error) {
	var value map[string][]Menu
	req := srv.prx.New(srv.urlPath)
	if srv.appName != "" {
		req = req.SetParam("app", srv.appName)
	}
	err := req.
		Result(&value).
		GET(ctx)
	return value, err
}

func (srv *apartClient) write() (bool, error) {
	value, err := srv.readLocal(context.Background())
	if err != nil {
		return false, err
	}

	req := srv.prx.New(srv.urlPath)
	if srv.appName != "" {
		req = req.SetParam("app", srv.appName)
	}
	return false, req.SetBody(value).
		POST(nil)
}

func (srv *apartClient) WhenChanged(cb func()) {
	if atomic.LoadInt32(&srv.closed) != 0 {
		panic(ErrAlreadyClosed)
	}

	srv.mu.Lock()
	srv.cbList = append(srv.cbList, cb)
	srv.mu.Unlock()
}

func (srv *apartClient) Flush() error {
	if atomic.LoadInt32(&srv.closed) != 0 {
		return ErrAlreadyClosed
	}

	defer recover()
	select {
	case srv.c <- struct{}{}:
	default:
	}
	return nil
}

// func (srv *apartClient) runSub() {
// 	errCount := 0
// 	hubURL := srv.wsrv.URLFor(srv.env.DaemonUrlPath, "/mq/")
// 	builder := hub.Connect(hubURL)

// 	for atomic.LoadInt32(&srv.closed) == 0 {
// 		topic, err := builder.SubscribeTopic(srv.queueName)
// 		if err != nil {
// 			errCount++
// 			if errCount%50 < 3 {
// 				srv.logger.Error("subscribe fail", log.Error(err))
// 			}

// 			select {
// 			case v, ok := <-srv.c:
// 				if ok {
// 					srv.c <- v
// 				}
// 			case <-time.After(1 * time.Second):
// 			}
// 			continue
// 		}
// 		srv.cw.Set(topic)

// 		errCount = 0
// 		err = topic.Run(func(sub *hub.Subscription, msg hub.Message) {
// 			value, err := srv.read(context.Background())
// 			srv.save(value, err)
// 		})
// 		if err != nil {
// 			srv.logger.Error("subscribe fail", log.Error(err))
// 		}
// 		srv.cw.Set(nil)

// 		func() {
// 			defer recover()

// 			select {
// 			case srv.c <- struct{}{}:
// 			default:
// 				srv.logger.Error("failed to send flush event")
// 			}
// 		}()
// 	}
// }

func (srv *apartClient) RunSub(listener moo.EventEmitter) {
	errCount := 0
	for atomic.LoadInt32(&srv.closed) == 0 {
		c, cancel := context.WithCancel(context.Background())
		srv.cw.Set(syncx.ToCloser(cancel))

		err := listener.On(c, func(ctx context.Context, _ string, _ interface{}) {
			value, err := srv.read(ctx)
			srv.save(value, err)
		})
		if err != nil {
			errCount++
			if errCount%50 < 3 {
				srv.logger.Error("subscribe fail", log.Error(err))
			}
		}
		if err = srv.Flush(); err != nil {
			srv.logger.Error("failed to send flush event")
		}
	}
}

func (srv *apartClient) run() {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()
	writed := false

	flush := func() {
		if skipped, err := srv.write(); err != nil {
			srv.logger.Error("write value fail", log.Error(err))
			writed = false
		} else {
			writed = true
			if skipped {
				srv.logger.Warn("write value is skipped", log.Error(err))
			} else {
				srv.logger.Info("write value is ok", log.Error(err))
			}
		}

		value, err := srv.read(context.Background())
		srv.save(value, err)
		if err != nil {
			srv.logger.Error("read value fail", log.Error(err))
			writed = false
		}

		if writed {
			timer.Reset(5 * time.Minute)
		} else {
			timer.Reset(10 * time.Second)
		}
	}

	for atomic.LoadInt32(&srv.closed) == 0 {
		select {
		case _, ok := <-srv.c:
			if !ok {
				return
			}
			flush()
		case <-timer.C:
			flush()
		}
	}
}
