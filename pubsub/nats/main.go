package pubsubnats

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/pubsub"
)

var ErrNoConnect = errors.New("connection is missing")

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
	queueGroup := env.Config.StringWithDefault(api.CfgPubsubNatsQueueGroup, "tpt_queue_sub")
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

func NewSender(env *moo.Environment, clientID string, logger log.Logger) api.Sender {
	if clientID == "" {
		clientID = "tpt-send-" + time.Now().Format(time.RFC3339)
	}
	queueURL := env.Config.StringWithDefault(api.CfgPubsubNatsURL, "")
	return &sender{
		logger:    logger,
		marshaler: GobMarshaler{},
		connurl:   queueURL,
		options: []nats.Option{
			nats.Name(clientID),
		},
	}
}

type sender struct {
	logger    log.Logger
	marshaler Marshaler
	connurl   string
	options   []nats.Option

	connecting int32
	lock       sync.Mutex
	conn       atomic.Value
}

func (s *sender) get() *nats.Conn {
	o := s.conn.Load()
	if o == nil {
		return nil
	}
	conn, _ := o.(*nats.Conn)
	return conn
}

func (s *sender) startConnect() {
	if atomic.CompareAndSwapInt32(&s.connecting, 0, 1) {
		go func() {
			defer atomic.StoreInt32(&s.connecting, 0)
			s.connect()
		}()
	}
}

func (s *sender) connect() (conn *nats.Conn, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	conn = s.get()
	if conn != nil {
		return conn, nil
	}

	conn, err = nats.Connect(s.connurl, s.options...)
	if err != nil {
		return nil, err
	}
	s.conn.Store(conn)
	return conn, nil
}

func (s *sender) Send(ctx context.Context, topic, source string, payload interface{}) error {
	conn := s.get()
	if conn == nil {
		s.startConnect()
		return ErrNoConnect
	}

	msg := pubsub.NewMessage(source, payload)
	b, err := s.marshaler.Marshal(topic, msg)
	if err != nil {
		return err
	}

	return conn.Publish(topic, b)
}
