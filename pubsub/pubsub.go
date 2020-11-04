package pubsub

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ThreeDotsLabs/watermill"
	httpmill "github.com/ThreeDotsLabs/watermill-http/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
)

type Message = message.Message
type Publisher = message.Publisher
type Subscriber = message.Subscriber
type HandleErrorFunc = httpmill.HandleErrorFunc

func NewMessage(source string, payload interface{}) *Message {
	bs, err := json.Marshal(payload)
	if err != nil {
		errors.Panic(errors.Wrap(err, "create '"+source+"' error"))
	}
	return message.NewMessage(watermill.NewUUID(), message.Payload(bs))
}

func NewHTTPPublisher(urlstr string, logger log.Logger) (Publisher, error) {
	var config = httpmill.PublisherConfig{
		MarshalMessageFunc: httpmill.MarshalMessageFunc(func(requrl string, msg *message.Message) (*http.Request, error) {
			return httpmill.DefaultMarshalMessageFunc(urlutil.Join(urlstr, requrl), msg)
		}),
		Client:                            httputil.InsecureHttpClent,
		DoNotLogResponseBodyOnServerError: true,
	}
	return httpmill.NewPublisher(config, NewLoggerAdapter(logger))
}

func NewHTTPSubscriber(urlstr string, logger log.Logger) (http.Handler, Subscriber, error) {
	var config = httpmill.SubscriberConfig{
		UnmarshalMessageFunc: httpmill.UnmarshalMessageFunc(func(requrl string, req *http.Request) (*message.Message, error) {
			return httpmill.DefaultUnmarshalMessageFunc(requrl, req)
		}),
	}
	subscriber, err := httpmill.NewSubscriber(":0", config, NewLoggerAdapter(logger))
	if err != nil {
		return nil, nil, err
	}
	return subscriber, subscriber, nil
}

// func NewSSE(upstreamSubscriber Subscriber, errorHandler HandleErrorFunc, logger watermill.LoggerAdapter) (http.Handler, error) {
// 	router, err := httpmill.NewSSERouter(httpmill.SSERouterConfig{
// 		UpstreamSubscriber: upstreamSubscriber,
// 		ErrorHandler:       errorHandler,
// 	}, watermill.NopLogger{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return http.HandlerFunc(router.), nil
// }

func DrainToBus(ctx context.Context, logger log.Logger, topicName string, bus *moo.Bus, ch <-chan *Message, convert func(context.Context, *Message) (interface{}, error)) {
	for msg := range ch {
		evt, err := convert(ctx, msg)
		if err != nil {
			msg.Nack()
			logger.Warn("解析消息失败", log.Error(err))
			continue
		}

		err = bus.Emit(ctx, topicName, evt)
		if err != nil {
			msg.Nack()
			logger.Warn("解析消息失败", log.Error(err))
			continue
		}

		msg.Ack()
		logger.Warn("转发消息到 bus 成功", log.Error(err))
	}
}

func NewLoggerAdapter(logger log.Logger) watermill.LoggerAdapter {
	return loggerAdapter{logger}
}

type loggerAdapter struct {
	logger log.Logger
}

func (a loggerAdapter) toFields(fields watermill.LogFields) []log.Field {
	var innerfields = make([]log.Field, 0, len(fields))
	for key, value := range fields {
		innerfields = append(innerfields, log.Any(key, value))
	}
	return innerfields
}

func (a loggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	var innerfields = a.toFields(fields)
	innerfields = append(innerfields, log.Error(err))
	a.logger.Error(msg, innerfields...)
}

func (a loggerAdapter) Info(msg string, fields watermill.LogFields) {
	a.logger.Info(msg, a.toFields(fields)...)
}
func (a loggerAdapter) Debug(msg string, fields watermill.LogFields) {
	a.logger.Debug(msg, a.toFields(fields)...)
}
func (a loggerAdapter) Trace(msg string, fields watermill.LogFields) {
	a.logger.Debug(msg, a.toFields(fields)...)
}
func (a loggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return loggerAdapter{a.logger.With(a.toFields(fields)...)}
}
