package moo

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/syncx"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/api"
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

func (a *Message) Equal(b *Message) bool {
	return a.CreatedAt.Equal(b.CreatedAt)
}

const (
	MessageChangeRecheck = iota
	MessageChangeCreated
	MessageChangeUpdated
	MessageChangeDeleted
)

type MessageChangeListenerFunc func(int, *Message)

type MessagePlaceholder interface {
	Set(level MessageLevel, msg string)
	Clear()
}

type placeholder struct {
	mu *sync.Mutex

	onChange MessageChangeListenerFunc
	err      Message
}

func (ph *placeholder) Set(level MessageLevel, msg string) {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if msg != "" && ph.err.Content == "" {
		ph.err.Level = level
		ph.err.Content = msg
		ph.err.CreatedAt = time.Now()

		old := ph.toMessage()
		ph.onChange(MessageChangeCreated, &old)
	} else if ph.err.Content != msg {
		if msg == "" {
			old := ph.toMessage()
			ph.err.Level = level
			ph.err.Content = msg
			ph.onChange(MessageChangeDeleted, &old)
		} else {
			ph.err.Level = level
			ph.err.Content = msg
			ph.err.CreatedAt = time.Now()
			old := ph.toMessage()
			ph.onChange(MessageChangeUpdated, &old)
		}
	}
}

func (ph *placeholder) Clear() {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	msg := ph.toMessage()
	ph.err.Content = ""
	ph.onChange(MessageChangeDeleted, &msg)
}

func (ph *placeholder) isMessage() bool {
	return ph.err.Content != ""
}

func (ph *placeholder) toMessage() Message {
	return ph.err
}

type MessageChangeListenerSetter interface {
		SetMessageChangeListener(MessageChangeListenerFunc)
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
	onChange     MessageChangeListenerFunc
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
		onChange: list.onChange,
		mu:       &list.mu,
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

	if setup, ok := provider.(MessageChangeListenerSetter); ok {
		setup.SetMessageChangeListener(list.onChange)
	}
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

	list.onChange(MessageChangeCreated, err)
}

func (list *MessageList) Remove(id string) {
	list.mu.Lock()
	defer list.mu.Unlock()

	for idx, e := range list.list {
		if e.ID == id {
			old := e // 必须先拷贝一下
			if idx < len(list.list)-1 {
				copy(list.list[idx:], list.list[idx+1:])
			}
			list.list = list.list[:len(list.list)-1]
			list.onChange(MessageChangeDeleted, &old)
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

type watcher struct {
	logger log.Logger
	list   *MessageList
	bus    *Bus
	last   []Message
}

func (w *watcher) on(action int, msg *Message) {

	var cb func()

	switch action {
	case MessageChangeRecheck:
		cb = func() {
			ctx := context.Background()
			w.check(ctx)
		}
	case MessageChangeCreated:
		if w.last != nil {
			w.last = append(w.last, *msg)
		}
		cb = func() {
			ctx := context.Background()
			err := w.bus.Emit(ctx, api.BusMessageEventCreated, msg)
			if err != nil {
				w.logger.Warn("发送 Message 变动失败", log.Error(err))
			}
		}
	case MessageChangeUpdated:
		if w.last != nil {
			for idx := range w.last {
				if w.last[idx].ID == msg.ID {
					w.last[idx].Content = msg.Content
					w.last[idx].CreatedAt = msg.CreatedAt
				}
			}
		}
		cb = func() {
			ctx := context.Background()
			err := w.bus.Emit(ctx, api.BusMessageEventUpdated, msg)
			if err != nil {
				w.logger.Warn("发送 Message 变动失败", log.Error(err))
			}
		}
	case MessageChangeDeleted:
		if w.last != nil {
			for idx := range w.last {
				if w.last[idx].ID == msg.ID {
					if idx < len(w.last)-1 {
						copy(w.last[idx:], w.last[idx+1:])
					}
					w.last = w.last[:len(w.last)-1]
				}
			}
		}

		cb = func() {
			ctx := context.Background()
			err := w.bus.Emit(ctx, api.BusMessageEventDeleted, msg)
			if err != nil {
				w.logger.Warn("发送 Message 变动失败", log.Error(err))
			}
		}
	}

	go cb()
}

func (w *watcher) check(ctx context.Context) {
	current := w.list.All()
	if w.last == nil {
		w.last = current
		return
	}
	created, updated, deleted := diffMessages(current, w.last)

	for idx := range created {
		err := w.bus.Emit(ctx, api.BusMessageEventCreated, &created[idx])
		if err != nil {
			w.logger.Warn("发送 Message 变动失败", log.Error(err))
		}
	}
	for idx := range updated {
		err := w.bus.Emit(ctx, api.BusMessageEventUpdated, &updated[idx])
		if err != nil {
			w.logger.Warn("发送 Message 变动失败", log.Error(err))
		}
	}
	for idx := range deleted {
		err := w.bus.Emit(ctx, api.BusMessageEventDeleted, &deleted[idx])
		if err != nil {
			w.logger.Warn("发送 Message 变动失败", log.Error(err))
		}
	}
	w.last = current
}

func diffMessages(current, last []Message) (created, updated, deleted []Message) {
	for idx := range current {
		a := findMessageByID(last, current[idx].ID)
		if a == nil {
			created = append(created, current[idx])
			continue
		}
		if !a.Equal(&current[idx]) {
			updated = append(updated, current[idx])
			continue
		}
	}

	for idx := range last {
		a := findMessageByID(current, last[idx].ID)
		if a == nil {
			deleted = append(deleted, last[idx])
			continue
		}
	}
	return created, updated, deleted
}

func findMessageByID(list []Message, id string) *Message {
	for idx, e := range list {
		if e.ID == id {
			return &list[idx]
		}
	}
	return nil
}

func init() {
	On(func(*Environment) Option {
		return Provide(func() *MessageList {
			return &MessageList{
				onChange: func(int, *Message) {},
			}
		})
	})

	On(func(*Environment) Option {
		return Invoke(func(lifecycle Lifecycle, httpSrv *HTTPServer, msgList *MessageList) error {
			httpSrv.FastRoute(false, "messages", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(msgList.All())
			}))
			return nil
		})
	})

	On(func(*Environment) Option {
		return Invoke(func(env *Environment,
			lifecycle Lifecycle,
			bus *Bus,
			msgList *MessageList,
			logger log.Logger) error {
			logger = logger.Named("messages.watcher")
			bus.RegisterTopics(api.BusMessageEventCreated)
			bus.RegisterTopics(api.BusMessageEventUpdated)
			bus.RegisterTopics(api.BusMessageEventDeleted)

			w := &watcher{
				logger: logger,
				list:   msgList,
				bus:    bus,
			}
			msgList.onChange = w.on
			var timer syncx.Timer

			lifecycle.Append(Hook{
				OnStart: func(context.Context) error {
					timer.Start(30*time.Second, func() bool {
						w.check(context.Background())
						return true
					})
					return nil
				},
				OnStop: func(context.Context) error {
					timer.Stop()
					return nil
				},
			})
			return nil
		})
	})
}
