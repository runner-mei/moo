package pubsubnats

// import (
// 	"github.com/runner-mei/goutils/urlutil"
// 	"github.com/runner-mei/log"
// 	"github.com/runner-mei/moo"
// 	"github.com/runner-mei/moo/pubsub"

//   nats "github.com/nats-io/nats.go"
//   "github.com/ThreeDotsLabs/watermill"
//   "github.com/ThreeDotsLabs/watermill/message"
// )

// func NewPublisher(env *moo.Environment, clientID string, logger log.Logger) (pubsub.Publisher, error) {
//   if clientID == "" {
//     clientID = "tpt-pub-" + time.Now().Format(time.RFC3339)
//   }
//   queueURL := env.Config.StringWithDefault(api.CfgQueueURL, "")
//   return nats.NewStreamingPublisher(
//       nats.StreamingPublisherConfig{
//           ClusterID: env.Config.StringWithDefault(api.CfgQueueCluster, "tpt-cluster"),
//           ClientID:  clientID,
//           StanOptions: []nats.Option{
//               stan.NatsURL( queueURL ),
//           },
//           Marshaler: nats.GobMarshaler{},
//       },
//       pubsub.NewLoggerAdapter(logger),
//   )
// }

// func NewSubscriber(env *moo.Environment, clientID string, logger log.Logger) (pubsub.Subscriber, error) {
//   if clientID == "" {
//     clientID = "tpt-sub-" + time.Now().Format(time.RFC3339)
//   }
//   queueURL := env.Config.StringWithDefault(api.CfgQueueURL, "")

// return nats.NewStreamingSubscriber(
//         nats.StreamingSubscriberConfig{
//             ClusterID:       env.Config.StringWithDefault(api.CfgQueueCluster, "tpt-cluster"),
//             ClientID:         clientID,
//             QueueGroup:       "example",
//             DurableName:      "my-durable",
//             SubscribersCount: 4, // how many goroutines should consume messages
//            CloseTimeout:     time.Minute,
//             AckWaitTimeout:   time.Second * 30,
//             StanOptions: []stan.Option{
//                 stan.NatsURL(queueURL),
//             },
//             Unmarshaler: nats.GobMarshaler{},
//         },
//       pubsub.NewLoggerAdapter(logger),
//     )
// }
