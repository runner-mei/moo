package pub

import (
	"log"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/pubsub"
)

func init() {
	moo.On(func() moo.Option {
		return moo.Provide(func(env *moo.Environment, logger log.Logger) (pubsub.Publisher, error) {
			return pubsub.NewHTTPPublisher(urlutil.Join(env.DaemonUrlPath, "pubsub"), logger)
		})
	})
}