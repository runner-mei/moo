package moo

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/runner-mei/errors"
)

type MessageLevel string

const (
	MsgInfo  MessageLevel = "info"
	MsgWarn  MessageLevel = "warn"
	MsgError MessageLevel = "error"
	MsgFatal MessageLevel = "fatal"
)

type Message struct {
	ID        string       `json:"id"`
	Source    string       `json:"source"`
	Level     MessageLevel `json:"level"`
	Content   string       `json:"message"`
	CreatedAt time.Time    `json:"created_at"`
}

type MessagePlaceholder interface {
	Set(level MessageLevel, msg string)
	Clear()
}

type placeholder struct {
	mu *sync.Mutex

	err Message
}

func (ph *placeholder) Set(level MessageLevel, msg string) {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if msg != "" && ph.err.Content == "" {
		ph.err.CreatedAt = time.Now()
	}
	ph.err.Content = msg
}

func (ph *placeholder) Clear() {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	ph.err.Content = ""
}

func (ph *placeholder) isMessage() bool {
	return ph.err.Content != ""
}

func (ph *placeholder) toMessage() Message {
	return ph.err
}

type MessageProvider interface {
	Get() []Message
}

type MessageProvideFunc func() []Message

func (cb MessageProvideFunc) Get() []Message {
	return cb()
}

type MessageList struct {
	mu           sync.Mutex
	list         []Message
	providers    []MessageProvider
	placeholders []placeholder
}

func (list *MessageList) Placeholder(id, source string) MessagePlaceholder {
	if id == "" {
		panic(errors.New("id is missing"))
	}
	list.mu.Lock()
	defer list.mu.Unlock()

	for idx, ph := range list.placeholders {
		if ph.err.ID == id {
			return &list.placeholders[idx]
		}
	}

	list.placeholders = append(list.placeholders, placeholder{
		mu: &list.mu,
		err: Message{
			ID:     id,
			Source: source,
		},
	})
	return &list.placeholders[len(list.placeholders)-1]
}

func (list *MessageList) SetupProvider(provider MessageProvider) {
	list.mu.Lock()
	defer list.mu.Unlock()
	list.providers = append(list.providers, provider)
}

func (list *MessageList) Add(err *Message) {
	if err.ID == "" {
		panic(errors.New("id is missing"))
	}
	list.mu.Lock()
	defer list.mu.Unlock()

	for _, e := range list.list {
		if e.ID == err.ID {
			return
		}
	}

	list.list = append(list.list, *err)
}

func (list *MessageList) Remove(id string) {
	list.mu.Lock()
	defer list.mu.Unlock()

	for idx, e := range list.list {
		if e.ID == id {
			copy(list.list[idx:], list.list[idx+1:])
			break
		}
	}
}

func (list *MessageList) All() []Message {
	list.mu.Lock()
	defer list.mu.Unlock()

	var results = make([]Message, len(list.list))
	for idx := range list.list {
		results[idx] = list.list[idx]
	}
	for idx := range list.placeholders {
		if list.placeholders[idx].isMessage() {
			results = append(results, list.placeholders[idx].toMessage())
		}
	}
	for idx := range list.providers {
		messages := list.providers[idx].Get()
		if len(messages) > 0 {
			results = append(results, messages...)
		}
	}
	return results
}

func init() {
	On(func() Option {
		return Provide(func() *MessageList {
			return &MessageList{}
		})
	})

	On(func() Option {
		return Invoke(func(lifecycle Lifecycle, httpSrv *HTTPServer, msgList *MessageList) error {
			httpSrv.FastRoute(false, "messages", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(msgList.All())
			}))
			return nil
		})
	})
}

// func OnSystemComponentStatusEvent(ctx context.Context, topicName string, value interface{}) {

// }

// func init() {
// 			bus.RegisterTopics(api.BusSystemComponentStatus)
// 			lifecycle.Append(Hook{
// 				OnStart: func(context.Context) error {
// 					bus.Register("message_list_listener", &BusHandler{
// 						Matcher: BusSystemComponentStatus,
// 						Handle:  OnSystemComponentStatusEvent,
// 					})
// 					return nil
// 				},
// 				OnStop: func(context.Context) error {
// 					bus.Unregister("message_list_listener")
// 					return nil
// 				},
// 			})
// 		}
