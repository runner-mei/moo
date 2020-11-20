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
	compnats "github.com/runner-mei/moo/components/nats"
	"github.com/runner-mei/moo/components/pubsub"
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

func NewPublisher(env *moo.Environment, factory *compnats.Factory, clientID string, noAcks []string, logger log.Logger) (pubsub.Publisher, error) {
	if clientID == "" {
		clientID = "tpt-pub-" + time.Now().Format(time.RFC3339)
	}
	return NewStreamingPublisher(
		StreamingPublisherConfig{
			Factory:    factory,
			ClientName: clientID,
			IsDefault:  env.Config.BoolWithDefault(api.CfgPubsubNatsUseDefaultConn, false),
			Marshaler:  GobMarshaler{},
			Options: []nats.Option{
				nats.Name(clientID),
			},
			NoAcks: toNoAcks(noAcks),
		},
		pubsub.NewLoggerAdapter(logger),
	)
}

func NewSubscriber(env *moo.Environment, factory *compnats.Factory, clientID string, noAcks []string, logger log.Logger) (pubsub.Subscriber, error) {
	if clientID == "" {
		clientID = "tpt-sub-" + time.Now().Format(time.RFC3339)
	}
	queueGroup := env.Config.StringWithDefault(api.CfgPubsubNatsQueueGroup, "tpt_queue_sub")
	subscribersCount := env.Config.IntWithDefault(api.CfgPubsubNatsSubThreads, 10)
	return NewStreamingSubscriber(
		StreamingSubscriberConfig{
			Factory:    factory,
			ClientName: clientID,
			IsDefault:  env.Config.BoolWithDefault(api.CfgPubsubNatsUseDefaultConn, false),
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

func NewSender(env *moo.Environment, factory *compnats.Factory, useDefaultConn bool, clientID string, logger log.Logger) *sender {
	if clientID == "" {
		clientID = "tpt-send-" + time.Now().Format(time.RFC3339)
	}
	return &sender{
		logger:         logger,
		marshaler:      GobMarshaler{},
		factory:        factory,
		useDefaultConn: useDefaultConn,
		clientID:       clientID,
	}
}

var _ api.Sender = &sender{}

type sender struct {
	logger         log.Logger
	marshaler      Marshaler
	useDefaultConn bool
	factory        *compnats.Factory
	clientID       string

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

func (s *sender) StartConnect() {
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
	if s.useDefaultConn {
		conn, err = s.factory.Default()
	} else {
		conn, err = s.factory.Create(s.logger, s.clientID)
	}
	if err != nil {
		return nil, err
	}
	s.conn.Store(conn)
	return conn, nil
}

func (s *sender) Send(ctx context.Context, topic, source string, payload interface{}) error {
	conn := s.get()
	if conn == nil {
		s.StartConnect()
		return ErrNoConnect
	}

	msg := pubsub.NewMessage(source, payload)
	b, err := s.marshaler.Marshal(topic, msg)
	if err != nil {
		return err
	}

	return conn.Publish(topic, b)
}
