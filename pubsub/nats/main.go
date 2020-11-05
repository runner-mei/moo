package pubsubnats

import (
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/pubsub"
)

func NewPublisher(env *moo.Environment, clientID string, logger log.Logger) (pubsub.Publisher, error) {
	if clientID == "" {
		clientID = "tpt-pub-" + time.Now().Format(time.RFC3339)
	}
	queueURL := env.Config.StringWithDefault(api.CfgPubsubNatsURL, "")
	return NewStreamingPublisher(
		StreamingPublisherConfig{
			URL:       queueURL,
			Marshaler: GobMarshaler{},
			Options: []nats.Option{
				nats.Name(clientID),
			},
		},
		pubsub.NewLoggerAdapter(logger),
	)
}

func NewSubscriber(env *moo.Environment, clientID string, logger log.Logger) (pubsub.Subscriber, error) {
	if clientID == "" {
		clientID = "tpt-sub-" + time.Now().Format(time.RFC3339)
	}
	queueURL := env.Config.StringWithDefault(api.CfgPubsubNatsURL, "")
	queueGroup := env.Config.StringWithDefault(api.CfgPubsubNatsQueueGroup, "")
	subscribersCount := env.Config.IntWithDefault(api.CfgPubsubNatsSubThreads, 10)

	return NewStreamingSubscriber(
		StreamingSubscriberConfig{
			URL:        queueURL,
			QueueGroup: queueGroup,
			// DurableName:      "my-durable",
			SubscribersCount: subscribersCount, // how many goroutines should consume messages
			CloseTimeout:     time.Minute,
			Unmarshaler:      GobMarshaler{},
			Options: []nats.Option{
				nats.Name(clientID),
			},
		},
		pubsub.NewLoggerAdapter(logger),
	)
}
