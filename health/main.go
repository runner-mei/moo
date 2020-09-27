package health

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/pubsub"
)

type HealthComponents struct {
	logger  log.Logger
	source  string
	timeout int64

	components atomic.Value
}

type component struct {
	title  string
	lastAt int64
}

func (com *component) toMessage(source, id string, timeout int64) (bool, moo.Message) {
	sec := atomic.LoadInt64(&com.lastAt)
	if (sec + timeout) < time.Now().Unix() {
		return false, moo.Message{}
	}
	t := time.Unix(sec, 0)
	return true, moo.Message{
		Source:    source,
		ID:        "keeplived." + id,
		Level:     moo.MsgError,
		Content:   com.title + " 不活动了，最后活活时间 - " + t.Format(time.RFC3339),
		CreatedAt: t,
	}
}

func (hs *HealthComponents) getComponents() map[string]*component {
	o := hs.components.Load()
	if o == nil {
		return nil
	}
	comps, _ := o.(map[string]*component)
	return comps
}

func (hs *HealthComponents) activeComponent(id string, unixSec int64) *component {
	com := hs.addComponent(id, "")
	atomic.StoreInt64(&com.lastAt, unixSec)
	return com
}

func (hs *HealthComponents) addComponent(id, title string) *component {
	var comps map[string]*component
	o := hs.components.Load()
	if o != nil {
		comps, _ = o.(map[string]*component)
	}

	if comps == nil {
		if title == "" {
			title = id
		}
		value := &component{title: title, lastAt: time.Now().Unix()}
		hs.components.Store(map[string]*component{
			id: value,
		})
		return value
	}

	if value, exists := comps[id]; exists {
		if title != "" {
			value.title = title
		}
		atomic.StoreInt64(&value.lastAt, time.Now().Unix())
		return value
	}
	newCopyed := map[string]*component{}
	for key, value := range comps {
		newCopyed[key] = value
	}
	if title == "" {
		title = id
	}
	value := &component{title: title, lastAt: time.Now().Unix()}
	newCopyed[id] = value
	hs.components.Store(newCopyed)
	return value
}

func (hs *HealthComponents) removeComponent(id string) *component {
	var comps map[string]*component
	o := hs.components.Load()
	if o != nil {
		comps, _ = o.(map[string]*component)
	}
	if comps == nil {
		return nil
	}

	old, exists := comps[id]
	if !exists {
		return nil
	}
	newCopyed := map[string]*component{}
	for key, value := range comps {
		if key == id {
			continue
		}
		newCopyed[key] = value
	}
	hs.components.Store(newCopyed)
	return old
}

func (hs *HealthComponents) Get() []moo.Message {
	var messages []moo.Message
	for key, comp := range hs.getComponents() {
		ok, msg := comp.toMessage(hs.source, key, hs.timeout)
		if ok {
			messages = append(messages, msg)
		}
	}
	return messages
}

func (hs *HealthComponents) OnEvent(ctx context.Context, topicName string, value interface{}) {
	evt, ok := value.(*api.SysKeepaliveEvent)
	if !ok {
		hs.logger.Warn("不可识的 action", log.Stringer("value", log.StringerFunc(func() string {
			return fmt.Sprintf("%T", value)
		})))
		return
	}

	switch evt.Action {
	case api.SysKeepaliveEventAdd:
		hs.addComponent(evt.App, evt.Title)
	case api.SysKeepaliveEventRemove:
		hs.removeComponent(evt.App)
	case api.SysKeepaliveEventActive, "":
		hs.activeComponent(evt.App, time.Now().Unix())
	default:
		hs.logger.Warn("不可识的 action", log.String("action", evt.Action), log.String("app", evt.App), log.String("title", evt.Title))
	}
}

func DrainToBus(logger log.Logger, bus *moo.Bus, ch <-chan *pubsub.Message) {
	for msg := range ch {
		var evt api.SysKeepaliveEvent
		err := json.Unmarshal(msg.Payload, &evt)
		if err != nil {
			msg.Nack()
			logger.Warn("解析消息失败", log.Error(err))
			continue
		}
		msg.Ack()
		logger.Warn("转发消息到 bus 成功", log.Error(err))
	}
}

func init() {
	moo.On(func() moo.Option {
		return moo.Invoke(func(env *moo.Environment, lifecycle moo.Lifecycle, bus *moo.Bus, subscriber pubsub.Subscriber, msgList *moo.MessageList, logger log.Logger) error {
			logger = logger.Named("health.keeplived.commponents")
			bus.RegisterTopics(api.BusSysKeepaliveStatus)

			components := &HealthComponents{
				logger:  logger,
				source:  "health.keeplived.commponents",
				timeout: env.Config.Int64WithDefault(api.CfgHealthKeepliveTimeout, 60*5),
			}

			ch, err := subscriber.Subscribe(context.Background(), "keeplive")
			if err != nil {
				return err
			}
			go DrainToBus(logger, bus, ch)

			lifecycle.Append(moo.Hook{
				OnStart: func(context.Context) error {
					msgList.SetupProvider(components)
					bus.Register("keeplive_listener", &moo.BusHandler{
						Matcher: api.BusSysKeepaliveStatus,
						Handle:  components.OnEvent,
					})
					return nil
				},
				OnStop: func(context.Context) error {
					bus.Unregister("keeplive_listener")
					return nil
				},
			})
			return nil
		})
	})
}
