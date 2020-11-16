package pubsubnats

import (
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/pubsub"
)

type OutNoAck struct {
	moo.Out

	Name string `group:"nats_noack"`
}

type InNoAcks struct {
	moo.In

	Names []string `group:"nats_noack"`
}

func NoAck(name string) OutNoAck {
	return OutNoAck{Name: name}
}

func NewPublisher(env *moo.Environment, clientID string, noAcks []string, logger log.Logger) (pubsub.Publisher, error) {
	if clientID == "" {
		clientID = "tpt-pub-" + time.Now().Format(time.RFC3339)
	}
	queueURL := env.Config.StringWithDefault(api.CfgPubsubNatsURL, "")
	if queueURL == "" {
		return nil, errors.New("Nats 服务器参数不正确： URL 为空")
	}
	return NewStreamingPublisher(
		StreamingPublisherConfig{
			URL:       queueURL,
			Marshaler: GobMarshaler{},
			Options: []nats.Option{
				nats.Name(clientID),
			},
			NoAcks: toNoAcks(noAcks),
		},
		pubsub.NewLoggerAdapter(logger),
	)
}

func NewSubscriber(env *moo.Environment, clientID string, noAcks []string, logger log.Logger) (pubsub.Subscriber, error) {
	if clientID == "" {
		clientID = "tpt-sub-" + time.Now().Format(time.RFC3339)
	}
	queueURL := env.Config.StringWithDefault(api.CfgPubsubNatsURL, "")
	queueGroup := env.Config.StringWithDefault(api.CfgPubsubNatsQueueGroup, "")
	subscribersCount := env.Config.IntWithDefault(api.CfgPubsubNatsSubThreads, 10)

	if queueURL == "" {
		return nil, errors.New("Nats 服务器参数不正确： URL 为空")
	}
	return NewStreamingSubscriber(
		StreamingSubscriberConfig{
			URL:        queueURL,
			QueueGroup: queueGroup,
			// DurableName:      "my-durable",
			NoAcks:           toNoAcks(noAcks),
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

func toNoAcks(noacks []string) map[string]bool {
	results := map[string]bool{}
	for _, noack := range noacks {
		results[noack] = true
	}
	return results
}
