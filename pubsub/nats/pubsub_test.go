package pubsubnats_test

import (
	"os"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/tests"
	nats "github.com/nats-io/nats.go"
	pubsubnats "github.com/runner-mei/moo/pubsub/nats"
	"github.com/stretchr/testify/require"
)

func newPubSub(t *testing.T, clientID string, queueName string) (message.Publisher, message.Subscriber) {
	logger := watermill.NewStdLogger(true, true)

	natsURL := os.Getenv("WATERMILL_TEST_NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	options := []nats.Option{
		nats.Name(clientID),
		// 	stan.NatsURL(natsURL),
		// 	nats.ConnectWait(time.Second * 15),
	}

	pub, err := pubsubnats.NewStreamingPublisher(pubsubnats.StreamingPublisherConfig{
		URL:       natsURL,
		Marshaler: pubsubnats.GobMarshaler{},
		Options:   options,
	}, logger)
	require.NoError(t, err)

	sub, err := pubsubnats.NewStreamingSubscriber(pubsubnats.StreamingSubscriberConfig{
		URL:        natsURL,
		QueueGroup: queueName,
		//DurableName:      "durable-name",
		SubscribersCount: 10,
		//AckWaitTimeout:   time.Second, // AckTiemout < 5 required for continueAfterErrors
		Unmarshaler: pubsubnats.GobMarshaler{},
		Options:     options,
	}, logger)
	require.NoError(t, err)

	return pub, sub
}

func createPubSub(t *testing.T) (message.Publisher, message.Subscriber) {
	return newPubSub(t, watermill.NewUUID(), "test-queue")
}

func createPubSubWithDurable(t *testing.T, consumerGroup string) (message.Publisher, message.Subscriber) {
	return newPubSub(t, consumerGroup, consumerGroup)
}

func TestPublishSubscribe(t *testing.T) {
	tests.TestPubSub(
		t,
		tests.Features{
			ConsumerGroups:      true,
			ExactlyOnceDelivery: false,
			GuaranteedOrder:     false,
			Persistent:          true,
		},
		createPubSub,
		createPubSubWithDurable,
	)
}
