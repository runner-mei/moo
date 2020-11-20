package pub

import (
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	compnats "github.com/runner-mei/moo/components/nats"
	"github.com/runner-mei/moo/components/pubsub"
	pubsubnats "github.com/runner-mei/moo/components/pubsub/nats"
)

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(env *moo.Environment, noAcks pubsubnats.InNoAcks, factory *compnats.Factory, logger log.Logger) (pubsub.Publisher, error) {
			return pubsubnats.NewPublisher(env, factory, "", noAcks.Names, logger.Named("pubsub"))
		})
	})
}
