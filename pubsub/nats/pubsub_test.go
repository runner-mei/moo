package pubsubnats_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/subscriber"
	"github.com/ThreeDotsLabs/watermill/pubsub/tests"
	nats "github.com/nats-io/nats.go"
	pubsubnats "github.com/runner-mei/moo/pubsub/nats"
	"github.com/stretchr/testify/require"
)

func newPubSub(t *testing.T, clientID string, queueName string) (message.Publisher, message.Subscriber) {
	logger := watermill.NewStdLogger(true, true)

	natsURL := os.Getenv("WATERMILL_TEST_NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:37105"
	}

	options := []nats.Option{
		nats.Name(clientID),
		// 	stan.NatsURL(natsURL),
		// 	nats.ConnectWait(time.Second * 15),
	}

	pub, err := pubsubnats.NewStreamingPublisher(pubsubnats.StreamingPublisherConfig{
		URL:             natsURL,
		Marshaler:       pubsubnats.GobMarshaler{},
		Options:         options,
		SendWaitTimeout: time.Second, // AckTiemout < 5 required for continueAfterErrors
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
	return newPubSub(t, "testclient", "test-queue")
}

func createPubSubWithDurable(t *testing.T, consumerGroup string) (message.Publisher, message.Subscriber) {
	return newPubSub(t, consumerGroup, consumerGroup)
}

func TestPublishSubscribe(t *testing.T) {
	t.Skip("todo - fix")

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

func TestNatsPubSub(t *testing.T) {
	pub, sub := createPubSub(t)

	defer func() {
		require.NoError(t, pub.Close())
		require.NoError(t, sub.Close())
	}()

	msgs, err := sub.Subscribe(context.Background(), "test")
	require.NoError(t, err)

	receivedMessages := make(chan message.Messages)
	go func() {
		received, _ := subscriber.BulkRead(msgs, 100, time.Second*10)
		receivedMessages <- received
	}()

	publishedMessages := tests.PublishSimpleMessages(t, 100, pub, "test")

	tests.AssertAllMessagesReceived(t, publishedMessages, <-receivedMessages)
}
