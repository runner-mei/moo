package sub

import (
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/pubsub"
	pubsubnats "github.com/runner-mei/moo/pubsub/nats"
)

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(env *moo.Environment, noAcks pubsubnats.InNoAcks, logger log.Logger) (pubsub.Subscriber, error) {
			return pubsubnats.NewSubscriber(env, "", noAcks.Names, logger.Named("pubsub"))
		})
	})
}
