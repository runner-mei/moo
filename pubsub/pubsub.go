package pubsub

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ThreeDotsLabs/watermill"
	httpmill "github.com/ThreeDotsLabs/watermill-http/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/urlutil"
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
	return httpmill.NewPublisher(config, watermill.NopLogger{})
}

func NewHTTPSubscriber(urlstr string, logger log.Logger) (http.Handler, Subscriber, error) {
	var config = httpmill.SubscriberConfig{
		UnmarshalMessageFunc: httpmill.UnmarshalMessageFunc(func(requrl string, req *http.Request) (*message.Message, error) {
			return httpmill.DefaultUnmarshalMessageFunc(requrl, req)
		}),
	}
	subscriber, err := httpmill.NewSubscriber(":0", config, watermill.NopLogger{})
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
