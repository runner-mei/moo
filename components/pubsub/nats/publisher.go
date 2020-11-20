package pubsubnats

import (
	"bytes"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	nats "github.com/nats-io/nats.go"
	"github.com/runner-mei/errors"
	compnats "github.com/runner-mei/moo/components/nats"
)

var ErrErrorResponse = errors.New("server responded with error status")

type StreamingPublisherConfig struct {
	Factory    *compnats.Factory
	ClientName string
	IsDefault  bool

	// StanOptions are custom options for a connection.
	Options []nats.Option

	// Marshaler is marshaler used to marshal messages to nats format.
	Marshaler Marshaler

	NoAcks          map[string]bool
	SendWaitTimeout time.Duration
}

type StreamingPublisherPublishConfig struct {
	// Marshaler is marshaler used to marshal messages to nats format.
	Marshaler Marshaler

	NoAcks          map[string]bool
	SendWaitTimeout time.Duration
}

func (c StreamingPublisherConfig) Validate() error {
	if c.Marshaler == nil {
		return errors.New("StreamingPublisherConfig.Marshaler is missing")
	}

	return nil
}

func (c StreamingPublisherConfig) GetStreamingPublisherPublishConfig() StreamingPublisherPublishConfig {
	return StreamingPublisherPublishConfig{
		Marshaler:       c.Marshaler,
		NoAcks:          c.NoAcks,
		SendWaitTimeout: c.SendWaitTimeout,
	}
}

type StreamingPublisher struct {
	conn   *nats.Conn
	config StreamingPublisherPublishConfig
	logger watermill.LoggerAdapter
}

// NewStreamingPublisher creates a new StreamingPublisher.
//
// When using custom NATS hostname, you should pass it by options StreamingPublisherConfig.StanOptions:
//		// ...
//		StanOptions: []nats.Option{
//			nats.NatsURL("nats://your-nats-hostname:4222"),
//		}
//		// ...
func NewStreamingPublisher(config StreamingPublisherConfig, logger watermill.LoggerAdapter) (*StreamingPublisher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var conn *nats.Conn
	var err error
	if config.IsDefault {
		conn, err = config.Factory.Default()
	} else {
		conn, err = config.Factory.Create(nil, config.ClientName, config.Options...)
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to NATS")
	}

	return NewStreamingPublisherWithConn(conn, config.GetStreamingPublisherPublishConfig(), logger)
}

func NewStreamingPublisherWithConn(conn *nats.Conn, config StreamingPublisherPublishConfig, logger watermill.LoggerAdapter) (*StreamingPublisher, error) {
	if logger == nil {
		logger = watermill.NopLogger{}
	}

	return &StreamingPublisher{
		conn:   conn,
		config: config,
		logger: logger,
	}, nil
}

// Publish publishes message to NATS.
//
// Publish will not return until an ack has been received from NATS Streaming.
// When one of messages delivery fails - function is interrupted.
func (p StreamingPublisher) Publish(topic string, messages ...*message.Message) error {
	noAck := false
	if p.config.NoAcks != nil {
		noAck = p.config.NoAcks[topic]
	}

	for _, msg := range messages {
		messageFields := watermill.LogFields{
			"message_uuid": msg.UUID,
			"topic_name":   topic,
		}

		p.logger.Trace("Publishing message", messageFields)

		b, err := p.config.Marshaler.Marshal(topic, msg)
		if err != nil {
			return err
		}

		if noAck {
			if err := p.conn.Publish(topic, b); err != nil {
				return errors.Wrap(err, "sending message failed")
			}
		} else {
			resp, err := p.conn.Request(topic, b, p.config.SendWaitTimeout)
			if err != nil {
				return errors.Wrap(err, "sending message failed")
			}

			if !bytes.Equal(resp.Data, []byte("OK")) {
				return errors.Wrap(ErrErrorResponse, string(resp.Data))
			}
		}
		p.logger.Trace("Message published", messageFields)
	}

	return nil
}

func (p StreamingPublisher) Close() error {
	p.logger.Trace("Closing publisher", nil)
	defer p.logger.Trace("StreamingPublisher closed", nil)

	p.conn.Close()

	return nil
}
