package sub

import (
	"net/http"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/components/pubsub"
)

type SubscribeHttpHandler struct {
	Handler http.Handler
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(env *moo.Environment, logger log.Logger) (pubsub.Subscriber, *SubscribeHttpHandler, error) {
			handler, subscriber, err := pubsub.NewHTTPSubscriber(urlutil.Join(env.DaemonUrlPath, "pubsub"), logger)
			if err != nil {
				return nil, nil, err
			}
			return subscriber, &SubscribeHttpHandler{Handler: handler}, nil
		})
	})

	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(httpSrv *moo.HTTPServer, handler *SubscribeHttpHandler) error {
			httpSrv.FastRoute(true, "pubsub", handler.Handler)
			return nil
		})
	})
}
